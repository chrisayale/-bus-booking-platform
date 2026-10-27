package whatsapp

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/bus-booking/BE-bus-booking-platform/internal/utils"
	"github.com/jmoiron/sqlx"
	qrcodegen "github.com/skip2/go-qrcode"
	"github.com/tuotoo/qrcode"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

const welcomeMenu = `Bienvenue sur GUV (Guichet unique des voyages) 🚌

1️⃣ Recherche directe de voyages
2️⃣ Recherche par agence
3️⃣ Créer une agence
4️⃣ Vérification / Guichet
5️⃣ Aide & Assistance

Répondez avec le numéro de votre choix.`

// Step constants for payment flow (these are assumed to be defined elsewhere like other steps)
// If they cause undefined errors, they may need to be defined in the appropriate constants file
// For now, we'll reference them and hope they're defined elsewhere like StepMenu, etc.

func init() {
	_ = fmt.Sprintf("%T", (*sqlx.DB)(nil))
}

func GetRealPhoneNumber(client *whatsmeow.Client, jid types.JID) string {
	if jid.Server == "s.whatsapp.net" {
		return jid.User
	}
	if jid.Server == "lid" && client != nil && client.Store != nil {
		pnJID, err := client.Store.LIDs.GetPNForLID(context.Background(), jid)
		if err == nil && !pnJID.IsEmpty() && pnJID.Server == "s.whatsapp.net" {
			return pnJID.User
		}
	}
	return jid.User
}

// BookingDetailsResult holds the result of the booking lookup query for controller payment validation
type BookingDetailsResult struct {
	ID                 string    `db:"id"`
	BookingID          string    `db:"booking_id"`
	BookingReference   string    `db:"booking_reference"`
	TotalAmount        float64   `db:"total_amount"`
	Status             string    `db:"status"`
	PaymentStatus      string    `db:"payment_status"`
	PaymentMethod      string    `db:"payment_method"`
	CreatedAt          time.Time `db:"created_at"`
	UpdatedAt          time.Time `db:"updated_at"`
	JourneyID          string    `db:"journey_id"`
	DepartureDate      string    `db:"departure_date"`
	DepartureTime      string    `db:"departure_time"`
	ArrivalDate        string    `db:"arrival_date"`
	ArrivalTime        string    `db:"arrival_time"`
	RouteID            string    `db:"route_id"`
	Origin             string    `db:"origin"`
	Destination        string    `db:"destination"`
	BasePrice          float64   `db:"base_price"`
	OrganizationID     string    `db:"organization_id"`
	OrganizationName   string    `db:"organization_name"`
	TicketIDs          string    `db:"ticket_ids"`
	PassengerNames     string    `db:"passenger_names"`
	FirstPassengerName string    `db:"first_passenger_name"`
	PassengerPhones    string    `db:"passenger_phones"`
	SeatNumbers        string    `db:"seat_numbers"`
	PassengerName      string    `db:"passenger_name"`
	PassengerPhone     string    `db:"passenger_phone"`
	TicketID           string    `db:"ticket_id"`
	SeatNumber         string    `db:"seat_number"`
	Price              float64   `db:"price"`
}

type controllerPINResult struct {
	UserID     string `db:"user_id"`
	AgentName  string `db:"agent_name"`
	AgencyName string `db:"agency_name"`
}

// verifyControllerPINWithSender verifies a controller/agent PIN using the real MSISDN (sender_pn)
// Primary strategy: Real phone number (sender_pn) + PIN match
// Fallback strategies: PIN-only, LID-based matching for edge cases
func verifyControllerPINWithSender(sender types.JID, pin string) *controllerPINResult {
	var res controllerPINResult

	// Get the real MSISDN (sender_pn) from the sender JID
	// This resolves LID (e.g., 175247701618854@lid) to real phone (e.g., 243974645174@s.whatsapp.net)
	realPhone := GetRealPhoneNumber(Client, sender)
	rawUser := sender.User
	normalizedRealPhone := strings.TrimPrefix(realPhone, "+")

	// Log the real phone vs LID session for debugging
	log.Printf("[PIN_AUTH] Real Phone: %s, LID Session: %s, PIN: %s", normalizedRealPhone, rawUser, pin)

	// STRATEGY 1: Real phone (sender_pn) + PIN match (PRIMARY)
	// Uses the actual phone number from WhatsApp's LID resolution
	log.Printf("[PIN_AUTH] Strategy 1: Real phone + PIN - SELECT ... WHERE (REPLACE(phone,'+','')=$1 OR RIGHT(REGEXP_REPLACE(phone,'\\D','','g'),9)=RIGHT($1,9)) AND pin=$2 AND is_active=true AND role IN ('controller','agent')", normalizedRealPhone, pin)
	err := DB.Get(&res, `
		SELECT
			u.id AS user_id,
			COALESCE(NULLIF(u.full_name, ''), COALESCE(NULLIF(u.email, ''), 'Agent')) AS agent_name,
			COALESCE(NULLIF(o.name, ''), 'Aucune agence assignée') AS agency_name
		FROM users u
		LEFT JOIN organizations o ON u.organization_id = o.id
		WHERE (
			REPLACE(u.phone, '+', '') = $1
			OR RIGHT(REGEXP_REPLACE(u.phone, '\D', '', 'g'), 9) = RIGHT($1, 9)
		)
		AND u.pin = $2
		AND u.is_active = true
		AND u.role IN ('controller', 'agent')
	`, normalizedRealPhone, pin)
	if err == nil {
		log.Printf("[PIN_AUTH] ✓ Strategy 1 SUCCESS: Controller/agent authenticated via real phone + PIN (realPhone: %s, sender: %s, user_id: %s, agent: %s, agency: %s)", normalizedRealPhone, rawUser, res.UserID, res.AgentName, res.AgencyName)
		return &res
	}
	log.Printf("[PIN_AUTH] ✗ Strategy 1 FAILED: %v", err)

	// STRATEGY 2: PIN-only match (FALLBACK - does not depend on phone number)
	// Allows controllers to authenticate from any device/number using just their PIN
	log.Printf("[PIN_AUTH] Strategy 2: PIN-only query - SELECT ... WHERE pin='%s' AND is_active=true AND role IN ('controller','agent')", pin)
	err = DB.Get(&res, `
		SELECT
			u.id AS user_id,
			COALESCE(NULLIF(u.full_name, ''), COALESCE(NULLIF(u.email, ''), 'Agent')) AS agent_name,
			COALESCE(NULLIF(o.name, ''), 'Aucune agence assignée') AS agency_name
		FROM users u
		LEFT JOIN organizations o ON u.organization_id = o.id
		WHERE u.pin = $1 AND u.is_active = true
		  AND u.role IN ('controller', 'agent')
	`, pin)
	if err == nil {
		log.Printf("[PIN_AUTH] ✓ Strategy 2 SUCCESS: Controller/agent authenticated via PIN-only match (sender: %s, user_id: %s, agent: %s, agency: %s)", rawUser, res.UserID, res.AgentName, res.AgencyName)
		return &res
	}
	log.Printf("[PIN_AUTH] ✗ Strategy 2 FAILED: %v", err)

	// STRATEGY 3: LID-based exact match (fallback for edge cases)
	// Uses the raw sender LID user part for matching
	lidNormalized := strings.TrimPrefix(rawUser, "+")
	lidDigitsOnly := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, lidNormalized)

	if len(lidDigitsOnly) >= 9 {
		log.Printf("[PIN_AUTH] Strategy 3: LID exact match - SELECT ... WHERE pin='%s' AND REPLACE(phone,'+','')='%s'", pin, lidDigitsOnly)
		err = DB.Get(&res, `
			SELECT
				u.id AS user_id,
				COALESCE(NULLIF(u.full_name, ''), COALESCE(NULLIF(u.email, ''), 'Agent')) AS agent_name,
				COALESCE(NULLIF(o.name, ''), 'Aucune agence assignée') AS agency_name
			FROM users u
			LEFT JOIN organizations o ON u.organization_id = o.id
			WHERE u.pin = $1 AND u.is_active = true
			  AND REPLACE(u.phone, '+', '') = $2
			  AND u.role IN ('controller', 'agent')
		`, pin, lidDigitsOnly)
		if err == nil {
			log.Printf("[PIN_AUTH] ✓ Strategy 3 SUCCESS: Controller/agent authenticated via LID exact match (sender: %s, user_id: %s)", rawUser, res.UserID)
			return &res
		}
		log.Printf("[PIN_AUTH] ✗ Strategy 3 FAILED: %v", err)
	}

	// STRATEGY 4: LID suffix match (last 9 digits)
	if len(lidDigitsOnly) >= 9 {
		suffix := lidDigitsOnly[len(lidDigitsOnly)-9:]
		log.Printf("[PIN_AUTH] Strategy 4: LID suffix match (last 9 digits=%s) - SELECT ... WHERE pin='%s' AND REPLACE(phone,'+','') LIKE '%%%s'", suffix, pin, suffix)
		err = DB.Get(&res, `
			SELECT
				u.id AS user_id,
				COALESCE(NULLIF(u.full_name, ''), COALESCE(NULLIF(u.email, ''), 'Agent')) AS agent_name,
				COALESCE(NULLIF(o.name, ''), 'Aucune agence assignée') AS agency_name
			FROM users u
			LEFT JOIN organizations o ON u.organization_id = o.id
			WHERE u.pin = $1 AND u.is_active = true
			  AND REPLACE(u.phone, '+', '') LIKE '%' || $2
			  AND u.role IN ('controller', 'agent')
		`, pin, suffix)
		if err == nil {
			log.Printf("[PIN_AUTH] ✓ Strategy 4 SUCCESS: Controller/agent authenticated via LID suffix match (sender: %s, user_id: %s)", rawUser, res.UserID)
			return &res
		}
		log.Printf("[PIN_AUTH] ✗ Strategy 4 FAILED: %v", err)
	}

	log.Printf("[PIN_AUTH] All strategies failed for realPhone=%s, sender=%s, pin=%s", normalizedRealPhone, rawUser, pin)
	return nil
}

// Legacy function kept for compatibility
func verifyControllerPIN(phone string, pin string) *controllerPINResult {
	var res controllerPINResult
	normalizedPhone := strings.TrimPrefix(phone, "+")

	log.Printf("[PIN_AUTH_LEGACY] Starting PIN verification for phone=%s, pin=%s", phone, pin)

	// STRATEGY 1: PIN-only match (PRIMARY)
	log.Printf("[PIN_AUTH_LEGACY] Strategy 1: PIN-only query - SELECT ... WHERE pin='%s' AND is_active=true AND role IN ('controller','agent')", pin)
	err := DB.Get(&res, `
		SELECT
			u.id AS user_id,
			COALESCE(NULLIF(u.full_name, ''), COALESCE(NULLIF(u.email, ''), 'Agent')) AS agent_name,
			COALESCE(NULLIF(o.name, ''), 'Aucune agence assignée') AS agency_name
		FROM users u
		LEFT JOIN organizations o ON u.organization_id = o.id
		WHERE u.pin = $1 AND u.is_active = true
		  AND u.role IN ('controller', 'agent')
	`, pin)
	if err == nil {
		log.Printf("[PIN_AUTH_LEGACY] ✓ Strategy 1 SUCCESS: Controller/agent authenticated via PIN-only match (phone: %s, user_id: %s)", phone, res.UserID)
		return &res
	}
	log.Printf("[PIN_AUTH_LEGACY] ✗ Strategy 1 FAILED: %v", err)

	// STRATEGY 2: Exact phone + PIN match
	log.Printf("[PIN_AUTH_LEGACY] Strategy 2: Exact phone match - SELECT ... WHERE pin='%s' AND REPLACE(phone,'+','')='%s'", pin, normalizedPhone)
	err = DB.Get(&res, `
		SELECT
			u.id AS user_id,
			COALESCE(NULLIF(u.full_name, ''), COALESCE(NULLIF(u.email, ''), 'Agent')) AS agent_name,
			COALESCE(NULLIF(o.name, ''), 'Aucune agence assignée') AS agency_name
		FROM users u
		LEFT JOIN organizations o ON u.organization_id = o.id
		WHERE u.pin = $1 AND u.is_active = true
		  AND REPLACE(u.phone, '+', '') = $2
		  AND u.role IN ('controller', 'agent')
	`, pin, normalizedPhone)
	if err == nil {
		log.Printf("[PIN_AUTH_LEGACY] ✓ Strategy 2 SUCCESS: Controller/agent authenticated via exact phone match (phone: %s, user_id: %s)", phone, res.UserID)
		return &res
	}
	log.Printf("[PIN_AUTH_LEGACY] ✗ Strategy 2 FAILED: %v", err)

	log.Printf("[PIN_AUTH_LEGACY] All strategies failed for phone=%s, pin=%s", phone, pin)
	return nil
}

// verifyControllerPINByPin verifies a controller PIN with phone number check (with fallback)
func verifyControllerPINByPin(phone string, pin string) *controllerPINResult {
	var res controllerPINResult
	// Normalize phone number (remove leading + if present)
	normalizedPhone := strings.TrimPrefix(phone, "+")

	log.Printf("[PIN_AUTH_BYPIN] Starting PIN verification for phone=%s, pin=%s", phone, pin)

	// STRATEGY 1: PIN-only match (PRIMARY)
	log.Printf("[PIN_AUTH_BYPIN] Strategy 1: PIN-only query - SELECT ... WHERE pin='%s' AND is_active=true AND role IN ('controller','agent')", pin)
	err := DB.Get(&res, `
		SELECT
			u.id AS user_id,
			COALESCE(NULLIF(u.full_name, ''), COALESCE(NULLIF(u.email, ''), 'Agent')) AS agent_name,
			COALESCE(NULLIF(o.name, ''), 'Aucune agence assignée') AS agency_name
		FROM users u
		LEFT JOIN organizations o ON u.organization_id = o.id
		WHERE u.pin = $1 AND u.is_active = true
		  AND u.role IN ('controller', 'agent')
	`, pin)
	if err == nil {
		log.Printf("[PIN_AUTH_BYPIN] ✓ Strategy 1 SUCCESS: Controller/agent authenticated via PIN-only match (phone: %s, user_id: %s)", phone, res.UserID)
		return &res
	}
	log.Printf("[PIN_AUTH_BYPIN] ✗ Strategy 1 FAILED: %v", err)

	// STRATEGY 2: Exact phone + PIN match
	log.Printf("[PIN_AUTH_BYPIN] Strategy 2: Exact phone match - SELECT ... WHERE pin='%s' AND REPLACE(phone,'+','')='%s'", pin, normalizedPhone)
	err = DB.Get(&res, `
		SELECT
			u.id AS user_id,
			COALESCE(NULLIF(u.full_name, ''), COALESCE(NULLIF(u.email, ''), 'Agent')) AS agent_name,
			COALESCE(NULLIF(o.name, ''), 'Aucune agence assignée') AS agency_name
		FROM users u
		LEFT JOIN organizations o ON u.organization_id = o.id
		WHERE u.pin = $1 AND u.is_active = true
		  AND REPLACE(u.phone, '+', '') = $2
		  AND u.role IN ('controller', 'agent')
	`, pin, normalizedPhone)
	if err == nil {
		log.Printf("[PIN_AUTH_BYPIN] ✓ Strategy 2 SUCCESS: Controller/agent authenticated via exact phone match (phone: %s, user_id: %s)", phone, res.UserID)
		return &res
	}
	log.Printf("[PIN_AUTH_BYPIN] ✗ Strategy 2 FAILED: %v", err)

	log.Printf("[PIN_AUTH_BYPIN] All strategies failed for phone=%s, pin=%s", phone, pin)
	return nil
}

func formatTicketDetail(ticket *CreatedTicket) string {
	var b strings.Builder
	b.WriteString("🎫 *Détails du billet* :\n")
	b.WriteString(fmt.Sprintf("📌 *Référence* : %s\n", valueOr(ticket.BookingReference, "N/A")))
	b.WriteString(fmt.Sprintf("👤 *Passager* : %s\n", valueOr(ticket.PassengerName, "N/A")))
	b.WriteString(fmt.Sprintf("📱 *Téléphone* : %s\n", valueOr(ticket.PassengerPhone, "N/A")))
	b.WriteString(fmt.Sprintf("🏢 *Agence* : %s\n", valueOr(ticket.AgencyName, "N/A")))
	b.WriteString(fmt.Sprintf("🚌 *Trajet* : %s ➔ %s\n", valueOr(ticket.DepartureCity, "N/A"), valueOr(ticket.ArrivalCity, "N/A")))
	b.WriteString(fmt.Sprintf("📅 *Départ* : %s\n", valueOr(formatDateTime(ticket.Date, ticket.DepartureTime), "N/A")))
	b.WriteString(fmt.Sprintf("🏁 *Arrivée* : %s\n", valueOr(formatDateTime(ticket.ArrivalDate, ticket.ArrivalTime), "N/A")))
	b.WriteString(fmt.Sprintf("💺 *Siège* : N° %s\n", valueOr(ticket.SeatNumber, "N/A")))
	b.WriteString(fmt.Sprintf("💰 *Prix* : %.2f USD\n", ticket.Price))
	b.WriteString(fmt.Sprintf("💳 *Paiement* : Espèces\n"))
	b.WriteString(fmt.Sprintf("📊 *Statut* : %s\n", valueOr(ticket.Status, "N/A")))
	b.WriteString("\n")
	b.WriteString("📄 *Télécharger le billet PDF* :\n")
	if ticket.DownloadToken != "" {
		b.WriteString(fmt.Sprintf("https://%s/tickets/download/%s\n", AppURL, ticket.DownloadToken))
	} else {
		b.WriteString("Non disponible\n")
	}
	b.WriteString("\nTapez *menu* pour revenir au menu principal.")
	return b.String()
}

// getPromptForStep returns the appropriate prompt for the current step of the session.
func getPromptForStep(s *Session) string {
	switch s.Step {
	case StepMenu:
		return welcomeMenu
	case StepSearchOrigin:
		return "🏁 Lieu de départ ?"
	case StepSearchDestination:
		return "🎯 Lieu de destination ?"
	case StepSearchDateChoice:
		return "📅 Quelle date ?\n\n1️⃣ Aujourd'hui\n2️⃣ Saisir une date\n\nRépondez avec le numéro de votre choix."
	case StepSearchDate:
		return "📅 Entrez la date du voyage (ex : 13/09/2026 ou 13 septembre 2026)."
	case StepSearchResults:
		// This step expects a dynamic prompt based on s.Results, but we return a generic one.
		// The normal flow will override this by building the prompt before calling getPromptForStep.
		if len(s.Results) > 0 {
			return formatPaginatedResults(s)
		}
		return "🔢 Tapez le numéro du voyage choisi."
	case StepSeatsCount:
		return "💺 Combien de sièges voulez-vous réserver ?"
	case StepPassengerName:
		if s.CurrentPassengerIndex < s.SeatsNeeded {
			return fmt.Sprintf("👤 Passager %d/%d — Nom complet ?", s.CurrentPassengerIndex+1, s.SeatsNeeded)
		}
		return "👤 Nom complet du passager ?" // fallback
	case StepPassengerPhone:
		return "📱 Numéro de téléphone de ce passager (ou tapez * pour passer) ?"
	case StepConfirm:
		// Build summary without duplicating the confirmation question
		summary := buildConfirmSummary(s)
		// Remove any trailing confirmation question if it exists in the summary
		summary = strings.TrimSuffix(summary, "\n\nConfirmez-vous la réservation ? (oui/non)")
		summary = strings.TrimSuffix(summary, "\n\n✅ Confirmez-vous la réservation ? (oui/non)")
		return summary + "\n\n✅ Confirmez-vous la réservation ? (oui/non)"
	case StepPaymentMethodChoice:
		return "💳 Comment voulez-vous payer ?\n\n1️⃣ Payer en espèces\n2️⃣ Mobile Money (Airtel, M-Pesa, Orange, AfriMoney)\n3️⃣ Payer par carte (Visa, Mastercard)\n\nRépondez avec le numéro de votre choix."
	case StepCurrencyChoice:
		return "💰 Quelle devise souhaitez-vous utiliser pour le paiement ?\n\n1️⃣ USD (Dollars)\n2️⃣ CDF (Franc congolais)\n\nRépondez avec le numéro de votre choix."
	case StepPhonePaymentChoice:
		return "📱 Choisissez le numéro de téléphone pour Mobile Money :\n\n1️⃣ Utiliser mon numéro WhatsApp\n2️⃣ Saisir un autre numéro\n\nRépondez avec le numéro de votre choix."
	case StepFlexPayPhoneInput:
		return "📱 Entrez le numéro de téléphone pour Mobile Money (format : 243XXXXXXXXX) :"
	case StepPaymentProcessing:
		// Compute amount and phone number for display
		baseAmountUSD := s.SelectedJourney.Price * float64(len(s.Passengers))
		var totalAmount float64
		var paymentNumber string
		if s.Currency == "CDF" {
			rate, err := utils.GetCurrentExchangeRate(DB)
			if err != nil {
				log.Printf("Error getting exchange rate: %v, using default 2850", err)
				rate = 2850
			}
			totalAmount = baseAmountUSD * rate
		} else {
			totalAmount = baseAmountUSD
		}
		if s.PaymentPhone != "" {
			paymentNumber = s.PaymentPhone
		} else {
			// Use WhatsApp number
			paymentNumber = GetRealPhoneNumber(Client, types.JID{User: s.UserPhone, Server: "s.whatsapp.net"})
			if paymentNumber == "" {
				paymentNumber = s.UserPhone // fallback
			}
		}
		return fmt.Sprintf("💳 CONFIRMATION DU PAIEMENT\n\n• Montant : %.2f %s\n• Numéro de paiement : %s\n\nVoulez-vous valider le paiement ?\n\n1️⃣ Oui (Lancer le paiement)\n2️⃣ Non (Annuler)", totalAmount, s.Currency, paymentNumber)
	case StepPaymentProcessingWait:
		return "⏳ Traitement en cours...\nVeuillez patienter pendant que nous contactons le service de paiement."
	case StepPaymentCompleted:
		// Compute amount for display
		baseAmountUSD := s.SelectedJourney.Price * float64(len(s.Passengers))
		var totalAmount float64
		if s.Currency == "CDF" {
			rate, err := utils.GetCurrentExchangeRate(DB)
			if err != nil {
				log.Printf("Error getting exchange rate: %v, using default 2850", err)
				rate = 2850
			}
			totalAmount = baseAmountUSD * rate
		} else {
			totalAmount = baseAmountUSD
		}
		// Get payment number for display
		var paymentNumber string
		if s.PaymentPhone != "" {
			paymentNumber = s.PaymentPhone
		} else {
			// Use WhatsApp number
			paymentNumber = GetRealPhoneNumber(Client, types.JID{User: s.UserPhone, Server: "s.whatsapp.net"})
			if paymentNumber == "" {
				paymentNumber = s.UserPhone // fallback
			}
		}
		return fmt.Sprintf("📱 Un message USSD de confirmation de %.2f %s vient d'être envoyé sur le numéro %s.\n🔒 Veuillez saisir votre code PIN Mobile Money sur votre téléphone pour valider.\n⏳ Une fois validé, vous recevrez votre billet ici automatiquement !", totalAmount, s.Currency, paymentNumber)
	case StepAgencyChoice:
		return formatAgencyList(s)
	case StepAgencyMenu:
		return fmt.Sprintf("🏢 %s\n\n1️⃣ Voir la liste des voyages disponibles\n2️⃣ Rechercher un voyage\n\nRépondez avec le numéro de votre choix.", s.SelectedAgency.Name)
	case StepAgencyDateChoice:
		return "📅 Quelle date ?\n\n1️⃣ Aujourd'hui\n2️⃣ Saisir une date\n\nRépondez avec le numéro de votre choix."
	case StepAgencyDateInput:
		return "📅 Entrez la date du voyage (ex : 13/09/2026 ou 13 septembre 2026)."
	case StepAgencyVoyages:
		return formatPaginatedResults(s)
	case StepMyTicketsPhone:
		return "📱 Entrez le numéro de téléphone pour chercher vos billets :"
	case StepAgencyName:
		return "🏢 Nom de l'agence ?"
	case StepAgencyManagerName:
		return "👔 Nom complet du responsable ?"
	case StepAgencyAddress:
		return "📍 Ville / Adresse ?"
	case StepAgencyContactPhone:
		return "📞 Numéro de téléphone officiel de l'agence ?"
	case StepAgencyConfirm:
		return fmt.Sprintf("✅ Veuillez confirmer les informations de l'agence :\n\nNom de l'agence : %s\nResponsable : %s\nAdresse : %s\nTéléphone : %s\n\nRépondez par 'oui' pour confirmer ou 'menu' pour annuler.", s.AgencyName, s.ManagerName, s.Address, s.ContactPhone)
	case StepPasswordChangeMenu:
		return "🔑 Voulez-vous changer votre mot de passe ?\n\n1️⃣ Oui\n2️⃣ Non\n\nRépondez avec le numéro de votre choix."
	case StepVerificationMenu:
		return "🎫 *Vérification / Guichet*\n\n1️⃣ Voir mes billets\n2️⃣ Guichet Contrôleur\n3️⃣ Retour au menu principal\n\nRépondez avec le numéro de votre choix."
	case StepViewTickets:
		return "Sélectionnez un billet pour voir les détails :"
	case StepSelectTicketDetails:
		return "" // Dynamic prompt handled in process()
	case StepControllerInput:
		return "🔐 Veuillez saisir votre code PIN agent :"
	case StepControllerActions:
		return "" // Dynamic prompt handled in process()
	case StepControllerPaymentReferenceInput:
		return "💳 *Validation de Paiement*\n\nVeuillez saisir le numéro de réservation (Booking Reference) ou le numéro de téléphone du passager :"
	case StepControllerPaymentMethod:
		return "💳 *Validation de Paiement*\n\nComment souhaitez-vous rechercher la réservation ?\n\n1️⃣ Par référence de réservation (ex: REF12345)\n2️⃣ Par numéro de téléphone du passager\n3️⃣ Par image / photo du QR Code\n\n0️⃣ Retour au menu contrôleur\n\nRépondez avec le numéro de votre choix."
	case StepControllerInputBookingRef:
		return "✍️ Veuillez saisir la **référence de la réservation** :"
	case StepControllerInputPhone:
		return "📱 Veuillez saisir le **numéro de téléphone du passager** :"
	case StepControllerInputQRCode:
		return "📸 Veuillez **envoyer la photo / l'image du QR code** du billet :"
	case StepControllerCheckInConfirm:
		return "" // Dynamic prompt handled in process()
	case StepControllerCheckInProcess:
		return "" // Dynamic prompt handled in process()
	case StepControllerEnterPin:
		return "👮 *Guichet Contrôleur - Authentification*\n\nVeuillez saisir votre code PIN agent :"
	case StepControllerSelectAction:
		return "" // Dynamic prompt handled in process()
	case StepControllerValidatePayment:
		return "💳 *Validation de Paiement*\n\nVeuillez fournir l'une des informations suivantes :\n1️⃣ **Référence de réservation** (ex: BBb428a003)\n2️⃣ **Numéro de téléphone du passager** (ex: 243974645174)\n3️⃣ **Nom complet du passager** (ex: Lazing Kasuku)\n4️⃣ Ou **Envoyez directement l'image du QR Code** du billet.\n\nTapez *menu* pour annuler."
	case StepControllerDoCheckin:
		return "🎫 *Check-in*\n\nVeuillez saisir la référence du billet ou le token :"
	case StepControllerSearchByRef:
		return "✍️ Veuillez entrer la référence de réservation (ex: BBb428a003) :"
	case StepControllerSearchByPhone:
		return "📱 Veuillez entrer le numéro de téléphone du passager (ex: 243974645174) :"
	case StepControllerSearchByName:
		return "👤 Veuillez entrer le nom complet du passager (ex: Lazing Kasuku) :"
	case StepControllerSearchByQR:
		return "📸 Veuillez envoyer l'image du QR Code du billet :"
	default:
		return "❓ Étape inconnue. Tapez *menu* pour revenir au menu principal."
	}
	return ""
}

func formatDate(dateStr string) string {
	if dateStr == "" {
		return ""
	}
	parts := strings.Split(dateStr, "-")
	if len(parts) != 3 {
		return dateStr
	}
	return fmt.Sprintf("%s/%s/%s", parts[2], parts[1], parts[0])
}

func formatTime(timeStr string) string {
	if timeStr == "" {
		return ""
	}
	parts := strings.Split(timeStr, ":")
	if len(parts) < 2 {
		return timeStr
	}
	hour := parts[0]
	minute := parts[1]
	if len(hour) == 1 {
		hour = "0" + hour
	}
	if len(minute) == 1 {
		minute = "0" + minute
	}
	return fmt.Sprintf("%sh%s", hour, minute)
}

func formatDateTime(dateStr, timeStr string) string {
	if dateStr == "" {
		return timeStr
	}
	formattedDate := formatDate(dateStr)
	if timeStr == "" {
		return formattedDate
	}
	formattedTime := formatTime(timeStr)
	return fmt.Sprintf("%s à %s", formattedDate, formattedTime)
}

func calculateDuration(startTimeStr, endTimeStr string) string {
	if startTimeStr == "" || endTimeStr == "" {
		return ""
	}

	startParts := strings.Split(startTimeStr, ":")
	if len(startParts) < 2 {
		return ""
	}
	startHour, err1 := strconv.Atoi(startParts[0])
	startMinute, err2 := strconv.Atoi(startParts[1])
	if err1 != nil || err2 != nil {
		return ""
	}

	endParts := strings.Split(endTimeStr, ":")
	if len(endParts) < 2 {
		return ""
	}
	endHour, err3 := strconv.Atoi(endParts[0])
	endMinute, err4 := strconv.Atoi(endParts[1])
	if err3 != nil || err4 != nil {
		return ""
	}

	startTotal := startHour*60 + startMinute
	endTotal := endHour*60 + endMinute

	if endTotal < startTotal {
		endTotal += 24 * 60
	}

	durationMinutes := endTotal - startTotal
	durationHours := durationMinutes / 60
	durationMins := durationMinutes % 60

	return fmt.Sprintf("%dh%02d", durationHours, durationMins)
}

func formatMyTickets(tickets []CreatedTicket) string {
	if len(tickets) == 0 {
		return "Aucun billet trouvé."
	}
	var b strings.Builder
	b.WriteString("Voici vos billets :\n\n")
	for i, t := range tickets {
		b.WriteString(fmt.Sprintf("%s Billet %d\n", numberEmoji(i+1), i+1))
		b.WriteString(fmt.Sprintf("   Passenger : %s\n", t.PassengerName))
		b.WriteString(fmt.Sprintf("   Siège     : %s\n", t.SeatNumber))
		b.WriteString(fmt.Sprintf("   Prix      : %.2f USD\n", t.Price))
		b.WriteString(fmt.Sprintf("   Télécharger : %s\n\n", t.DownloadURL))
	}
	return b.String()
}

func RegisterHandlers() {
	log.Printf("[INFO] Registering WhatsApp event handlers")
	Client.AddEventHandler(func(evt interface{}) {
		// DEBUG: Log ALL events received
		log.Printf("[WHATSAPP-EVENT] Received event type: %T", evt)

		switch v := evt.(type) {
		case *events.Message:
			// DEBUG: Log ALL incoming message events with full details
			log.Printf("[WHATSAPP-DEBUG] Received message event from %s (IsFromMe: %v, MessageType: %T, Timestamp: %v, IsGroup: %v)",
				v.Info.Sender.User, v.Info.IsFromMe, v.Message, v.Info.Timestamp, v.Info.IsGroup)

			// Log the raw message structure for debugging
			if v.Message != nil {
				msgType := "unknown"
				if v.Message.GetConversation() != "" {
					msgType = "conversation"
				} else if v.Message.GetExtendedTextMessage() != nil {
					msgType = "extendedText"
				} else if v.Message.GetImageMessage() != nil {
					msgType = "image"
				} else if v.Message.GetDocumentMessage() != nil {
					msgType = "document"
				} else if v.Message.GetVideoMessage() != nil {
					msgType = "video"
				} else if v.Message.GetAudioMessage() != nil {
					msgType = "audio"
				} else if v.Message.GetStickerMessage() != nil {
					msgType = "sticker"
				} else if v.Message.GetLocationMessage() != nil {
					msgType = "location"
				} else if v.Message.GetContactMessage() != nil {
					msgType = "contact"
				} else if v.Message.GetButtonsResponseMessage() != nil {
					msgType = "buttonsResponse"
				} else if v.Message.GetListResponseMessage() != nil {
					msgType = "listResponse"
				} else if v.Message.GetTemplateMessage() != nil {
					msgType = "template"
				} else if v.Message.GetReactionMessage() != nil {
					msgType = "reaction"
				} else if v.Message.GetPollCreationMessage() != nil {
					msgType = "pollCreation"
				} else if v.Message.GetPollUpdateMessage() != nil {
					msgType = "pollUpdate"
				}
				log.Printf("[WHATSAPP-DEBUG] Message type: %s, ID: %s, Chat: %s", msgType, v.Info.ID, v.Info.Chat)

				// Log text content if available
				text := extractText(v)
				if text != "" {
					log.Printf("[WHATSAPP-DEBUG] Extracted text: %q", text)
				}
			} else {
				log.Printf("[WHATSAPP-DEBUG] Message is nil!")
			}

			// Don't return early - let handleIncomingMessage process all message types
			handleIncomingMessage(v)
		case *events.Receipt:
			log.Printf("[WHATSAPP-DEBUG] Receipt event: Type=%s, MessageIDs=%v, Chat=%s", v.Type, v.MessageIDs, v.Chat)
		case *events.Presence:
			log.Printf("[WHATSAPP-DEBUG] Presence event: From=%s, Unavailable=%v", v.From, v.Unavailable)
		case *events.HistorySync:
			log.Printf("[WHATSAPP-DEBUG] HistorySync event received")
		case *events.AppState:
			log.Printf("[WHATSAPP-DEBUG] AppState event received")
		case *events.OfflineSyncCompleted:
			log.Printf("[WHATSAPP-DEBUG] OfflineSyncCompleted event")
		case *events.OfflineSyncPreview:
			log.Printf("[WHATSAPP-DEBUG] OfflineSyncPreview event")
		default:
			log.Printf("[WHATSAPP-DEBUG] Unhandled event type: %T", evt)
		}
	})

	// Also add a handler for connection state changes
	Client.AddEventHandler(func(evt interface{}) {
		if connEvt, ok := evt.(interface{ String() string }); ok {
			log.Printf("[WHATSAPP-CONNECTION] State change: %s", connEvt.String())
		}
	})
}

// TODO: Test edit
func handleIncomingMessage(evt *events.Message) {
	// Ensure client is initialized
	if Client == nil {
		log.Printf("[ERROR] WhatsApp client is nil, cannot process message")
		return
	}

	if evt.Info.IsFromMe {
		log.Printf("[WHATSAPP-DEBUG] Ignoring message from self")
		return
	}

	sender := evt.Info.Sender
	phone := sender.User
	session := getSession(phone)

	// Check if this is an image message and we're in the controller QR code input step
	msg := evt.Message
	imageMsg := msg.GetImageMessage()
	if imageMsg != nil && session.Step == StepControllerInputQRCode {
		log.Printf("Received image message for QR code scanning from %s", phone)

		// Try to extract QR code from image
		qrContent, err := extractQRFromImage(evt)
		if err != nil {
			// QR code decode failed - send friendly error with fallback option
			errorMsg := `⚠️ *Impossible de lire le QR Code sur cette photo.*

Conseils :
• Assurez-vous que la photo est bien nette et bien éclairée.
• Vous pouvez réessayer d'envoyer une photo plus claire.

💡 *Alternative rapide :*
Tapez directement les **6 à 10 caractères de la référence** du billet (ex: BBb428a003) ou tapez *menu* pour annuler.`

			saveSession(phone, session)
			SendText(sender, errorMsg)
			return
		}

		// QR code decoded successfully, process as text input
		text := strings.TrimSpace(qrContent)
		log.Printf("QR code decoded successfully for %s: %q", phone, text)
		reply := process(phone, session, text, sender)
		saveSession(phone, session)
		SendText(sender, reply)
		return
	}

	// Handle regular text messages
	text := strings.TrimSpace(extractText(evt))
	if text == "" {
		log.Printf("[WHATSAPP-DEBUG] No extractable text from message for %s", phone)
		return
	}

	log.Printf("Handling message for phone %s, current step %s, text: %q", phone, session.Step, text)

	reply := process(phone, session, text, sender)
	saveSession(phone, session)
	SendText(sender, reply)
}

func process(phone string, s *Session, text string, sender types.JID) string {
	// Safety check: ensure session is not nil
	if s == nil {
		log.Printf("[ERROR] Session is nil for phone %s, creating new session", phone)
		s = &Session{Step: StepMenu, UserPhone: phone, Data: make(map[string]interface{})}
	}

	// Ensure Data map is initialized
	if s.Data == nil {
		s.Data = make(map[string]interface{})
	}

	log.Printf("Processing message for phone %s, current step %s, text: %q", phone, s.Step, text)

	// Ensure Client is available
	if Client == nil {
		log.Printf("[ERROR] WhatsApp client is nil in process()")
		return "❌ Service temporairement indisponible. Veuillez réessayer dans quelques instants."
	}

	// Update UserPhone to the real phone number from the sender JID
	realPhone := GetRealPhoneNumber(Client, sender)
	if realPhone != "" {
		s.UserPhone = realPhone
	}
	lower := strings.ToLower(text)
	// Handle back navigation
	if lower == "0" || lower == "retour" {
		GoBack(phone)
		return getPromptForStep(s)
	}
	if lower == "menu" || lower == "annuler" {
		// Reset the current session directly instead of calling resetSession
		// to avoid saving the old session back over the reset one
		s.Step = StepMenu
		s.PreviousStep = ""
		s.Origin = ""
		s.Destination = ""
		s.Date = ""
		s.Results = nil
		s.SelectedJourneyID = ""
		s.SelectedJourney = JourneyResult{}
		s.SelectedAgencyID = ""
		s.SelectedAgency = Agency{}
		s.FilterAgencyID = ""
		s.SeatsNeeded = 0
		s.Passengers = nil
		s.CurrentPassengerIndex = 0
		s.PaymentPhone = ""
		s.PaymentMethod = ""
		s.Currency = ""
		s.Agencies = nil
		s.AgencyPage = 0
		s.AgencyName = ""
		s.ManagerName = ""
		s.Address = ""
		s.ContactPhone = ""
		s.UserPhone = phone // Keep the user's phone number
		return welcomeMenu
	}

	// Prevent re-processing payment flow if payment is already in progress (but allow processing confirmation/cancellation from StepPaymentProcessing)
	switch s.Step {
	case StepPaymentProcessingInProgress, StepPaymentProcessingWait, StepPaymentCompleted:
		// If payment is already in progress, send a short message and don't change state
		return "⏳ Un paiement est déjà en cours de traitement. Veuillez valider le code PIN sur votre téléphone ou patienter."
	}

	switch s.Step {
	case StepMenu:
		switch text {
		case "1":
			s.PreviousStep = s.Step
			s.Step = StepSearchOrigin
			return getPromptForStep(s)
		case "2":
			agencies, err := getActiveAgencies()
			if err != nil {
				return "Erreur lors de la récupération des agences. Tapez *menu* pour réessayer."
			}
			log.Printf("Found %d active agencies for phone %s", len(agencies), phone)
			if len(agencies) == 0 {
				return "Aucune agence active trouvée. Tapez *menu* pour réessayer."
			}
			s.Agencies = agencies
			s.PreviousStep = s.Step
			s.Step = StepAgencyChoice
			return getPromptForStep(s)
		case "3":
			s.PreviousStep = s.Step
			s.Step = StepAgencyName
			return getPromptForStep(s)
		case "4":
			// Vérification / Guichet
			s.PreviousStep = s.Step
			s.Step = StepVerificationMenu
			return getPromptForStep(s)
		case "5":
			// Aide & Assistance
			return "📞 Pour toute assistance ou information, notre service client est à votre disposition :\n- Ligne directe & WhatsApp : +243974645174\n- Email : ayaletrans@gmail.com"
		default:
			return welcomeMenu
		}

	case StepSearchOrigin:
		// Validate origin input
		if strings.TrimSpace(text) == "" {
			return "Veuillez entrer un lieu de départ valide."
		}
		s.Origin = strings.TrimSpace(text)
		s.PreviousStep = s.Step
		s.Step = StepSearchDestination
		return getPromptForStep(s)

	case StepSearchDestination:
		// Validate destination input
		if strings.TrimSpace(text) == "" {
			return "Veuillez entrer un lieu de destination valide."
		}
		s.Destination = strings.TrimSpace(text)
		s.PreviousStep = s.Step
		s.Step = StepSearchDateChoice
		return getPromptForStep(s)

	case StepSearchDateChoice:
		if text == "1" {
			s.Date = time.Now().Format("2006-01-02")
			s.PreviousStep = s.Step
			s.Step = StepSearchResults // We'll handle the search in the next step
			// We need to do the search and then return the prompt for StepSearchResults
			// But note: we are changing the step to StepSearchResults, so we will return the prompt for that step.
			// However, we need to run the search first to populate s.Results.
			// Let's do the search now and then return the prompt for StepSearchResults.
			results, err := SearchJourneys(s.Origin, s.Destination, s.Date, "")
			if err == nil && len(results) > 0 {
				s.Results = results
				// Reset voyage pagination for new results
				s.VoyagePage = 0
				return formatPaginatedResults(s)
			}
			var suggestions []JourneyResult
			sameRoute, _ := SearchUpcomingJourneys(s.Origin, s.Destination, 5)
			suggestions = append(suggestions, sameRoute...)
			otherRoutes, _ := SearchJourneysOnDate(s.Date, 5)
			for _, r := range otherRoutes {
				found := false
				for _, existing := range suggestions {
					if existing.ID == r.ID {
						found = true
						break
					}
				}
				if !found {
					suggestions = append(suggestions, r)
				}
			}
			if len(suggestions) == 0 {
				resetSession(phone)
				return "Aucun voyage trouvé pour ce trajet ni pour cette date. Tapez *menu* pour recommencer."
			}
			s.Results = suggestions
			// Reset voyage pagination for new results
			s.VoyagePage = 0
			return fmt.Sprintf("Aucun voyage trouvé pour %s → %s le %s.\nVoici d'autres voyages disponibles :\n\n%s",
				s.Origin, s.Destination, s.Date, formatPaginatedResults(s))
		}
		if text == "2" {
			s.PreviousStep = s.Step
			s.Step = StepSearchDate
			return getPromptForStep(s)
		}
		return "Choix invalide. Tapez 1 pour aujourd'hui ou 2 pour saisir une date."

	case StepSearchDate:
		date := parseDate(text)
		if date == "" {
			return "Format de date invalide. Essayez par exemple 13/09/2026 ou 13 septembre 2026."
		}
		s.Date = date
		s.PreviousStep = s.Step
		s.Step = StepSearchResults
		// We need to do the search and then return the prompt for StepSearchResults
		results, err := SearchJourneys(s.Origin, s.Destination, s.Date, "")
		if err == nil && len(results) > 0 {
			s.Results = results
			// Reset voyage pagination for new results
			s.VoyagePage = 0
			return formatPaginatedResults(s)
		}
		var suggestions []JourneyResult
		var sameRoute []JourneyResult
		var otherRoutes []JourneyResult
		sameRoute, _ = SearchUpcomingJourneys(s.Origin, s.Destination, 5)
		suggestions = append(suggestions, sameRoute...)
		otherRoutes, _ = SearchJourneysOnDate(s.Date, 5)
		for _, r := range otherRoutes {
			found := false
			for _, existing := range suggestions {
				if existing.ID == r.ID {
					found = true
					break
				}
			}
			if !found {
				suggestions = append(suggestions, r)
			}
		}
		if len(suggestions) == 0 {
			resetSession(phone)
			return "Aucun voyage trouvé pour ce trajet ni pour cette date. Tapez *menu* pour recommencer."
		}
		s.Results = suggestions
		// Reset voyage pagination for new results
		s.VoyagePage = 0
		return fmt.Sprintf("Aucun voyage trouvé pour %s → %s le %s.\nVoici d'autres voyages disponibles :\n\n%s",
			s.Origin, s.Destination, s.Date, formatPaginatedResults(s))

	case StepSearchResults:
		// Handle voyage selection from search results with pagination
		trimmed := strings.TrimSpace(text)

		// Handle fixed numeric navigation codes ONLY (strict matching)
		// Navigation is EXCLUSIVELY "01" (next) and "00" (previous)
		if trimmed == "01" || trimmed == "0️⃣1️⃣" {
			// Next page
			pageSize := 10
			startIndex := s.VoyagePage * pageSize
			endIndex := startIndex + pageSize
			if endIndex > len(s.Results) {
				endIndex = len(s.Results)
			}
			if endIndex < len(s.Results) {
				s.VoyagePage++
				return getPromptForStep(s)
			}
			return "Vous êtes déjà sur la dernière page."
		}
		if trimmed == "00" || trimmed == "0️⃣0️⃣" {
			// Previous page
			if s.VoyagePage > 0 {
				s.VoyagePage--
				return getPromptForStep(s)
			}
			return "Vous êtes déjà sur la première page."
		}

		// Fallback: text-based navigation (for backward compatibility)
		lower := strings.ToLower(trimmed)
		if lower == "precedent" || lower == "précédent" || lower == "previous" {
			if s.VoyagePage > 0 {
				s.VoyagePage--
				return getPromptForStep(s)
			}
			return "Vous êtes déjà sur la première page."
		}
		if lower == "suivant" || lower == "next" {
			pageSize := 10
			startIndex := s.VoyagePage * pageSize
			endIndex := startIndex + pageSize
			if endIndex > len(s.Results) {
				endIndex = len(s.Results)
			}
			if endIndex < len(s.Results) {
				s.VoyagePage++
				return getPromptForStep(s)
			}
			return "Vous êtes déjà sur la dernière page."
		}

		// Handle numeric selection (global index) - ANY other number is item selection
		choice, err := strconv.Atoi(trimmed)
		if err != nil {
			return "Choix invalide. Veuillez entrer un numéro valide (00 pour précédent, 01 pour suivant)."
		}

		// Validate the global index
		if choice < 1 || choice > len(s.Results) {
			return fmt.Sprintf("Numéro invalide. Veuillez entrer un numéro entre 1 et %d.", len(s.Results))
		}

		// Convert global index (1-based) to array index (0-based)
		realIndex := choice - 1

		// Verify the item is on the current page
		pageSize := 10
		startIndex := s.VoyagePage * pageSize
		endIndex := startIndex + pageSize
		if endIndex > len(s.Results) {
			endIndex = len(s.Results)
		}
		if realIndex < startIndex || realIndex >= endIndex {
			// User selected an item not on current page - guide them
			currentPageNum := s.VoyagePage + 1
			totalPages := (len(s.Results) + pageSize - 1) / pageSize
			return fmt.Sprintf("Cet élément est sur une autre page (page %d/%d).\nUtilisez 00 pour précédent, 01 pour suivant, ou tapez directement le numéro.", currentPageNum, totalPages)
		}

		selected := s.Results[realIndex]
		s.SelectedJourneyID = selected.ID
		s.SelectedJourney = selected
		s.PreviousStep = s.Step
		s.Step = StepSeatsCount
		// Reset pagination for next time
		s.VoyagePage = 0
		return getPromptForStep(s)

	case StepSeatsCount:
		n, err := strconv.Atoi(text)
		if err != nil || n < 1 || n > 10 {
			return "Entrez un nombre valide de sièges (entre 1 et 10)."
		}
		s.SeatsNeeded = n
		s.Passengers = make([]PassengerDraft, 0, n)
		s.CurrentPassengerIndex = 0
		s.PreviousStep = s.Step
		s.Step = StepPassengerName
		return getPromptForStep(s)

	case StepPassengerName:
		// Validate name is not empty and not exclusively numeric
		trimmed := strings.TrimSpace(text)
		if trimmed == "" {
			return "Veuillez entrer un nom valide."
		}
		// Check if the name is exclusively numeric
		if _, err := strconv.Atoi(trimmed); err == nil {
			return "Le nom ne peut pas être composé uniquement de chiffres. Veuillez entrer un nom valide."
		}
		s.Passengers = append(s.Passengers, PassengerDraft{Name: trimmed})
		s.PreviousStep = s.Step
		s.Step = StepPassengerPhone
		return getPromptForStep(s)

	case StepPassengerPhone:
		idx := len(s.Passengers) - 1
		if text != "*" {
			// Validate phone number: must be exactly 12 digits and start with 243
			if _, err := strconv.Atoi(text); err != nil {
				return "Numéro de téléphone invalide. Veuillez entrer uniquement des chiffres (ex: 243XXXXXXXXX) ou * pour passer."
			}
			if len(text) != 12 {
				return "Numéro de téléphone invalide. Le numéro doit contenir exactement 12 chiffres (ex: 243XXXXXXXXX) ou * pour passer."
			}
			if !strings.HasPrefix(text, "243") {
				return "Numéro de téléphone invalide. Le numéro doit commencer par 243 (ex: 243XXXXXXXXX) ou * pour passer."
			}
			s.Passengers[idx].Phone = strings.TrimSpace(text)
		}
		s.CurrentPassengerIndex++
		if s.CurrentPassengerIndex < s.SeatsNeeded {
			s.PreviousStep = s.Step
			s.Step = StepPassengerName
			return getPromptForStep(s)
		}
		s.PreviousStep = s.Step
		s.Step = StepConfirm
		return getPromptForStep(s)

	case StepConfirm:
		if strings.ToLower(text) != "oui" {
			resetSession(phone)
			return "Réservation annulée. Tapez *menu* pour recommencer."
		}
		bookedSeats, _ := GetBookedSeats(s.SelectedJourneyID)
		assignSeats(s.Passengers, bookedSeats)
		s.PreviousStep = s.Step
		s.Step = StepPaymentMethodChoice
		return getPromptForStep(s)

	case StepPaymentMethodChoice:
		switch text {
		case "1": // Cash payment
			// Process cash payment immediately
			guestName := s.Passengers[0].Name
			guestEmail := fmt.Sprintf("whatsapp+%s@ayaletrans.local", phone)

			tickets, ref, err := CreateGuestBooking(s.SelectedJourneyID, guestName, guestEmail, s.Passengers, AppURL, "cash")
			if err != nil {
				if s.NotificationSent {
					resetSession(phone)
					return ""
				}
				s.NotificationSent = true
				resetSession(phone)
				return "Erreur lors de la réservation : " + err.Error() + "\nTapez *menu* pour réessayer."
			}

			sendTicketCard(sender, ref, s.SelectedJourney, tickets[0])

			// Send additional tickets for multiple passengers
			for i := 1; i < len(tickets); i++ {
				sendTicketCard(sender, ref, s.SelectedJourney, tickets[i])
			}

			baseAmountUSD := s.SelectedJourney.Price * float64(len(s.Passengers))
			totalAmountUSD := baseAmountUSD
			rate, err := utils.GetCurrentExchangeRate(DB)
			if err != nil {
				log.Printf("Error getting exchange rate: %v, using default 2850", err)
				rate = 2850
			}
			totalAmountCDF := baseAmountUSD * rate
			cashMsg := fmt.Sprintf("✅ Votre réservation est confirmée ! Votre carte de billet a été générée ci-dessus.\n\n💰 Montant total à payer : %.2f USD / %.2f CDF\n\n💰 Paiement en attente : Veuillez effectuer le paiement cash au guichet avant le départ pour valider définitivement votre billet.\n🎫 Montrez ce billet à l'embarquement.\n💡 Note : Les passagers payant par Mobile Money ou Carte sont priorit�res sur les places disponibles.", totalAmountUSD, totalAmountCDF)

			if s.NotificationSent {
				resetSession(phone)
				return ""
			}
			s.NotificationSent = true
			resetSession(phone)
			return cashMsg

		case "2": // Mobile Money
			s.PaymentMethod = "mobile"
			s.PreviousStep = s.Step
			s.Step = StepCurrencyChoice
			return getPromptForStep(s)

		case "3": // Card payment
			s.PaymentMethod = "card"
			s.PreviousStep = s.Step
			s.Step = StepCurrencyChoice
			return getPromptForStep(s)

		default:
			return "Choix invalide. Tapez 1 pour espèces, 2 pour Mobile Money ou 3 pour carte."
		}

	case StepCurrencyChoice:
		switch text {
		case "1":
			s.Currency = "USD"
			s.PreviousStep = s.Step
			s.Step = StepPhonePaymentChoice
			return getPromptForStep(s)
		case "2":
			s.Currency = "CDF"
			s.PreviousStep = s.Step
			s.Step = StepPhonePaymentChoice
			return getPromptForStep(s)
		default:
			return "Choix invalide. Répondez avec le numéro de votre choix."
		}

	case StepPhonePaymentChoice:
		switch text {
		case "1": // Use WhatsApp number
			// Don't store anything special - we'll get the real number during confirmation
			s.PreviousStep = s.Step
			s.Step = StepPaymentProcessing
			return getPromptForStep(s)
		case "2": // Enter another number
			s.PreviousStep = s.Step
			s.Step = StepFlexPayPhoneInput
			return getPromptForStep(s)
		default:
			return "Choix invalide. Répondez avec le numéro de votre choix."
		}

	case StepFlexPayPhoneInput:
		trimmedText := strings.TrimSpace(text)
		if trimmedText == "" {
			// Ask again for the number
			return "Entrez le numéro de téléphone pour Mobile Money (format : 243XXXXXXXXX) :"
		}

		// Validate format: should start with 243 and be 12 digits total (243 + 9 digits)
		if !strings.HasPrefix(trimmedText, "243") || len(trimmedText) != 12 {
			return "Numéro de téléphone invalide. Format attendu : 243XXXXXXXXX (12 chiffres commençant par 243)"
		}
		// Check if all characters are digits
		if _, err := strconv.Atoi(trimmedText); err != nil {
			return "Numéro de téléphone invalide. Veuillez entrer uniquement des chiffres."
		}

		// Store the number for mobile money
		if s.PaymentMethod == "mobile" {
			s.PaymentPhone = trimmedText
		}

		// Go to confirmation step
		s.PreviousStep = s.Step
		s.Step = StepPaymentProcessing
		return getPromptForStep(s)

	case StepPaymentProcessing:
		// Handle confirmation: 1 = Oui (Validate), 2 = Non (Cancel)
		if text == "1" {
			// Prevent reentrancy: set step to in-progress immediately
			s.Step = StepPaymentProcessingInProgress
			// User confirmed - initiate payment
			var paymentPhone string
			if s.PaymentPhone != "" {
				paymentPhone = s.PaymentPhone
			} else {
				// Use WhatsApp number
				paymentPhone = GetRealPhoneNumber(Client, types.JID{User: s.UserPhone, Server: "s.whatsapp.net"})
				if paymentPhone == "" {
					paymentPhone = s.UserPhone // fallback
				}
			}

			var redirectURL string
			var reference string
			var err error
			switch s.PaymentMethod {
			case "mobile":
				// For mobile money, we need to determine the phone number to use
				redirectURL, reference, err = InitiateMobileMoneyPayment(s.SelectedJourneyID, s.SelectedJourney, s.Passengers, paymentPhone, s.UserPhone, s.Currency, sender.String())
			case "card":
				// For card payment
				redirectURL, reference, err = InitiateOnlinePayment(s.SelectedJourneyID, s.SelectedJourney, s.Passengers, s.UserPhone, s.Currency, sender.String(), s.UserPhone)
			default:
				err = fmt.Errorf("unknown payment method")
			}
			// Use the variables to prevent "declared and not used" errors
			_ = redirectURL
			_ = reference
			if err != nil {
				if s.NotificationSent {
					resetSession(phone)
					return ""
				}
				s.NotificationSent = true
				s.PaymentError = err.Error()
				s.Step = StepPaymentFailed
				saveSession(phone, s)
				return fmt.Sprintf("❌ Erreur lors de l'initiation du paiement : %s\n\n1️⃣ Réessayer le paiement\n2️⃣ Retour au menu principal", s.PaymentError)
			}
			// Store the order number for tracking
			s.OrderNumber = reference
			saveSession(phone, s)

			// Payment initiated successfully
			// Build the USSD message by setting the step to completed
			s.Step = StepPaymentCompleted
			// Use redirectURL to avoid "declared but not used" error
			_ = redirectURL
			ussdMessage := getPromptForStep(s)
			// DO NOT reset session here - we need to keep data for USSD display and retry
			return ussdMessage
		} else if text == "2" {
			// User cancelled
			if s.NotificationSent {
				resetSession(phone)
				return ""
			}
			s.NotificationSent = true
			resetSession(phone)
			return "Paiement annulé. Tapez *menu* pour revenir au menu principal."
		} else {
			// Invalid input
			return "Choix invalide. Répondez par :\n1️⃣ Oui pour valider\n2️⃣ Non pour annuler"
		}

	case StepPaymentProcessingWait:
		// Process the payment
		var redirectURL string
		var reference string
		var err error

		switch s.PaymentMethod {
		case "mobile":
			// For mobile money, we need to determine the phone number to use
			var paymentPhone string
			if s.PaymentPhone != "" {
				paymentPhone = s.PaymentPhone
			} else {
				// Use WhatsApp number
				paymentPhone = GetRealPhoneNumber(Client, types.JID{User: s.UserPhone, Server: "s.whatsapp.net"})
				if paymentPhone == "" {
					paymentPhone = s.UserPhone // fallback
				}
			}
			// Initiate mobile money payment
			redirectURL, reference, err = InitiateMobileMoneyPayment(s.SelectedJourneyID, s.SelectedJourney, s.Passengers, paymentPhone, s.UserPhone, s.Currency, sender.String())
		case "card":
			// For card payment
			// Initiate online payment
			redirectURL, reference, err = InitiateOnlinePayment(s.SelectedJourneyID, s.SelectedJourney, s.Passengers, s.UserPhone, s.Currency, sender.String(), s.UserPhone)
		default:
			if s.NotificationSent {
				resetSession(phone)
				return ""
			}
			s.NotificationSent = true
			s.PaymentError = "Méthode de paiement inconnue"
			s.Step = StepPaymentFailed
			saveSession(phone, s)
			return "Méthode de paiement inconnue. Tapez *menu* pour réessayer."
		}

		if err != nil {
			if s.NotificationSent {
				resetSession(phone)
				return ""
			}
			s.NotificationSent = true
			s.PaymentError = err.Error()
			s.Step = StepPaymentFailed
			saveSession(phone, s)
			return fmt.Sprintf("❌ Erreur lors de la création du paiement : %s\n\n1️⃣ Réessayer le paiement\n2️⃣ Retour au menu principal", s.PaymentError)
		}

		// Payment initiated successfully - go to completed state
		s.PreviousStep = s.Step
		s.OrderNumber = reference
		saveSession(phone, s)
		s.Step = StepPaymentCompleted
		// Use redirectURL to avoid "declared but not used" error
		_ = redirectURL
		return getPromptForStep(s)

	case StepPaymentCompleted:
		// Handle user input after payment completion
		if text == "2" {
			// User wants to go back to main menu
			resetSession(phone)
			s.Step = StepMenu
			return welcomeMenu
		}
		// For any other input, show the USSD message again (but don't reset session yet)
		// Actually, we should only show this once, so for non-"2" input, we'll reset and return empty
		// But let's follow the pattern: if they didn't type "2", we treat it as invalid and reset
		if s.NotificationSent {
			resetSession(phone)
			return ""
		}
		s.NotificationSent = true
		resetSession(phone)
		return getPromptForStep(s)
	case StepPaymentFailed, StepPaymentExpired:
		// Handle user input after payment failure/expired
		if text == "1" {
			// User wants to retry payment
			if s.HasRetried {
				// Already retried once, cannot retry again
				if s.NotificationSent {
					resetSession(phone)
					return ""
				}
				s.NotificationSent = true
				resetSession(phone)
				return "❌ Nombre maximum de relances atteint. La réservation a été annulée.\n\nTapez *menu* pour revenir au menu principal."
			}
			// First retry allowed
			s.HasRetried = true
			s.Step = StepPaymentProcessing
			return getPromptForStep(s)
		} else if text == "2" || strings.ToLower(text) == "menu" || strings.ToLower(text) == "annuler" {
			// User wants to go back to main menu
			resetSession(phone)
			s.Step = StepMenu
			return welcomeMenu
		} else if text == "0" || strings.ToLower(text) == "retour" {
			// Go back to previous step (which should be payment method choice or similar)
			GoBack(phone)
			return getPromptForStep(s)
		} else {
			// Invalid input, show error message again
			if s.NotificationSent {
				resetSession(phone)
				return ""
			}
			s.NotificationSent = true
			resetSession(phone)
			var errorMsg string
			if s.PaymentError != "" {
				errorMsg = s.PaymentError
			} else {
				if s.Step == StepPaymentFailed {
					errorMsg = "Paiement échoué"
				} else {
					errorMsg = "Paiement expiré"
				}
			}
			return fmt.Sprintf("%v\n\n1️⃣ Relancer le paiement\n2️⃣ Retour au menu", errorMsg)
		}
	case StepAgencyChoice:
		// Handle agency selection with pagination
		trimmed := strings.TrimSpace(text)

		// Handle fixed numeric navigation codes ONLY (strict matching)
		// Navigation is EXCLUSIVELY "01" (next) and "00" (previous)
		if trimmed == "01" || trimmed == "0️⃣1️⃣" {
			// Next page
			pageSize := 10
			startIndex := s.AgencyPage * pageSize
			endIndex := startIndex + pageSize
			if endIndex > len(s.Agencies) {
				endIndex = len(s.Agencies)
			}
			if endIndex < len(s.Agencies) {
				s.AgencyPage++
				return getPromptForStep(s)
			}
			return "Vous êtes déjà sur la dernière page."
		}
		if trimmed == "00" || trimmed == "0️⃣0️⃣" {
			// Previous page
			if s.AgencyPage > 0 {
				s.AgencyPage--
				return getPromptForStep(s)
			}
			return "Vous êtes déjà sur la première page."
		}

		// Fallback: text-based navigation (for backward compatibility)
		lower := strings.ToLower(trimmed)
		if lower == "precedent" || lower == "précédent" || lower == "previous" {
			if s.AgencyPage > 0 {
				s.AgencyPage--
				return getPromptForStep(s)
			}
			return "Vous êtes déjà sur la première page."
		}
		if lower == "suivant" || lower == "next" {
			pageSize := 10
			startIndex := s.AgencyPage * pageSize
			endIndex := startIndex + pageSize
			if endIndex > len(s.Agencies) {
				endIndex = len(s.Agencies)
			}
			if endIndex < len(s.Agencies) {
				s.AgencyPage++
				return getPromptForStep(s)
			}
			return "Vous êtes déjà sur la dernière page."
		}

		// Handle numeric selection (global index) - ANY other number is item selection
		choice, err := strconv.Atoi(trimmed)
		if err != nil {
			return "Choix invalide. Veuillez entrer un numéro valide (00 pour précédent, 01 pour suivant)."
		}

		// Validate the global index
		if choice < 1 || choice > len(s.Agencies) {
			return fmt.Sprintf("Numéro invalide. Veuillez entrer un numéro entre 1 et %d.", len(s.Agencies))
		}

		// Convert global index (1-based) to array index (0-based)
		realIndex := choice - 1

		// Verify the item is on the current page (optional but good for UX)
		pageSize := 10
		startIndex := s.AgencyPage * pageSize
		endIndex := startIndex + pageSize
		if endIndex > len(s.Agencies) {
			endIndex = len(s.Agencies)
		}
		if realIndex < startIndex || realIndex >= endIndex {
			// User selected an item not on current page - guide them
			currentPageNum := s.AgencyPage + 1
			totalPages := (len(s.Agencies) + pageSize - 1) / pageSize
			return fmt.Sprintf("Cet élément est sur une autre page (page %d/%d).\nUtilisez 00 pour précédent, 01 pour suivant, ou tapez directement le numéro.", currentPageNum, totalPages)
		}

		selectedAgency := s.Agencies[realIndex]
		s.SelectedAgencyID = selectedAgency.ID
		s.SelectedAgency = selectedAgency
		// After selecting an agency, proceed to agency menu
		s.PreviousStep = s.Step
		s.Step = StepAgencyMenu
		// Reset pagination for next time
		s.AgencyPage = 0
		return getPromptForStep(s)
	case StepAgencyMenu:
		// Handle agency menu selection
		if text == "1" {
			// Show paginated list of voyages for the selected agency
			s.Date = "" // Clear any previous date
			s.PreviousStep = s.Step
			s.Step = StepAgencyDateChoice
			return getPromptForStep(s)
		} else if text == "2" {
			// Launch trip search restricted to the selected agency
			s.FilterAgencyID = s.SelectedAgencyID
			s.PreviousStep = s.Step
			s.Step = StepSearchOrigin
			return getPromptForStep(s)
		} else {
			return "Choix invalide. Veuillez sélectionner 1 ou 2."
		}
	case StepAgencyDateChoice:
		// Handle date selection for agency voyages
		if text == "1" {
			s.Date = time.Now().Format("2006-01-02")
			s.PreviousStep = s.Step
			s.Step = StepAgencyVoyages
			// Get voyages for the selected agency and date
			results, err := SearchJourneys("", "", s.Date, s.SelectedAgencyID)
			if err != nil {
				return "Erreur lors de la recherche des voyages. Tapez *menu* pour réessayer."
			}
			// Filter results to only include voyages from the selected agency
			var filteredResults []JourneyResult
			for _, r := range results {
				if r.AgencyName == s.SelectedAgency.Name {
					filteredResults = append(filteredResults, r)
				}
			}
			if len(filteredResults) == 0 {
				// Try to find upcoming journeys for this agency on this date (filtered by agency)
				var suggestions []JourneyResult
				sameRoute, _ := SearchUpcomingJourneys("", "", 50) // Get more to filter
				for _, r := range sameRoute {
					if r.AgencyName == s.SelectedAgency.Name {
						suggestions = append(suggestions, r)
					}
				}
				otherRoutes, _ := SearchJourneysOnDate(s.Date, 50) // Get more to filter
				for _, r := range otherRoutes {
					if r.AgencyName == s.SelectedAgency.Name {
						found := false
						for _, existing := range suggestions {
							if existing.ID == r.ID {
								found = true
								break
							}
						}
						if !found {
							suggestions = append(suggestions, r)
						}
					}
				}
				if len(suggestions) == 0 {
					return "Aucun voyage trouvé pour cette agence aujourd'hui. Tapez *menu* pour recommencer."
				}
				s.Results = suggestions
			// Reset voyage pagination for new results
			s.VoyagePage = 0
			return fmt.Sprintf("Aucun voyage trouvé pour cette agence aujourd'hui.\nVoici d'autres voyages disponibles :\n\n%s", formatPaginatedResults(s))
			}
			s.Results = filteredResults
			// Reset voyage pagination for new results
			s.VoyagePage = 0
			return formatPaginatedResults(s)
		} else if text == "2" {
			s.PreviousStep = s.Step
			s.Step = StepAgencyDateInput
			return getPromptForStep(s)
		} else {
			return "Choix invalide. Tapez 1 pour aujourd'hui ou 2 pour saisir une date."
		}
	case StepAgencyDateInput:
		// Handle custom date input for agency voyages
		date := parseDate(text)
		if date == "" {
			return "Format de date invalide. Essayez par exemple 13/09/2026 ou 13 septembre 2026."
		}
		s.Date = date
		s.PreviousStep = s.Step
		s.Step = StepAgencyVoyages
		// Get voyages for the selected agency and date
		results, err := SearchJourneys("", "", s.Date, s.SelectedAgencyID)
		if err != nil {
			return "Erreur lors de la recherche des voyages. Tapez *menu* pour réessayer."
		}
		// Filter results to only include voyages from the selected agency
		var filteredResults []JourneyResult
		for _, r := range results {
			if r.AgencyName == s.SelectedAgency.Name {
				filteredResults = append(filteredResults, r)
			}
		}
		if len(filteredResults) == 0 {
			// Try to find upcoming journeys for this agency on this date (filtered by agency)
			var suggestions []JourneyResult
			sameRoute, _ := SearchUpcomingJourneys("", "", 50) // Get more to filter
			for _, r := range sameRoute {
				if r.AgencyName == s.SelectedAgency.Name {
					suggestions = append(suggestions, r)
				}
			}
			otherRoutes, _ := SearchJourneysOnDate(s.Date, 50) // Get more to filter
			for _, r := range otherRoutes {
				if r.AgencyName == s.SelectedAgency.Name {
					found := false
					for _, existing := range suggestions {
						if existing.ID == r.ID {
							found = true
							break
						}
					}
					if !found {
						suggestions = append(suggestions, r)
					}
				}
			}
			if len(suggestions) == 0 {
				return fmt.Sprintf("Aucun voyage trouvé pour cette agence le %s. Tapez *menu* pour recommencer.", date)
			}
			s.Results = suggestions
			// Reset voyage pagination for new results
			s.VoyagePage = 0
			return fmt.Sprintf("Aucun voyage trouvé pour cette agence le %s.\nVoici d'autres voyages disponibles :\n\n%s", date, formatPaginatedResults(s))
		}
		s.Results = filteredResults
		// Reset voyage pagination for new results
		s.VoyagePage = 0
		return formatPaginatedResults(s)
	case StepAgencyVoyages:
		// Handle voyage selection from agency's voyage list with pagination
		trimmed := strings.TrimSpace(text)

		// Handle fixed numeric navigation codes ONLY (strict matching)
		// Navigation is EXCLUSIVELY "01" (next) and "00" (previous)
		if trimmed == "01" || trimmed == "0️⃣1️⃣" {
			// Next page
			pageSize := 10
			startIndex := s.VoyagePage * pageSize
			endIndex := startIndex + pageSize
			if endIndex > len(s.Results) {
				endIndex = len(s.Results)
			}
			if endIndex < len(s.Results) {
				s.VoyagePage++
				return getPromptForStep(s)
			}
			return "Vous êtes déjà sur la dernière page."
		}
		if trimmed == "00" || trimmed == "0️⃣0️⃣" {
			// Previous page
			if s.VoyagePage > 0 {
				s.VoyagePage--
				return getPromptForStep(s)
			}
			return "Vous êtes déjà sur la première page."
		}

		// Fallback: text-based navigation (for backward compatibility)
		lower := strings.ToLower(trimmed)
		if lower == "precedent" || lower == "précédent" || lower == "previous" {
			if s.VoyagePage > 0 {
				s.VoyagePage--
				return getPromptForStep(s)
			}
			return "Vous êtes déjà sur la première page."
		}
		if lower == "suivant" || lower == "next" {
			pageSize := 10
			startIndex := s.VoyagePage * pageSize
			endIndex := startIndex + pageSize
			if endIndex > len(s.Results) {
				endIndex = len(s.Results)
			}
			if endIndex < len(s.Results) {
				s.VoyagePage++
				return getPromptForStep(s)
			}
			return "Vous êtes déjà sur la dernière page."
		}

		// Handle numeric selection (global index) - ANY other number is item selection
		choice, err := strconv.Atoi(trimmed)
		if err != nil {
			return "Choix invalide. Veuillez entrer un numéro valide (00 pour précédent, 01 pour suivant)."
		}

		// Validate the global index
		if choice < 1 || choice > len(s.Results) {
			return fmt.Sprintf("Numéro invalide. Veuillez entrer un numéro entre 1 et %d.", len(s.Results))
		}

		// Convert global index (1-based) to array index (0-based)
		realIndex := choice - 1

		// Verify the item is on the current page
		pageSize := 10
		startIndex := s.VoyagePage * pageSize
		endIndex := startIndex + pageSize
		if endIndex > len(s.Results) {
			endIndex = len(s.Results)
		}
		if realIndex < startIndex || realIndex >= endIndex {
			// User selected an item not on current page - guide them
			currentPageNum := s.VoyagePage + 1
			totalPages := (len(s.Results) + pageSize - 1) / pageSize
			return fmt.Sprintf("Cet élément est sur une autre page (page %d/%d).\nUtilisez 00 pour précédent, 01 pour suivant, ou tapez directement le numéro.", currentPageNum, totalPages)
		}

		selected := s.Results[realIndex]
		s.SelectedJourneyID = selected.ID
		s.SelectedJourney = selected
		s.PreviousStep = s.Step
		s.Step = StepSeatsCount
		// Reset pagination for next time
		s.VoyagePage = 0
		return getPromptForStep(s)

	case StepVerificationMenu:
		switch text {
		case "1":
			// Show my tickets
			// Use the sender's WhatsApp phone number (s.UserPhone is already set from the sender JID)
			phone := s.UserPhone
			// Clean phone number for display (remove leading + if present)
			displayPhone := strings.TrimPrefix(phone, "+")

			tickets, err := GetTicketsByPhone(phone)
			if err != nil {
				log.Printf("[ERROR-SQL-TICKETS] %v", err)
				return "Erreur lors de la récupération des billets. Tapez *menu* pour réessayer."
			}
			if len(tickets) == 0 {
				return fmt.Sprintf("❌ *Aucun billet trouvé pour le numéro* +%s.\n\nTapez *menu* pour revenir au menu principal.", displayPhone)
			}

			// Build the ticket list message
			var b strings.Builder
			b.WriteString("🎫 *Vos Billets de Voyage :*\n\n")
			for i, ticket := range tickets {
				b.WriteString(fmt.Sprintf("%s Billet %d\n", numberEmoji(i+1), i+1))
				b.WriteString(fmt.Sprintf("   👤 Passager : %s\n", ticket.PassengerName))
				b.WriteString(fmt.Sprintf("   💺 Siège     : %s\n", ticket.SeatNumber))
				b.WriteString(fmt.Sprintf("   💰 Prix      : %.2f USD\n", ticket.Price))
				if ticket.DownloadToken != "" {
					b.WriteString(fmt.Sprintf("   📥 Télécharger : %s/tickets/download/%s\n", AppURL, ticket.DownloadToken))
				}
				b.WriteString("\n")
			}
			b.WriteString("Tapez le numéro du billet pour voir les détails ou tapez *menu* pour revenir.")

			// Store ticket IDs in session Data for later retrieval
			var ticketIDs []string
			for _, ticket := range tickets {
				ticketIDs = append(ticketIDs, ticket.ID)
			}
			s.Data["ticket_ids"] = ticketIDs
			s.PreviousStep = s.Step
			s.Step = StepSelectTicketDetails
			return b.String()
		case "2":
			// Controller menu - redirect to PIN entry
			s.PreviousStep = s.Step
			s.Step = StepControllerEnterPin
			return getPromptForStep(s)
		case "3":
			// Return to main menu
			s.Step = StepMenu
			return welcomeMenu
		default:
			return "Choix invalide. Veuillez sélectionner 1 pour voir vos billets, 2 pour le guichet contrôleur ou 3 pour retourner au menu principal."
		}

	case StepSelectTicketDetails:
		// Handle ticket selection for QR code display
		// Retrieve ticket IDs from session Data
		ticketIDsAny := s.Data["ticket_ids"]
		if ticketIDsAny == nil {
			return "Aucun billet trouvé. Veuillez d'abord sélectionner l'option 1 pour voir vos billets, puis revenir ici. Tapez *menu* pour retourner au menu principal."
		}
		ticketIDs, ok := ticketIDsAny.([]string)
		if !ok || ticketIDs == nil {
			return "Erreur de récupération des billets. Veuillez réessayer en sélectionnant l'option 1 à nouveau. Tapez *menu* pour retourner au menu principal."
		}

		// Validate ticket number input
		ticketNum, err := strconv.Atoi(text)
		if err != nil || ticketNum < 1 || ticketNum > len(ticketIDs) {
			return fmt.Sprintf("Numéro de billet invalide. Veuillez entrer un numéro entre 1 et %d, ou tapez *menu* pour retourner au menu principal.", len(ticketIDs))
		}

		// Get selected ticket ID from our stored list
		selectedTicketID := ticketIDs[ticketNum-1]

		// Get ticket details using the stored ticket ID
		ticket, err := GetTicketDetailsByID(selectedTicketID)
		if err != nil {
			return fmt.Sprintf("Erreur lors de la récupération des détails du billet : %v. Tapez *menu* pour retourner au menu principal.", err)
		}

		// Format and return ticket details with QR code
		return formatTicketDetail(ticket)

	case StepViewTickets:
		// Legacy handler - redirect to new flow
		ticketIDsAny := s.Data["ticket_ids"]
		if ticketIDsAny == nil {
			return "Aucun billet trouvé. Veuillez d'abord sélectionner l'option 1 pour voir vos billets, puis revenir ici. Tapez *menu* pour retourner au menu principal."
		}
		ticketIDs, ok := ticketIDsAny.([]string)
		if !ok || ticketIDs == nil {
			return "Erreur de récupération des billets. Veuillez réessayer en sélectionnant l'option 1 à nouveau. Tapez *menu* pour retourner au menu principal."
		}

		// Validate ticket number input
		ticketNum, err := strconv.Atoi(text)
		if err != nil || ticketNum < 1 || ticketNum > len(ticketIDs) {
			return fmt.Sprintf("Numéro de billet invalide. Veuillez entrer un numéro entre 1 et %d, ou tapez *menu* pour retourner au menu principal.", len(ticketIDs))
		}

		// Get selected ticket ID from our stored list
		selectedTicketID := ticketIDs[ticketNum-1]

		// Get ticket details using the stored ticket ID
		ticket, err := GetTicketDetailsByID(selectedTicketID)
		if err != nil {
			return fmt.Sprintf("Erreur lors de la récupération des détails du billet : %v. Tapez *menu* pour retourner au menu principal.", err)
		}

		// Format and return ticket details
		return formatTicketDetail(ticket)

	// Controller payment validation steps
	case StepControllerInput:
		// Handle controller PIN input
		if strings.TrimSpace(text) == "" {
			return "Veuillez saisir votre code PIN agent :"
		}

		pin := strings.TrimSpace(text)

		// Verify the PIN using the sender JID with flexible matching
		result := verifyControllerPINWithSender(sender, pin)
		if result == nil {
			log.Printf("Authentification agent/contrôleur échouée pour l'expéditeur %s et le PIN %s", sender.User, pin)
			return "❌ Code PIN incorrect ou numéro non autorisé. Tapez *menu* pour recommencer."
		}

		// PIN is valid, set controller authentication fields
		s.IsControllerAuthenticated = true
		s.AgentName = result.AgentName
		s.AssignedAgencies = []Agency{{Name: result.AgencyName}}

		// Move to controller menu
		s.PreviousStep = s.Step
		s.Step = StepControllerActions

		// Build controller menu message
		var menu strings.Builder
		menu.WriteString("✅ Authentification réussie !\n")
		menu.WriteString(fmt.Sprintf("Bienvenue, %s !\n", s.AgentName))
		menu.WriteString("👮‍♂️ *Guichet Contrôleur*\n\n")
		menu.WriteString("🏥 Agences affectées :\n")
		for i, agency := range s.AssignedAgencies {
			menu.WriteString(fmt.Sprintf("   %d. %s\n", i+1, agency.Name))
		}
		menu.WriteString("\nSélectionnez une action :\n")
		menu.WriteString("1️⃣ Valider le paiement\n")
		menu.WriteString("2️⃣ Faire le check-in\n")

		return menu.String()

	case StepControllerEnterPin:
		// Handle controller PIN entry
		if strings.TrimSpace(text) == "" {
			return "👮 *Guichet Contrôleur - Authentification*\n\nVeuillez saisir votre code PIN agent :"
		}

		pin := strings.TrimSpace(text)

		// Verify the PIN using the sender JID with flexible matching
		// This handles LID, phone numbers, and other identifier formats dynamically
		result := verifyControllerPINWithSender(sender, pin)
		if result == nil {
			log.Printf("Authentification agent/contrôleur échouée pour l'expéditeur %s et le PIN %s", sender.User, pin)
			return "❌ Code PIN incorrect ou numéro non autorisé. Veuillez réessayer ou tapez *menu* pour annuler."
		}

		// PIN is valid, set controller authentication fields
		s.IsControllerAuthenticated = true
		s.AgentName = result.AgentName
		s.AssignedAgencies = []Agency{{Name: result.AgencyName}}

		// Move to controller action selection
		s.PreviousStep = s.Step
		s.Step = StepControllerSelectAction

		// Build controller menu message with exact format
		var menu strings.Builder
		menu.WriteString("✅ *Authentification réussie !*\n")
		menu.WriteString(fmt.Sprintf("Bienvenue, %s !\n", s.AgentName))
		menu.WriteString("👮 *Guichet Contrôleur*\n\n")
		menu.WriteString("🏢 *Agences affectées :*\n")
		for i, agency := range s.AssignedAgencies {
			menu.WriteString(fmt.Sprintf("%d. %s\n", i+1, agency.Name))
		}
		menu.WriteString("\nSélectionnez une action :\n")
		menu.WriteString("1️⃣ Valider le paiement\n")
		menu.WriteString("2️⃣ Faire le check-in\n")

		return menu.String()

	case StepControllerSelectAction:
		// Handle controller action selection
		choice := strings.TrimSpace(text)
		switch choice {
		case "1", "1️⃣":
			s.PreviousStep = s.Step
			s.Step = StepControllerValidatePayment
			return getPromptForStep(s)
		case "2", "2️⃣":
			s.PreviousStep = s.Step
			s.Step = StepControllerDoCheckin
			return getPromptForStep(s)
		default:
			return "Choix invalide. Veuillez sélectionner 1 pour Valider le paiement ou 2 pour Faire le check-in, ou tapez *menu* pour retourner au menu principal."
		}

	case StepControllerValidatePayment:
		// Handle sub-menu selection (1/2/3/4) or direct input
		input := strings.TrimSpace(text)
		if input == "" {
			return getPromptForStep(s)
		}

		// If user types 1/2/3/4, switch to the corresponding search sub-step
		switch input {
		case "1", "1️⃣":
			s.PreviousStep = s.Step
			s.Step = StepControllerSearchByRef
			return getPromptForStep(s)
		case "2", "2️⃣":
			s.PreviousStep = s.Step
			s.Step = StepControllerSearchByPhone
			return getPromptForStep(s)
		case "3", "3️⃣":
			s.PreviousStep = s.Step
			s.Step = StepControllerSearchByName
			return getPromptForStep(s)
		case "4", "4️⃣":
			s.PreviousStep = s.Step
			s.Step = StepControllerSearchByQR
			return getPromptForStep(s)
		}

		// Otherwise, treat as direct input and search using the unified query
		return searchBookingForPaymentValidation(s, input)

	case StepControllerSearchByRef:
		// Search by booking reference
		input := strings.TrimSpace(text)
		if input == "" {
			return getPromptForStep(s)
		}
		return searchBookingForPaymentValidation(s, input)

	case StepControllerSearchByPhone:
		// Search by passenger phone - returns two lists (unpaid and paid)
		// Check if we already have search results in session (user is making a selection)
		if s.Data != nil {
			if _, hasResults := s.Data["controller_unpaid_results"]; hasResults {
				// User is selecting from the lists
				return handleControllerPhoneSelection(s, text)
			}
		}

		input := strings.TrimSpace(text)
		if input == "" {
			return getPromptForStep(s)
		}
		return searchBookingsByPhoneForController(s, input)

	case StepControllerSearchByName:
		// Search by passenger name
		input := strings.TrimSpace(text)
		if input == "" {
			return getPromptForStep(s)
		}
		return searchBookingForPaymentValidation(s, input)

	case StepControllerSearchByQR:
		// Search by QR token (text input as fallback)
		input := strings.TrimSpace(text)
		if input == "" {
			return getPromptForStep(s)
		}
		return searchBookingForPaymentValidation(s, input)

	case StepControllerDoCheckin:
		// Handle direct check-in: user enters booking reference, ticket reference, or token
		input := strings.TrimSpace(text)
		if input == "" {
			return getPromptForStep(s)
		}

		// Search for booking/ticket using the input
		var bookingResult BookingDetailsResult
		err := DB.Get(&bookingResult, `
		SELECT
			b.id AS booking_id,
			b.id AS id,
			b.booking_reference,
			b.status,
			b.status AS payment_status,
			t.id AS ticket_id,
			t.passenger_name,
			t.passenger_phone,
			t.seat_number,
			t.price,
			o.name AS organization_name,
			r.origin,
			r.destination,
			j.departure_date::text AS departure_date,
			j.departure_time::text AS departure_time
		FROM tickets t
		JOIN bookings b ON t.booking_id = b.id
		JOIN journeys j ON t.journey_id = j.id
		JOIN routes r ON j.route_id = r.id
		JOIN organizations o ON j.organization_id = o.id
		WHERE b.booking_reference = $1
		   OR t.id = $1
		   OR t.qr_token = $1
		LIMIT 1
	`, input)

		if err != nil {
			if err == sql.ErrNoRows {
				return "❌ Aucune réservation ou billet trouvé avec cette référence.\n\nVeuillez vérifier et réessayer, ou tapez *menu* pour annuler."
			}
			return fmt.Sprintf("❌ Erreur lors de la recherche : %v\n\nTapez *menu* pour annuler.", err)
		}

		// Check if ticket is already checked in
		if bookingResult.PaymentStatus == "checked_in" {
			return fmt.Sprintf("ℹ️ Ce billet (Réservation %s) est déjà enregistré (check-in effectué).\n\nTapez *menu* pour retourner au menu principal.", bookingResult.BookingReference)
		}

		// Check if booking is confirmed/paid
		if bookingResult.PaymentStatus != "confirmed" && bookingResult.PaymentStatus != "paid" {
			return fmt.Sprintf("⚠️ Cette réservation (%s) n'est pas encore payée/confirmée.\n\nVeuillez d'abord valider le paiement (option 1) avant de faire le check-in.\n\nTapez *menu* pour retourner au menu principal.", bookingResult.BookingReference)
		}

		// Store booking ID in session data for check-in
		if s.Data == nil {
			s.Data = make(map[string]interface{})
		}
		s.Data["controller_booking_id"] = bookingResult.BookingID
		s.Data["controller_booking_ref"] = bookingResult.BookingReference
		s.Data["controller_ticket_id"] = bookingResult.TicketID

		var passengerName string
		if bookingResult.PassengerNames != "" {
			names := strings.Split(bookingResult.PassengerNames, ",")
			if len(names) > 0 && names[0] != "" {
				passengerName = names[0]
			} else {
				passengerName = "Passager"
			}
		} else if bookingResult.FirstPassengerName != "" {
			passengerName = bookingResult.FirstPassengerName
		} else if bookingResult.PassengerName != "" {
			passengerName = bookingResult.PassengerName
		} else {
			passengerName = "Passager"
		}

		var seatNumber string
		if bookingResult.SeatNumber != "" {
			seatNumber = bookingResult.SeatNumber
		} else if bookingResult.SeatNumbers != "" {
			// Get first seat from comma-separated list
			seats := strings.Split(bookingResult.SeatNumbers, ",")
			if len(seats) > 0 && seats[0] != "" {
				seatNumber = seats[0]
			}
		}

		// Store additional data for check-in confirmation message
		s.Data["controller_passenger_name"] = passengerName
		s.Data["controller_seat_number"] = seatNumber
		s.Data["controller_origin"] = bookingResult.Origin
		s.Data["controller_destination"] = bookingResult.Destination

		// Show check-in confirmation with improved format
		s.PreviousStep = s.Step
		s.Step = StepControllerCheckInConfirm

		var b strings.Builder
		b.WriteString(fmt.Sprintf("ℹ️ Cette réservation (*%s*) est déjà confirmée/payée.\n\n", bookingResult.BookingReference))
		b.WriteString(fmt.Sprintf("👤 *Passager* : %s\n", passengerName))
		b.WriteString(fmt.Sprintf("🚌 *Trajet* : %s ➔ %s\n", bookingResult.Origin, bookingResult.Destination))
		b.WriteString(fmt.Sprintf("📅 *Départ* : %s à %s\n", formatDate(bookingResult.DepartureDate), bookingResult.DepartureTime))
		if seatNumber != "" {
			b.WriteString(fmt.Sprintf("💺 *Siège* : N° %s\n", seatNumber))
		}
		b.WriteString("\nVoulez-vous faire le check-in ?\n\n")
		b.WriteString("1️⃣ Oui (Check-in)\n")
		b.WriteString("2️⃣ Non (Retour)\n")
		return b.String()

	case StepControllerActions:
		// Handle controller menu selection
		switch strings.TrimSpace(text) {
		case "1", "1️⃣": // Valider le paiement
			s.PreviousStep = s.Step
			s.Step = StepControllerPaymentMethod
			return getPromptForStep(s)
		case "2", "2️⃣": // Faire le check-in
			s.PreviousStep = s.Step
			s.Step = StepControllerDoCheckin
			return getPromptForStep(s)
		default:
			return "Choix invalide. Veuillez sélectionner 1 pour valider le paiement ou 2 pour faire le check-in, or tapez *menu* pour retourner au menu principal."
		}

	case StepControllerPaymentMethod:
		// Handle payment validation method selection
		switch strings.TrimSpace(text) {
		case "1", "1️⃣": // Par référence de réservation
			s.PreviousStep = s.Step
			s.Step = StepControllerInputBookingRef
			return getPromptForStep(s)
		case "2", "2️⃣": // Par numéro de téléphone du passager
			s.PreviousStep = s.Step
			s.Step = StepControllerInputPhone
			return getPromptForStep(s)
		case "3", "3️⃣": // Par image / photo du QR Code
			s.PreviousStep = s.Step
			s.Step = StepControllerInputQRCode
			return getPromptForStep(s)
		case "0", "0️⃣": // Retour au menu contrôleur
			s.PreviousStep = s.Step
			s.Step = StepControllerActions
			return getPromptForStep(s)
		default:
			return getPromptForStep(s)
		}

	case StepControllerInputBookingRef:
		// Handle controller booking reference input
		if strings.TrimSpace(text) == "" {
			return "✍️ Veuillez saisir la **référence de la réservation** :"
		}

		reference := strings.TrimSpace(text)
		var bookingResult BookingDetailsResult
		var err error

		// Try to find booking by reference
		err = DB.Get(&bookingResult, `
			SELECT b.id, b.booking_reference, b.total_amount, b.status, b.created_at, b.updated_at,
				   j.id as journey_id, j.departure_date, j.departure_time, j.arrival_date, j.arrival_time,
				   r.id as route_id, r.origin, r.destination, r.base_price,
				   o.id as organization_id, o.name as organization_name,
					   GROUP_CONCAT(t.id) AS ticket_ids,
					   GROUP_CONCAT(t.passenger_name) AS passenger_names,
					   MIN(t.passenger_name) AS first_passenger_name,
					   GROUP_CONCAT(t.passenger_phone) AS passenger_phones,
					   GROUP_CONCAT(t.seat_number) AS seat_numbers
			FROM bookings b
			JOIN journeys j ON b.journey_id = j.id
			JOIN routes r ON j.route_id = r.id
			JOIN organizations o ON j.organization_id = o.id
			LEFT JOIN tickets t ON b.id = t.booking_id
			WHERE b.booking_reference = $1
			GROUP BY b.id, j.id, r.id, o.id
		`, reference)

		if err != nil {
			if err == sql.ErrNoRows {
				return "❌ Aucune réservation trouvée pour cette référence.\n\nVeuillez vérifier le numéro de réservation et réessayer, ou tapez *0* pour retourner."
			}
			return fmt.Sprintf("❌ Erreur lors de la recherche de la réservation : %v\n\nTapez *menu* pour retourner au menu principal.", err)
		}

		return processControllerBookingResult(s, bookingResult)

	case StepControllerInputPhone:
		// Handle controller passenger phone input
		if strings.TrimSpace(text) == "" {
			return "📱 Veuillez saisir le **numéro de téléphone du passager** :"
		}

		phoneInput := strings.TrimSpace(text)
		// Normalize the phone number
		normalizedPhone := strings.ReplaceAll(phoneInput, "+", "")
		normalizedPhone = strings.ReplaceAll(normalizedPhone, " ", "")
		normalizedPhone = strings.ReplaceAll(normalizedPhone, "-", "")

		var bookingResult BookingDetailsResult
		err := DB.Get(&bookingResult, `
			SELECT b.id, b.booking_reference, b.total_amount, b.status, b.created_at, b.updated_at,
				   j.id as journey_id, j.departure_date, j.departure_time, j.arrival_date, j.arrival_time,
				   r.id as route_id, r.origin, r.destination, r.base_price,
				   o.id as organization_id, o.name as organization_name,
					   GROUP_CONCAT(t.id) AS ticket_ids,
					   GROUP_CONCAT(t.passenger_name) AS passenger_names,
					   MIN(t.passenger_name) AS first_passenger_name,
					   GROUP_CONCAT(t.passenger_phone) AS passenger_phones,
					   GROUP_CONCAT(t.seat_number) AS seat_numbers
			FROM bookings b
			JOIN journeys j ON b.journey_id = j.id
			JOIN routes r ON j.route_id = r.id
			JOIN organizations o ON j.organization_id = o.id
			JOIN tickets t ON b.id = t.booking_id
			WHERE t.passenger_phone = $1
			GROUP BY b.id, j.id, r.id, o.id
		`, normalizedPhone)

		if err != nil {
			if err == sql.ErrNoRows {
				return "❌ Aucune réservation trouvée pour ce numéro de téléphone.\n\nVeuillez vérifier le numéro et réessayer, ou tapez *0* pour retourner."
			}
			return fmt.Sprintf("❌ Erreur lors de la recherche de la réservation : %v\n\nTapez *menu* pour retourner au menu principal.", err)
		}

		return processControllerBookingResult(s, bookingResult)

	case StepControllerInputQRCode:
		// Handle controller QR code image input or text fallback
		if strings.TrimSpace(text) == "" {
			return "📸 Veuillez **envoyer la photo / l'image du QR code** du billet, ou tapez directement la référence du billet :"
		}

		// Process text input as QR token or booking reference (could be from image decode or manual entry)
		qrInput := strings.TrimSpace(text)

		// Try to find ticket by QR token
		var bookingResult BookingDetailsResult
		err := DB.Get(&bookingResult, `
			SELECT b.id, b.booking_reference, b.total_amount, b.status, b.created_at, b.updated_at,
				   j.id as journey_id, j.departure_date, j.departure_time, j.arrival_date, j.arrival_time,
				   r.id as route_id, r.origin, r.destination, r.base_price,
				   o.id as organization_id, o.name as organization_name,
					   GROUP_CONCAT(t.id) AS ticket_ids,
					   GROUP_CONCAT(t.passenger_name) AS passenger_names,
					   MIN(t.passenger_name) AS first_passenger_name,
					   GROUP_CONCAT(t.passenger_phone) AS passenger_phones,
					   GROUP_CONCAT(t.seat_number) AS seat_numbers
			FROM bookings b
			JOIN journeys j ON b.journey_id = j.id
			JOIN routes r ON j.route_id = r.id
			JOIN organizations o ON j.organization_id = o.id
			LEFT JOIN tickets t ON b.id = t.booking_id
			WHERE t.qr_token = $1
			GROUP BY b.id, j.id, r.id, o.id
		`, qrInput)

		if err != nil {
			// Try as booking reference
			err = DB.Get(&bookingResult, `
				SELECT b.id, b.booking_reference, b.total_amount, b.status, b.created_at, b.updated_at,
					   j.id as journey_id, j.departure_date, j.departure_time, j.arrival_date, j.arrival_time,
					   r.id as route_id, r.origin, r.destination, r.base_price,
					   o.id as organization_id, o.name as organization_name,
						   GROUP_CONCAT(t.id) AS ticket_ids,
						   GROUP_CONCAT(t.passenger_name) AS passenger_names,
						   MIN(t.passenger_name) AS first_passenger_name,
						   GROUP_CONCAT(t.passenger_phone) AS passenger_phones,
						   GROUP_CONCAT(t.seat_number) AS seat_numbers
				FROM bookings b
				JOIN journeys j ON b.journey_id = j.id
				JOIN routes r ON j.route_id = r.id
				JOIN organizations o ON j.organization_id = o.id
				LEFT JOIN tickets t ON b.id = t.booking_id
				WHERE b.booking_reference = $1
				GROUP BY b.id, j.id, r.id, o.id
			`, qrInput)
		}

		if err != nil {
			if err == sql.ErrNoRows {
				return "❌ Aucun billet trouvé pour ce QR code / référence.\n\nVeuillez vérifier et réessayer, ou tapez *0* pour retourner."
			}
			return fmt.Sprintf("❌ Erreur lors de la recherche : %v\n\nTapez *menu* pour retourner au menu principal.", err)
		}

		return processControllerBookingResult(s, bookingResult)

	case StepControllerPaymentReferenceInput:
		// Handle controller payment reference input
		if strings.TrimSpace(text) == "" {
			return "Veuillez saisir le numéro de réservation (Booking Reference) ou le numéro de téléphone du passager :"
		}

		reference := strings.TrimSpace(text)
		var bookingResult BookingDetailsResult
		var err error

		// Try to find booking by reference first
		err = DB.Get(&bookingResult, `
			SELECT b.id, b.booking_reference, b.total_amount, b.status, b.created_at, b.updated_at,
				   j.id as journey_id, j.departure_date, j.departure_time, j.arrival_date, j.arrival_time,
				   r.id as route_id, r.origin, r.destination, r.base_price,
				   o.id as organization_id, o.name as organization_name,
					   GROUP_CONCAT(t.id) AS ticket_ids,
					   GROUP_CONCAT(t.passenger_name) AS passenger_names,
					   MIN(t.passenger_name) AS first_passenger_name,
					   GROUP_CONCAT(t.passenger_phone) AS passenger_phones,
					   GROUP_CONCAT(t.seat_number) AS seat_numbers
			FROM bookings b
			JOIN journeys j ON b.journey_id = j.id
			JOIN routes r ON j.route_id = r.id
			JOIN organizations o ON j.organization_id = o.id
			LEFT JOIN tickets t ON b.id = t.booking_id
			WHERE b.booking_reference = $1
			GROUP BY b.id, j.id, r.id, o.id
		`, reference)

		if err != nil {
			// If not found by reference, try to find by passenger phone
			// First, normalize the phone number (remove +, spaces, etc.)
			normalizedPhone := strings.ReplaceAll(reference, "+", "")
			normalizedPhone = strings.ReplaceAll(normalizedPhone, " ", "")
			normalizedPhone = strings.ReplaceAll(normalizedPhone, "-", "")

			err = DB.Get(&bookingResult, `
				SELECT b.id, b.booking_reference, b.total_amount, b.status, b.created_at, b.updated_at,
					   j.id as journey_id, j.departure_date, j.departure_time, j.arrival_date, j.arrival_time,
					   r.id as route_id, r.origin, r.destination, r.base_price,
					   o.id as organization_id, o.name as organization_name,
						   GROUP_CONCAT(t.id) AS ticket_ids,
						   GROUP_CONCAT(t.passenger_name) AS passenger_names,
						   MIN(t.passenger_name) AS first_passenger_name,
						   GROUP_CONCAT(t.passenger_phone) AS passenger_phones,
						   GROUP_CONCAT(t.seat_number) AS seat_numbers
			FROM bookings b
			JOIN journeys j ON b.journey_id = j.id
			JOIN routes r ON j.route_id = r.id
			JOIN organizations o ON j.organization_id = o.id
			JOIN tickets t ON b.id = t.booking_id
			WHERE t.passenger_phone = $1
			GROUP BY b.id, j.id, r.id, o.id
			`, normalizedPhone)
		}

		if err != nil {
			if err == sql.ErrNoRows {
				return "❌ Aucune réservation en attente trouvée pour cette référence/numéro.\n\nVeuillez vérifier le numéro de réservation ou le numéro de téléphone du passager et réessayer, ou tapez *menu* pour retourner au menu principal."
			}
			return fmt.Sprintf("❌ Erreur lors de la recherche de la réservation : %v\n\nTapez *menu* pour retourner au menu principal.", err)
		}

		// Check if booking is already paid/confirmed
		if bookingResult.Status == "confirmed" || bookingResult.Status == "paid" {
			return fmt.Sprintf("ℹ️ Cette réservation (%s) est déjà confirmée/payée.\n\nTapez *menu* pour retourner au menu principal.", bookingResult.BookingReference)
		}

		// Update booking status to confirmed
		_, err = DB.Exec(`UPDATE bookings SET status = 'confirmed' WHERE id = $1`, bookingResult.ID)
		if err != nil {
			return fmt.Sprintf("❌ Erreur lors de la confirmation de la réservation : %v\n\nTapez *menu* pour retourner au menu principal.", err)
		}

		// Get passenger name for confirmation message (first passenger)
		var passengerName string
		if bookingResult.PassengerNames != "" {
			names := strings.Split(bookingResult.PassengerNames, ",")
			if len(names) > 0 && names[0] != "" {
				passengerName = names[0]
			} else {
				passengerName = "Passager"
			}
		} else {
			passengerName = "Passager"
		}

		// Return to controller actions menu
		s.PreviousStep = s.Step
		s.Step = StepControllerActions

		return fmt.Sprintf("✅ Paiement validé pour le passager %s (Réservation %s) !\n\nSélectionnez une action :\n1️⃣ Valider le paiement\n2️⃣ Faire le check-in", passengerName, bookingResult.BookingReference)

	case StepControllerCheckInConfirm:
		// Handle check-in confirmation
		if strings.TrimSpace(text) == "1" || strings.TrimSpace(text) == "1️⃣" {
			// Process check-in immediately when user confirms
			bookingID := ""
			ticketID := ""
			bookingRef := ""
			passengerName := ""
			seatNumber := ""
			origin := ""
			destination := ""

			if s.Data != nil {
				if id, ok := s.Data["controller_booking_id"].(string); ok {
					bookingID = id
				}
				if tid, ok := s.Data["controller_ticket_id"].(string); ok {
					ticketID = tid
				}
				if ref, ok := s.Data["controller_booking_ref"].(string); ok {
					bookingRef = ref
				}
				if name, ok := s.Data["controller_passenger_name"].(string); ok {
					passengerName = name
				}
				if seat, ok := s.Data["controller_seat_number"].(string); ok {
					seatNumber = seat
				}
				if org, ok := s.Data["controller_origin"].(string); ok {
					origin = org
				}
				if dest, ok := s.Data["controller_destination"].(string); ok {
					destination = dest
				}
			}

			if bookingID == "" && ticketID == "" {
				return "❌ Erreur : aucune réservation sélectionnée. Tapez *menu* pour retourner au menu principal."
			}

			// Update ticket status to checked_in with guard condition on booking status
			var err error
			if ticketID != "" {
				_, err = DB.Exec(`
					UPDATE tickets
					SET status = 'checked_in', checked_in_at = NOW()
					WHERE (booking_id = $1 OR qr_token = $2)
					AND booking_id IN (SELECT id FROM bookings WHERE status = 'confirmed')`,
					bookingID, ticketID)
			} else {
				_, err = DB.Exec(`
					UPDATE tickets
					SET status = 'checked_in', checked_in_at = NOW()
					WHERE booking_id = $1
					AND booking_id IN (SELECT id FROM bookings WHERE status = 'confirmed')`,
					bookingID)
			}
			if err != nil {
				return fmt.Sprintf("❌ Erreur lors du check-in : %v\n\nTapez *menu* pour retourner au menu principal.", err)
			}

			// Build and send success message explicitly
			successMsg := fmt.Sprintf("✅ *Check-in effectué avec succès !*\n\n📌 *Référence* : %s\n👤 *Passager* : %s\n💺 *Siège* : N° %s\n🚌 *Trajet* : %s ➔ %s\n📊 *Statut* : checked_in (Embarqué)\n\nTapez *menu* pour revenir au menu principal.",
				bookingRef, passengerName, seatNumber, origin, destination)

			// Reset to menu
			s.Step = StepMenu
			s.PreviousStep = ""

			return successMsg
		} else if strings.TrimSpace(text) == "2" || strings.TrimSpace(text) == "2️⃣" || strings.ToLower(strings.TrimSpace(text)) == "0" || strings.ToLower(strings.TrimSpace(text)) == "retour" {
			s.PreviousStep = s.Step
			s.Step = StepControllerSelectAction
			// Return controller menu with "Opération annulée" message
			var menu strings.Builder
			menu.WriteString("↩️ *Opération annulée.*\n\n")
			menu.WriteString("Sélectionnez une action :\n")
			menu.WriteString("1️⃣ Valider le paiement\n")
			menu.WriteString("2️⃣ Faire le check-in\n")
			return menu.String()
		}
		return getPromptForStep(s)

		case StepControllerCheckInProcess:
			// This step should no longer be reached directly, but keep as fallback
			return "✅ Check-in déjà traité. Tapez *menu* pour revenir au menu principal."

		case StepControllerCashCheckinConfirm:
			// Handle cash payment + check-in confirmation
			choice := strings.TrimSpace(text)
			switch choice {
			case "1", "1️⃣": // Encaisser le Cash ET valider le Check-in (Recommandé)
				return processCashPaymentAndCheckin(s)
			case "2", "2️⃣": // Encaisser le Cash seulement
				return processCashPaymentOnly(s)
			case "3", "3️⃣", "0", "0️⃣", "retour", "annuler": // Annuler / Retour
				s.PreviousStep = s.Step
				s.Step = StepControllerSelectAction
				var menu strings.Builder
				menu.WriteString("↩️ *Opération annulée.*\n\n")
				menu.WriteString("Sélectionnez une action :\n")
				menu.WriteString("1️⃣ Valider le paiement\n")
				menu.WriteString("2️⃣ Faire le check-in\n")
				return menu.String()
			default:
				return "Choix invalide. Veuillez sélectionner 1, 2 ou 3, ou tapez *menu* pour annuler."
			}

		case StepControllerPayAllConfirm:
			// Handle pay all tickets confirmation
			return processControllerPayAllConfirm(s, text)

		case StepControllerActionChoice:
			// Handle unified action choice for both single ticket and pay all
			return processControllerActionChoice(s, text)

		// Agency creation flow
		case StepAgencyName:
			// Save agency name and ask for manager name
			s.AgencyName = strings.TrimSpace(text)
			if s.AgencyName == "" {
				return "Veuillez entrer un nom d'agence valide."
			}
			s.PreviousStep = s.Step
			s.Step = StepAgencyManagerName
			return getPromptForStep(s)

		case StepAgencyManagerName:
			// Save manager name and ask for address
			s.ManagerName = strings.TrimSpace(text)
			if s.ManagerName == "" {
				return "Veuillez entrer un nom de responsable valide."
			}
			s.PreviousStep = s.Step
			s.Step = StepAgencyAddress
			return getPromptForStep(s)

		case StepAgencyAddress:
			// Save address and ask for contact phone
			s.Address = strings.TrimSpace(text)
			if s.Address == "" {
				return "Veuillez entrer une ville/adresse valide."
			}
			s.PreviousStep = s.Step
			s.Step = StepAgencyContactPhone
			return getPromptForStep(s)

		case StepAgencyContactPhone:
			// Save contact phone and show confirmation
			contactPhone := strings.TrimSpace(text)
			// Validate phone format: should start with 243 and be 12 digits total
			if !strings.HasPrefix(contactPhone, "243") || len(contactPhone) != 12 {
				return "Numéro de téléphone invalide. Format attendu : 243XXXXXXXXX (12 chiffres commençant par 243)"
			}
			if _, err := strconv.Atoi(contactPhone); err != nil {
				return "Numéro de téléphone invalide. Veuillez entrer uniquement des chiffres."
			}
			s.ContactPhone = contactPhone
			s.PreviousStep = s.Step
			s.Step = StepAgencyConfirm
			return getPromptForStep(s)

		case StepAgencyConfirm:
			// Confirm agency creation
			lower := strings.ToLower(strings.TrimSpace(text))
			if lower != "oui" {
				resetSession(phone)
				return "Création d'agence annulée. Tapez *menu* pour revenir au menu principal."
			}

			// Create the agency (organization) in the database
			orgID, err := createAgency(s.AgencyName, s.ManagerName, s.Address, s.ContactPhone)
			if err != nil {
				return fmt.Sprintf("❌ Erreur lors de la création de l'agence : %v\n\nTapez *menu* pour réessayer.", err)
			}

			// Success message
			successMsg := fmt.Sprintf("✅ *Agence créée avec succès !*\n\n🏢 *Nom* : %s\n👔 *Responsable* : %s\n📍 *Adresse* : %s\n📞 *Téléphone* : %s\n🆔 *ID* : %s\n\nL'agence est maintenant disponible pour la gestion des voyages.\n\nTapez *menu* pour revenir au menu principal.", s.AgencyName, s.ManagerName, s.Address, s.ContactPhone, orgID)

			resetSession(phone)
			return successMsg

		default:
			return "Étape inconnue. Tapez *menu* pour revenir au menu principal."
		}
		return ""
}

// processCashPaymentAndCheckin processes cash payment and check-in in one step
func processCashPaymentAndCheckin(s *Session) string {
	bookingID := ""
	ticketID := ""
	bookingRef := ""
	passengerName := ""
	seatNumber := ""
	origin := ""
	destination := ""

	if s.Data != nil {
		if id, ok := s.Data["controller_booking_id"].(string); ok {
			bookingID = id
		}
		if tid, ok := s.Data["controller_ticket_id"].(string); ok {
			ticketID = tid
		}
		if ref, ok := s.Data["controller_booking_ref"].(string); ok {
			bookingRef = ref
		}
		if name, ok := s.Data["controller_passenger_name"].(string); ok {
			passengerName = name
		}
		if seat, ok := s.Data["controller_seat_number"].(string); ok {
			seatNumber = seat
		}
		if org, ok := s.Data["controller_origin"].(string); ok {
			origin = org
		}
		if dest, ok := s.Data["controller_destination"].(string); ok {
			destination = dest
		}
	}

	if bookingID == "" && ticketID == "" {
		return "❌ Erreur : aucune réservation sélectionnée. Tapez *menu* pour retourner au menu principal."
	}

	// Start transaction for atomic operations
	tx, err := DB.Beginx()
	if err != nil {
		return fmt.Sprintf("❌ Erreur lors du début de la transaction : %v\n\nTapez *menu* pour retourner au menu principal.", err)
	}

	// 1. Update booking status to confirmed
	_, err = tx.Exec(`UPDATE bookings SET status = 'confirmed' WHERE id = $1`, bookingID)
	if err != nil {
		tx.Rollback()
		return fmt.Sprintf("❌ Erreur lors de la confirmation de la réservation : %v\n\nTapez *menu* pour retourner au menu principal.", err)
	}

	// 2. Create/validate cash payment record
	// Check if payment already exists for this booking
	var paymentID string
	err = tx.Get(&paymentID, `SELECT id FROM payments WHERE reference = $1`, bookingRef)
	if err == sql.ErrNoRows {
		// Create new cash payment record
		_, err = tx.Exec(`
			INSERT INTO payments (reference, amount, currency, payment_method, status, description)
			VALUES ($1, $2, 'USD', 'CASH', 'SUCCESS', 'Paiement espèces au guichet')
		`, bookingRef+"_cash", 0.0) // Amount will be updated below
		if err != nil {
			tx.Rollback()
			return fmt.Sprintf("❌ Erreur lors de la création du paiement : %v\n\nTapez *menu* pour retourner au menu principal.", err)
		}
		// Update with correct amount from booking
		_, err = tx.Exec(`
			UPDATE payments SET amount = (SELECT total_amount FROM bookings WHERE id = $1)
			WHERE reference = $2
		`, bookingID, bookingRef+"_cash")
		if err != nil {
			tx.Rollback()
			return fmt.Sprintf("❌ Erreur lors de la mise à jour du montant : %v\n\nTapez *menu* pour retourner au menu principal.", err)
		}
	} else if err != nil {
		tx.Rollback()
		return fmt.Sprintf("❌ Erreur lors de la vérification du paiement : %v\n\nTapez *menu* pour retourner au menu principal.", err)
	} else {
		// Payment exists, update to SUCCESS
		_, err = tx.Exec(`UPDATE payments SET status = 'SUCCESS', updated_at = NOW() WHERE id = $1`, paymentID)
		if err != nil {
			tx.Rollback()
			return fmt.Sprintf("❌ Erreur lors de la mise à jour du paiement : %v\n\nTapez *menu* pour retourner au menu principal.", err)
		}
	}

	// 3. Update ticket status to checked_in with guard condition
	if ticketID != "" {
		_, err = tx.Exec(`
			UPDATE tickets
			SET status = 'checked_in', checked_in_at = NOW()
			WHERE (booking_id = $1 OR qr_token = $2)
			AND booking_id IN (SELECT id FROM bookings WHERE status = 'confirmed')`,
			bookingID, ticketID)
	} else {
		_, err = tx.Exec(`
			UPDATE tickets
			SET status = 'checked_in', checked_in_at = NOW()
			WHERE booking_id = $1
			AND booking_id IN (SELECT id FROM bookings WHERE status = 'confirmed')`,
			bookingID)
	}
	if err != nil {
		tx.Rollback()
		return fmt.Sprintf("❌ Erreur lors du check-in : %v\n\nTapez *menu* pour retourner au menu principal.", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Sprintf("❌ Erreur lors de la finalisation : %v\n\nTapez *menu* pour retourner au menu principal.", err)
	}

	// 4. Send WhatsApp notification to passenger
	passengerPhone := ""
	totalAmount := 0.0
	if s.Data != nil {
		if phone, ok := s.Data["controller_passenger_phone"].(string); ok {
			passengerPhone = phone
		}
		if amount, ok := s.Data["controller_total_amount"].(float64); ok {
			totalAmount = amount
		}
	}
	if passengerPhone == "" {
		// Try to get from ticket data
		var phoneStr string
		DB.Get(&phoneStr, `SELECT passenger_phone FROM tickets WHERE booking_id = $1 LIMIT 1`, bookingID)
		passengerPhone = phoneStr
	}
	sendCashValidationNotificationToPassenger(passengerPhone, bookingRef, passengerName, origin, destination, seatNumber, totalAmount)

	// Build success message
	successMsg := fmt.Sprintf("✅ *Encaissement Cash + Check-in effectués avec succès !*\n\n📌 *Référence* : %s\n👤 *Passager* : %s\n💺 *Siège* : N° %s\n🚌 *Trajet* : %s ➔ %s\n💰 *Montant encaissé* : %.2f USD\n📊 *Statut* : confirmed + checked_in (Embarqué)\n\nTapez *menu* pour revenir au menu principal.",
		bookingRef, passengerName, seatNumber, origin, destination, 0.0)

	// Try to get the actual total amount
	if s.Data != nil {
		if amount, ok := s.Data["controller_total_amount"].(float64); ok {
			successMsg = fmt.Sprintf("✅ *Encaissement Cash + Check-in effectués avec succès !*\n\n📌 *Référence* : %s\n👤 *Passager* : %s\n💺 *Siège* : N° %s\n🚌 *Trajet* : %s ➔ %s\n💰 *Montant encaissé* : %.2f USD\n📊 *Statut* : confirmed + checked_in (Embarqué)\n\nTapez *menu* pour revenir au menu principal.",
				bookingRef, passengerName, seatNumber, origin, destination, amount)
		}
	}

	// Reset to menu
	s.Step = StepMenu
	s.PreviousStep = ""

	return successMsg
}

// processCashPaymentOnly processes cash payment without check-in
func processCashPaymentOnly(s *Session) string {
	bookingID := ""
	bookingRef := ""
	passengerName := ""
	seatNumber := ""
	origin := ""
	destination := ""

	if s.Data != nil {
		if id, ok := s.Data["controller_booking_id"].(string); ok {
			bookingID = id
		}
		if ref, ok := s.Data["controller_booking_ref"].(string); ok {
			bookingRef = ref
		}
		if name, ok := s.Data["controller_passenger_name"].(string); ok {
			passengerName = name
		}
		if seat, ok := s.Data["controller_seat_number"].(string); ok {
			seatNumber = seat
		}
		if org, ok := s.Data["controller_origin"].(string); ok {
			origin = org
		}
		if dest, ok := s.Data["controller_destination"].(string); ok {
			destination = dest
		}
	}

	if bookingID == "" {
		return "❌ Erreur : aucune réservation sélectionnée. Tapez *menu* pour retourner au menu principal."
	}

	// Start transaction
	tx, err := DB.Beginx()
	if err != nil {
		return fmt.Sprintf("❌ Erreur lors du début de la transaction : %v\n\nTapez *menu* pour retourner au menu principal.", err)
	}

	// 1. Update booking status to confirmed
	_, err = tx.Exec(`UPDATE bookings SET status = 'confirmed' WHERE id = $1`, bookingID)
	if err != nil {
		tx.Rollback()
		return fmt.Sprintf("❌ Erreur lors de la confirmation de la réservation : %v\n\nTapez *menu* pour retourner au menu principal.", err)
	}

	// 2. Create/validate cash payment record
	var paymentID string
	err = tx.Get(&paymentID, `SELECT id FROM payments WHERE reference = $1`, bookingRef+"_cash")
	if err == sql.ErrNoRows {
		_, err = tx.Exec(`
			INSERT INTO payments (reference, amount, currency, payment_method, status, description)
			VALUES ($1, $2, 'USD', 'CASH', 'SUCCESS', 'Paiement espèces au guichet')
		`, bookingRef+"_cash", 0.0)
		if err != nil {
			tx.Rollback()
			return fmt.Sprintf("❌ Erreur lors de la création du paiement : %v\n\nTapez *menu* pour retourner au menu principal.", err)
		}
		_, err = tx.Exec(`
			UPDATE payments SET amount = (SELECT total_amount FROM bookings WHERE id = $1)
			WHERE reference = $2
		`, bookingID, bookingRef+"_cash")
		if err != nil {
			tx.Rollback()
			return fmt.Sprintf("❌ Erreur lors de la mise à jour du montant : %v\n\nTapez *menu* pour retourner au menu principal.", err)
		}
	} else if err != nil {
		tx.Rollback()
		return fmt.Sprintf("❌ Erreur lors de la vérification du paiement : %v\n\nTapez *menu* pour retourner au menu principal.", err)
	} else {
		_, err = tx.Exec(`UPDATE payments SET status = 'SUCCESS', updated_at = NOW() WHERE id = $1`, paymentID)
		if err != nil {
			tx.Rollback()
			return fmt.Sprintf("❌ Erreur lors de la mise à jour du paiement : %v\n\nTapez *menu* pour retourner au menu principal.", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Sprintf("❌ Erreur lors de la finalisation : %v\n\nTapez *menu* pour retourner au menu principal.", err)
	}

	// Send WhatsApp notification to passenger
	passengerPhone := ""
	totalAmount := 0.0
	if s.Data != nil {
		if phone, ok := s.Data["controller_passenger_phone"].(string); ok {
			passengerPhone = phone
		}
		if amount, ok := s.Data["controller_total_amount"].(float64); ok {
			totalAmount = amount
		}
	}
	if passengerPhone == "" {
		// Try to get from ticket data
		var phoneStr string
		DB.Get(&phoneStr, `SELECT passenger_phone FROM tickets WHERE booking_id = $1 LIMIT 1`, bookingID)
		passengerPhone = phoneStr
	}
	sendCashValidationNotificationToPassenger(passengerPhone, bookingRef, passengerName, origin, destination, seatNumber, totalAmount)

	// Build success message
	successMsg := fmt.Sprintf("✅ *Paiement Cash encaissé avec succès !*\n\n📌 *Référence* : %s\n👤 *Passager* : %s\n💺 *Siège* : N° %s\n🚌 *Trajet* : %s ➔ %s\n💰 *Montant encaissé* : %.2f USD\n📊 *Statut* : confirmed (Payé)\n\nLe passager pourra faire son check-in ultérieurement.\n\nTapez *menu* pour revenir au menu principal.",
		bookingRef, passengerName, seatNumber, origin, destination, 0.0)

	if s.Data != nil {
		if amount, ok := s.Data["controller_total_amount"].(float64); ok {
			successMsg = fmt.Sprintf("✅ *Paiement Cash encaissé avec succès !*\n\n📌 *Référence* : %s\n👤 *Passager* : %s\n💺 *Siège* : N° %s\n🚌 *Trajet* : %s ➔ %s\n💰 *Montant encaissé* : %.2f USD\n📊 *Statut* : confirmed (Payé)\n\nLe passager pourra faire son check-in ultérieurement.\n\nTapez *menu* pour revenir au menu principal.",
				bookingRef, passengerName, seatNumber, origin, destination, amount)
		}
	}

	// Return to controller actions menu
	s.PreviousStep = s.Step
	s.Step = StepControllerActions

	return successMsg
}

// searchBookingForPaymentValidation searches for a booking using the unified query
// and processes it for payment validation
func searchBookingForPaymentValidation(s *Session, input string) string {
	var bookingResult BookingDetailsResult

	// Normalize input for phone matching (extract digits only)
	inputDigits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, strings.TrimPrefix(input, "+"))

	// Compute suffix for flexible matching (last 9 digits or all if shorter)
	suffixLen := 9
	if len(inputDigits) < suffixLen {
		suffixLen = len(inputDigits)
	}
	inputSuffix := inputDigits[len(inputDigits)-suffixLen:]

	err := DB.Get(&bookingResult, `
		SELECT
			b.id AS booking_id,
			b.id AS id,
			b.booking_reference,
			b.status,
			b.status AS payment_status,
			t.id AS ticket_id,
			t.passenger_name,
			t.passenger_phone,
			t.seat_number,
			t.price,
			o.name AS organization_name,
			r.origin,
			r.destination,
			j.departure_date::text AS departure_date,
			j.departure_time::text AS departure_time
		FROM tickets t
		JOIN bookings b ON t.booking_id = b.id
		JOIN journeys j ON t.journey_id = j.id
		JOIN routes r ON j.route_id = r.id
		JOIN organizations o ON j.organization_id = o.id
		WHERE b.booking_reference = $1
		   OR t.passenger_phone ILIKE '%' || $1 || '%'
		   OR REGEXP_REPLACE(t.passenger_phone, '\D', '', 'g') = $2
		   OR REGEXP_REPLACE(t.passenger_phone, '\D', '', 'g') LIKE '%' || $3
		   OR t.passenger_name ILIKE '%' || $1 || '%'
		   OR t.qr_token = $1
		LIMIT 1
	`, input, inputDigits, inputSuffix)

	if err != nil {
		if err == sql.ErrNoRows {
			return "❌ Aucune réservation trouvée avec ces informations.\n\nVeuillez vérifier et réessayer, ou tapez *menu* pour annuler."
		}
		return fmt.Sprintf("❌ Erreur lors de la recherche : %v\n\nTapez *menu* pour annuler.", err)
	}

	// Process the found booking for payment validation
	return processControllerBookingResult(s, bookingResult)
}

// searchBookingsByPhoneForController searches for ALL bookings by passenger phone
// Returns separate lists for unpaid (cash pending) and paid/confirmed bookings
func searchBookingsByPhoneForController(s *Session, phoneInput string) string {
	// Normalize phone input - keep only digits
	inputDigits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, strings.TrimPrefix(phoneInput, "+"))

	// Log the search for debugging
	log.Printf("[CONTROLLER_SEARCH] Phone input: %s, Normalized: %s", phoneInput, inputDigits)

	// Query for UNPAID bookings (status=pending, payment_method=cash)
	// Uses flexible matching: compares normalized digits of passenger_phone with normalized input
	var unpaidResults []BookingDetailsResult
	err := DB.Select(&unpaidResults, `
		SELECT
			b.id AS booking_id,
			b.id AS id,
			b.booking_reference,
			b.total_amount,
			b.status,
			b.status AS payment_status,
			b.payment_method,
			t.id AS ticket_id,
			t.passenger_name,
			t.passenger_phone,
			t.seat_number,
			t.price,
			o.name AS organization_name,
			r.origin,
			r.destination,
			j.departure_date::text AS departure_date,
			j.departure_time::text AS departure_time
		FROM tickets t
		JOIN bookings b ON t.booking_id = b.id
		JOIN journeys j ON t.journey_id = j.id
		JOIN routes r ON j.route_id = r.id
		JOIN organizations o ON j.organization_id = o.id
		WHERE b.status = 'pending'
		  AND b.payment_method = 'cash'
		  AND (
			-- Normalized exact match or suffix/partial match
			REGEXP_REPLACE(t.passenger_phone, '\D', '', 'g') = $1
			OR REGEXP_REPLACE(t.passenger_phone, '\D', '', 'g') LIKE '%' || $1
			OR REGEXP_REPLACE(t.passenger_phone, '\D', '', 'g') LIKE $1 || '%'
		  )
		ORDER BY b.created_at DESC
		LIMIT 20
	`, inputDigits)

	if err != nil {
		log.Printf("[CONTROLLER_SEARCH] Unpaid query error: %v", err)
		return fmt.Sprintf("❌ Erreur lors de la recherche : %v\n\nTapez *menu* pour annuler.", err)
	}

	// Query for PAID/CONFIRMED bookings (status=confirmed or paid)
	var paidResults []BookingDetailsResult
	err = DB.Select(&paidResults, `
		SELECT
			b.id AS booking_id,
			b.id AS id,
			b.booking_reference,
			b.total_amount,
			b.status,
			b.status AS payment_status,
			b.payment_method,
			t.id AS ticket_id,
			t.passenger_name,
			t.passenger_phone,
			t.seat_number,
			t.price,
			o.name AS organization_name,
			r.origin,
			r.destination,
			j.departure_date::text AS departure_date,
			j.departure_time::text AS departure_time
		FROM tickets t
		JOIN bookings b ON t.booking_id = b.id
		JOIN journeys j ON t.journey_id = j.id
		JOIN routes r ON j.route_id = r.id
		JOIN organizations o ON j.organization_id = o.id
		WHERE b.status IN ('confirmed', 'paid')
		  AND (
			-- Normalized exact match or suffix/partial match
			REGEXP_REPLACE(t.passenger_phone, '\D', '', 'g') = $1
			OR REGEXP_REPLACE(t.passenger_phone, '\D', '', 'g') LIKE '%' || $1
			OR REGEXP_REPLACE(t.passenger_phone, '\D', '', 'g') LIKE $1 || '%'
		  )
		ORDER BY b.created_at DESC
		LIMIT 20
	`, inputDigits)

	if err != nil {
		log.Printf("[CONTROLLER_SEARCH] Paid query error: %v", err)
		return fmt.Sprintf("❌ Erreur lors de la recherche : %v\n\nTapez *menu* pour annuler.", err)
	}

	// Store results in session for selection
	if s.Data == nil {
		s.Data = make(map[string]interface{})
	}
	s.Data["controller_unpaid_results"] = unpaidResults
	s.Data["controller_paid_results"] = paidResults
	s.Data["controller_search_type"] = "phone"

	// Format the two lists using card-based layout grouped by booking reference
	var b strings.Builder

	// Helper function to format departure time
	formatDeparture := func(dateStr, timeStr string) string {
		if dateStr == "" || timeStr == "" {
			return "N/A"
		}
		// Format: "2026-10-23" + "06:30:00" -> "23/10 à 06h30"
		if len(dateStr) >= 10 && len(timeStr) >= 5 {
			day := dateStr[8:10]
			month := dateStr[5:7]
			hour := timeStr[0:5]
			return fmt.Sprintf("%s/%s à %sh", day, month, hour)
		}
		return dateStr + " " + timeStr
	}

	// Helper to convert number to emoji (1️⃣, 2️⃣, etc.)
	numberToEmoji := func(n int) string {
		emojis := []string{"0️⃣", "1️⃣", "2️⃣", "3️⃣", "4️⃣", "5️⃣", "6️⃣", "7️⃣", "8️⃣", "9️⃣", "🔟"}
		if n >= 0 && n < len(emojis) {
			return emojis[n]
		}
		// For numbers > 10, build from digits
		digits := strconv.Itoa(n)
		var result strings.Builder
		for _, d := range digits {
			if d >= '0' && d <= '9' {
				result.WriteString(emojis[d-'0'])
			}
		}
		return result.String()
	}

	// Helper to group results by booking reference and render cards
	renderBookingCards := func(results []BookingDetailsResult, isPaid bool, startNum int) (string, int) {
		if len(results) == 0 {
			return "", startNum
		}

		// Group by booking reference
		type groupedBooking struct {
			BookingRef      string
			TotalAmount     float64
			AgencyName      string
			Origin          string
			Destination     string
			DepartureDate   string
			DepartureTime   string
			Tickets         []BookingDetailsResult
			GroupIndex      int // Index for "Pay All" option (T1, T2, etc.)
		}

		bookingMap := make(map[string]*groupedBooking)
		bookingOrder := []string{}

		for _, result := range results {
			if _, exists := bookingMap[result.BookingReference]; !exists {
				bookingMap[result.BookingReference] = &groupedBooking{
					BookingRef:    result.BookingReference,
					TotalAmount:   result.TotalAmount,
					AgencyName:    result.OrganizationName,
					Origin:        result.Origin,
					Destination:   result.Destination,
					DepartureDate: result.DepartureDate,
					DepartureTime: result.DepartureTime,
					Tickets:       []BookingDetailsResult{},
					GroupIndex:    len(bookingOrder) + 1, // 1-based index for T1, T2, etc.
				}
				bookingOrder = append(bookingOrder, result.BookingReference)
			}
			bookingMap[result.BookingReference].Tickets = append(bookingMap[result.BookingReference].Tickets, result)
		}

		var cardBuilder strings.Builder
		num := startNum

		for _, ref := range bookingOrder {
			gb := bookingMap[ref]
			departureStr := formatDeparture(gb.DepartureDate, gb.DepartureTime)

			cardBuilder.WriteString(fmt.Sprintf("📌 *Réservation* : %s\n", gb.BookingRef))
			cardBuilder.WriteString(fmt.Sprintf("🏢 *Agence* : %s\n", gb.AgencyName))
			cardBuilder.WriteString(fmt.Sprintf("🚌 *Trajet* : %s ➔ %s\n", gb.Origin, gb.Destination))
			cardBuilder.WriteString(fmt.Sprintf("📅 *Départ* : %s\n", departureStr))
			cardBuilder.WriteString(fmt.Sprintf("💰 *Total Groupe* : %.2f USD\n\n", gb.TotalAmount))

			cardBuilder.WriteString("👥 *Passagers / Sièges* :\n")
			for i, ticket := range gb.Tickets {
				seatNum := ticket.SeatNumber
				if seatNum == "" {
					seatNum = "N/A"
				}
				passengerName := ticket.PassengerName
				if passengerName == "" {
					passengerName = "Passager"
				}

				actionLabel := "valider"
				actionPrefix := ""
				if isPaid {
					actionLabel = "Check-in"
					actionPrefix = "P"
				}
				emojiNum := numberToEmoji(i + 1)
				cardBuilder.WriteString(fmt.Sprintf("  • %s 👤 %s — 💺 Siège %s (💵 %.2f USD) ➔ Tapez **%s%d** pour %s\n",
					emojiNum, passengerName, seatNum, ticket.Price, actionPrefix, num, actionLabel))
				num++
			}

			// Add "Pay All" option for unpaid bookings only
			if !isPaid {
				payAllEmoji := numberToEmoji(gb.GroupIndex)
				cardBuilder.WriteString(fmt.Sprintf("\n⚡ *Action Rapide* :\n"))
				cardBuilder.WriteString(fmt.Sprintf("  • %s Tapez **T%d** pour payer TOUTE la réservation (%.2f USD)\n",
					payAllEmoji, gb.GroupIndex, gb.TotalAmount))
			}
			cardBuilder.WriteString("\n")
		}

		return cardBuilder.String(), num
	}

	// UNPAID list - Cash pending
	b.WriteString("🔴 *BILLETS IMPAYÉS (Paiement Cash en attente)*\n\n")
	unpaidCards, nextNum := renderBookingCards(unpaidResults, false, 1)
	if unpaidCards == "" {
		b.WriteString("  Aucun billet impayé trouvé.\n\n")
	} else {
		b.WriteString(unpaidCards)
	}

	// PAID list - Confirmed/Paid
	b.WriteString("🟢 *BILLETS PAYÉS / CONFIRMÉS*\n\n")
	paidCards, _ := renderBookingCards(paidResults, true, nextNum)
	if paidCards == "" {
		b.WriteString("  Aucun billet payé trouvé.\n\n")
	} else {
		b.WriteString(paidCards)
	}

	if len(unpaidResults) == 0 && len(paidResults) == 0 {
		b.WriteString(fmt.Sprintf("❌ Aucune réservation trouvée pour le numéro : *%s*\n\n", phoneInput))
		b.WriteString("Numéro recherché (normalisé) : " + inputDigits + "\n\n")
		b.WriteString("Veuillez vérifier le numéro et réessayer, ou tapez *menu* pour annuler.")
		return b.String()
	}

	b.WriteString("Tapez le numéro pour sélectionner un billet :\n")
	b.WriteString("• Pour les impayés : tapez le numéro (1, 2, ...)\n")
	b.WriteString("• Pour les payés : tapez P + numéro (ex: P1, P2)\n")
	b.WriteString("• Pour payer TOUT un groupe : tapez T + numéro (ex: T1, T2)\n\n")
	b.WriteString("0️⃣0️⃣ Précédent\n")
	b.WriteString("0️⃣1️⃣ Suivant\n")

	return b.String()
}

// handleControllerPhoneSelection handles user selection from the phone search results
func handleControllerPhoneSelection(s *Session, input string) string {
	input = strings.TrimSpace(strings.ToUpper(input))

	unpaidResults, ok1 := s.Data["controller_unpaid_results"].([]BookingDetailsResult)
	paidResults, ok2 := s.Data["controller_paid_results"].([]BookingDetailsResult)

	if !ok1 || !ok2 {
		// Results not in session, restart search
		delete(s.Data, "controller_unpaid_results")
		delete(s.Data, "controller_paid_results")
		delete(s.Data, "controller_search_type")
		return searchBookingsByPhoneForController(s, input)
	}

	// Handle navigation
	if input == "00" || input == "0️⃣0️⃣" || strings.ToLower(input) == "precedent" {
		s.PreviousStep = s.Step
		s.Step = StepControllerValidatePayment
		return getPromptForStep(s)
	}
	if input == "01" || input == "0️⃣1️⃣" || strings.ToLower(input) == "suivant" {
		return "Navigation suivante non disponible pour cette recherche.\n\nTapez *menu* pour retourner au menu principal."
	}

	// Check if selection is for "Pay All" (T prefix)
	isPayAllSelection := strings.HasPrefix(input, "T")
	if isPayAllSelection {
		selectionStr := strings.TrimPrefix(input, "T")
		groupIndex, err := strconv.Atoi(selectionStr)
		if err != nil || groupIndex < 1 {
			return "❌ Sélection invalide. Tapez T + numéro du groupe (ex: T1), ou *menu* pour annuler."
		}
		return handleControllerPayAllSelection(s, groupIndex, unpaidResults)
	}

	// Check if selection is for paid list (P prefix)
	isPaidSelection := strings.HasPrefix(input, "P")
	selectionStr := strings.TrimPrefix(input, "P")
	selectionNum, err := strconv.Atoi(selectionStr)
	if err != nil || selectionNum < 1 {
		return "❌ Sélection invalide. Tapez le numéro du billet (ex: 1 ou P1), ou T + groupe (ex: T1), ou *menu* pour annuler."
	}

	var selectedResult *BookingDetailsResult
	if isPaidSelection {
		if selectionNum > len(paidResults) {
			return fmt.Sprintf("❌ Numéro invalide. Il y a %d billet(s) payé(s).\n\nTapez *menu* pour annuler.", len(paidResults))
		}
		selectedResult = &paidResults[selectionNum-1]
	} else {
		if selectionNum > len(unpaidResults) {
			return fmt.Sprintf("❌ Numéro invalide. Il y a %d billet(s) impayé(s).\n\nTapez *menu* pour annuler.", len(unpaidResults))
		}
		selectedResult = &unpaidResults[selectionNum-1]
	}

	// Clear search results from session
	delete(s.Data, "controller_unpaid_results")
	delete(s.Data, "controller_paid_results")
	delete(s.Data, "controller_search_type")

	// Process the selected booking
	return processControllerBookingResult(s, *selectedResult)
}

// handleControllerPayAllSelection handles the "Pay All" selection for a booking group
// Stores ticket IDs in session and redirects to unified action sub-menu
func handleControllerPayAllSelection(s *Session, groupIndex int, unpaidResults []BookingDetailsResult) string {
	// Group unpaid results by booking reference to find the target group
	bookingMap := make(map[string][]BookingDetailsResult)
	bookingOrder := []string{}

	for _, result := range unpaidResults {
		if _, exists := bookingMap[result.BookingReference]; !exists {
			bookingOrder = append(bookingOrder, result.BookingReference)
		}
		bookingMap[result.BookingReference] = append(bookingMap[result.BookingReference], result)
	}

	if groupIndex > len(bookingOrder) {
		return fmt.Sprintf("❌ Groupe invalide. Il y a %d groupe(s) de réservation(s).\n\nTapez *menu* pour annuler.", len(bookingOrder))
	}

	targetRef := bookingOrder[groupIndex-1]
	tickets := bookingMap[targetRef]

	if len(tickets) == 0 {
		return "❌ Aucun billet dans ce groupe.\n\nTapez *menu* pour annuler."
	}

	// Get the first ticket to extract booking info
	firstTicket := tickets[0]
	bookingID := firstTicket.BookingID
	bookingRef := firstTicket.BookingReference
	passengerName := firstTicket.PassengerName
	origin := firstTicket.Origin
	destination := firstTicket.Destination
	totalAmount := firstTicket.TotalAmount

	// Store all ticket IDs in session for processing
	ticketIDs := make([]string, len(tickets))
	passengerPhones := make([]string, len(tickets))
	passengerNames := make([]string, len(tickets))
	seatNumbers := make([]string, len(tickets))

	for i, t := range tickets {
		ticketIDs[i] = t.TicketID
		passengerPhones[i] = t.PassengerPhone
		passengerNames[i] = t.PassengerName
		seatNumbers[i] = t.SeatNumber
	}

	// Store in session for the payment processing
	if s.Data == nil {
		s.Data = make(map[string]interface{})
	}
	s.Data["controller_booking_id"] = bookingID
	s.Data["controller_booking_ref"] = bookingRef
	s.Data["controller_passenger_name"] = passengerName
	s.Data["controller_origin"] = origin
	s.Data["controller_destination"] = destination
	s.Data["controller_total_amount"] = totalAmount
	s.Data["controller_pay_all"] = true
	s.Data["controller_ticket_ids"] = ticketIDs
	s.Data["controller_passenger_phones"] = passengerPhones
	s.Data["controller_passenger_names"] = passengerNames
	s.Data["controller_seat_numbers"] = seatNumbers

	// Clear search results
	delete(s.Data, "controller_unpaid_results")
	delete(s.Data, "controller_paid_results")
	delete(s.Data, "controller_search_type")

	// Show unified action sub-menu
	s.PreviousStep = s.Step
	s.Step = StepControllerActionChoice

	var b strings.Builder
	b.WriteString(fmt.Sprintf("💳 *Action requise pour Réservation %s*\n\n", bookingRef))
	b.WriteString(fmt.Sprintf("👥 *Groupe* : %d billet(s)\n", len(tickets)))
	b.WriteString(fmt.Sprintf("🚌 *Trajet* : %s ➔ %s\n", origin, destination))
	b.WriteString(fmt.Sprintf("📅 *Départ* : %s\n", formatDate(firstTicket.DepartureDate)))
	b.WriteString(fmt.Sprintf("\n💰 *Montant total* : %.2f USD\n\n", totalAmount))

	b.WriteString("📋 *Billets concernés* :\n")
	for i, t := range tickets {
		name := t.PassengerName
		if name == "" {
			name = "Passager"
		}
		seat := t.SeatNumber
		if seat == "" {
			seat = "N/A"
		}
		b.WriteString(fmt.Sprintf("  %d️⃣ %s — 💺 Siège %s (💵 %.2f USD)\n", i+1, name, seat, t.Price))
	}

	b.WriteString("\nSélectionnez une action :\n")
	b.WriteString("1️⃣ Encaisser le Cash ET valider le Check-in (Recommandé)\n")
	b.WriteString("2️⃣ Encaisser le Cash seulement\n")
	b.WriteString("3️⃣ Annuler / Retour\n")

	return b.String()
}

// processControllerPayAllConfirm handles the confirmation for paying all tickets in a booking
func processControllerPayAllConfirm(s *Session, input string) string {
	input = strings.TrimSpace(input)
	if input == "1" || input == "1️⃣" {
		return processCashPaymentAllTickets(s)
	}
	// Cancel - go back to search
	s.Step = StepControllerValidatePayment
	return getPromptForStep(s)
}

// processCashPaymentAllTickets processes cash payment for ALL tickets in a booking
func processCashPaymentAllTickets(s *Session) string {
	bookingID := ""
	bookingRef := ""
	origin := ""
	destination := ""
	totalAmount := 0.0
	var ticketIDs []string
	var passengerPhones []string
	var passengerNames []string
	var seatNumbers []string

	if s.Data != nil {
		if id, ok := s.Data["controller_booking_id"].(string); ok {
			bookingID = id
		}
		if ref, ok := s.Data["controller_booking_ref"].(string); ok {
			bookingRef = ref
		}
		if org, ok := s.Data["controller_origin"].(string); ok {
			origin = org
		}
		if dest, ok := s.Data["controller_destination"].(string); ok {
			destination = dest
		}
		if amount, ok := s.Data["controller_total_amount"].(float64); ok {
			totalAmount = amount
		}
		if ids, ok := s.Data["controller_ticket_ids"].([]string); ok {
			ticketIDs = ids
		}
		if phones, ok := s.Data["controller_passenger_phones"].([]string); ok {
			passengerPhones = phones
		}
		if names, ok := s.Data["controller_passenger_names"].([]string); ok {
			passengerNames = names
		}
		if seats, ok := s.Data["controller_seat_numbers"].([]string); ok {
			seatNumbers = seats
		}
	}

	if bookingID == "" || len(ticketIDs) == 0 {
		return "❌ Erreur : aucune réservation sélectionnée. Tapez *menu* pour retourner au menu principal."
	}

	// Start transaction for atomic operations
	tx, err := DB.Beginx()
	if err != nil {
		return fmt.Sprintf("❌ Erreur lors du début de la transaction : %v\n\nTapez *menu* pour retourner au menu principal.", err)
	}

	// 1. Update booking status to confirmed
	_, err = tx.Exec(`UPDATE bookings SET status = 'confirmed' WHERE id = $1`, bookingID)
	if err != nil {
		tx.Rollback()
		return fmt.Sprintf("❌ Erreur lors de la confirmation de la réservation : %v\n\nTapez *menu* pour retourner au menu principal.", err)
	}

	// 2. Create/validate cash payment record
	var paymentID string
	err = tx.Get(&paymentID, `SELECT id FROM payments WHERE reference = $1`, bookingRef+"_cash")
	if err == sql.ErrNoRows {
		// Create new cash payment record
		_, err = tx.Exec(`
			INSERT INTO payments (reference, amount, currency, payment_method, status, description)
			VALUES ($1, $2, 'USD', 'CASH', 'SUCCESS', 'Paiement espèces au guichet (groupe)')
		`, bookingRef+"_cash", totalAmount)
		if err != nil {
			tx.Rollback()
			return fmt.Sprintf("❌ Erreur lors de la création du paiement : %v\n\nTapez *menu* pour retourner au menu principal.", err)
		}
	} else if err != nil {
		tx.Rollback()
		return fmt.Sprintf("❌ Erreur lors de la vérification du paiement : %v\n\nTapez *menu* pour retourner au menu principal.", err)
	} else {
		// Payment exists, update to SUCCESS
		_, err = tx.Exec(`UPDATE payments SET status = 'SUCCESS', updated_at = NOW() WHERE id = $1`, paymentID)
		if err != nil {
			tx.Rollback()
			return fmt.Sprintf("❌ Erreur lors de la mise à jour du paiement : %v\n\nTapez *menu* pour retourner au menu principal.", err)
		}
	}

	// 3. Update ALL tickets status to checked_in with guard condition
	for _, ticketID := range ticketIDs {
		_, err = tx.Exec(`
			UPDATE tickets
			SET status = 'checked_in', checked_in_at = NOW(), payment_validated = TRUE
			WHERE id = $1
			AND booking_id IN (SELECT id FROM bookings WHERE status = 'confirmed')`,
			ticketID)
		if err != nil {
			tx.Rollback()
			return fmt.Sprintf("❌ Erreur lors du check-in : %v\n\nTapez *menu* pour retourner au menu principal.", err)
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Sprintf("❌ Erreur lors de la finalisation : %v\n\nTapez *menu* pour retourner au menu principal.", err)
	}

	// 4. Send WhatsApp notification to ALL passengers
	for i, phone := range passengerPhones {
		if phone != "" {
			name := passengerNames[i]
			if name == "" {
				name = "Passager"
			}
			seat := seatNumbers[i]
			if seat == "" {
				seat = "N/A"
			}
			sendCashValidationNotificationToPassenger(phone, bookingRef, name, origin, destination, seat, totalAmount)
		}
	}

	// Build success message
	successMsg := fmt.Sprintf("✅ *Paiement Groupe encaissé avec succès !*\n\n📌 *Référence* : %s\n👥 *Nombre de billets* : %d\n🚌 *Trajet* : %s ➔ %s\n💰 *Montant total encaissé* : %.2f USD\n📊 *Statut* : confirmed + checked_in (Tous embarqués)\n\nTapez *menu* pour revenir au menu principal.",
		bookingRef, len(ticketIDs), origin, destination, totalAmount)

	// Reset to menu
	s.Step = StepMenu
	s.PreviousStep = ""

	// Clean up session data
	delete(s.Data, "controller_pay_all")
	delete(s.Data, "controller_ticket_ids")
	delete(s.Data, "controller_passenger_phones")
	delete(s.Data, "controller_passenger_names")
	delete(s.Data, "controller_seat_numbers")

	return successMsg
}

// processControllerActionChoice handles the unified action sub-menu after ticket/group selection
// Shows 3 options: Cash+Checkin, Cash Only, Cancel
func processControllerActionChoice(s *Session, input string) string {
	input = strings.TrimSpace(input)

	// Check if this is a group payment (pay all) or single ticket
	isPayAll := false
	if s.Data != nil {
		if val, ok := s.Data["controller_pay_all"].(bool); ok {
			isPayAll = val
		}
	}

	switch input {
	case "1", "1️⃣": // Encaisser le Cash ET valider le Check-in (Recommandé)
		if isPayAll {
			return processCashPaymentAndCheckinUnified(s, true)
		}
		return processCashPaymentAndCheckinUnified(s, false)

	case "2", "2️⃣": // Encaisser le Cash seulement
		if isPayAll {
			return processCashPaymentOnlyUnified(s, true)
		}
		return processCashPaymentOnlyUnified(s, false)

	case "3", "3️⃣", "0", "0️⃣", "retour", "annuler": // Annuler / Retour
		// Go back to controller action menu
		s.PreviousStep = s.Step
		s.Step = StepControllerSelectAction
		var menu strings.Builder
		menu.WriteString("↩️ *Opération annulée.*\n\n")
		menu.WriteString("Sélectionnez une action :\n")
		menu.WriteString("1️⃣ Valider le paiement\n")
		menu.WriteString("2️⃣ Faire le check-in\n")
		return menu.String()

	default:
		return "Choix invalide. Veuillez sélectionner 1, 2 ou 3, ou tapez *menu* pour annuler."
	}
}

// processCashPaymentAndCheckinUnified processes cash payment and check-in in one step
// Handles both single ticket (isPayAll=false) and group tickets (isPayAll=true)
func processCashPaymentAndCheckinUnified(s *Session, isPayAll bool) string {
	bookingID := ""
	bookingRef := ""
	passengerName := ""
	origin := ""
	destination := ""
	totalAmount := 0.0
	var ticketIDs []string
	var passengerPhones []string
	var passengerNames []string
	var seatNumbers []string

	if s.Data != nil {
		if id, ok := s.Data["controller_booking_id"].(string); ok {
			bookingID = id
		}
		if ref, ok := s.Data["controller_booking_ref"].(string); ok {
			bookingRef = ref
		}
		if name, ok := s.Data["controller_passenger_name"].(string); ok {
			passengerName = name
		}
		if org, ok := s.Data["controller_origin"].(string); ok {
			origin = org
		}
		if dest, ok := s.Data["controller_destination"].(string); ok {
			destination = dest
		}
		if amount, ok := s.Data["controller_total_amount"].(float64); ok {
			totalAmount = amount
		}

		if isPayAll {
			if ids, ok := s.Data["controller_ticket_ids"].([]string); ok {
				ticketIDs = ids
			}
			if phones, ok := s.Data["controller_passenger_phones"].([]string); ok {
				passengerPhones = phones
			}
			if names, ok := s.Data["controller_passenger_names"].([]string); ok {
				passengerNames = names
			}
			if seats, ok := s.Data["controller_seat_numbers"].([]string); ok {
				seatNumbers = seats
			}
		} else {
			// Single ticket
			if tid, ok := s.Data["controller_ticket_id"].(string); ok {
				ticketIDs = []string{tid}
			}
			if phone, ok := s.Data["controller_passenger_phone"].(string); ok {
				passengerPhones = []string{phone}
			}
			if seat, ok := s.Data["controller_seat_number"].(string); ok {
				seatNumbers = []string{seat}
			}
			passengerNames = []string{passengerName}
		}
	}

	if bookingID == "" || len(ticketIDs) == 0 {
		return "❌ Erreur : aucune réservation sélectionnée. Tapez *menu* pour retourner au menu principal."
	}

	// Start transaction for atomic operations
	tx, err := DB.Beginx()
	if err != nil {
		return fmt.Sprintf("❌ Erreur lors du début de la transaction : %v\n\nTapez *menu* pour retourner au menu principal.", err)
	}

	// 1. Update booking status to confirmed
	_, err = tx.Exec(`UPDATE bookings SET status = 'confirmed' WHERE id = $1`, bookingID)
	if err != nil {
		tx.Rollback()
		return fmt.Sprintf("❌ Erreur lors de la confirmation de la réservation : %v\n\nTapez *menu* pour retourner au menu principal.", err)
	}

	// 2. Create/validate cash payment record
	var paymentID string
	err = tx.Get(&paymentID, `SELECT id FROM payments WHERE reference = $1`, bookingRef+"_cash")
	if err == sql.ErrNoRows {
		// Create new cash payment record
		_, err = tx.Exec(`
			INSERT INTO payments (reference, amount, currency, payment_method, status, description)
			VALUES ($1, $2, 'USD', 'CASH', 'SUCCESS', 'Paiement espèces au guichet')
		`, bookingRef+"_cash", totalAmount)
		if err != nil {
			tx.Rollback()
			return fmt.Sprintf("❌ Erreur lors de la création du paiement : %v\n\nTapez *menu* pour retourner au menu principal.", err)
		}
	} else if err != nil {
		tx.Rollback()
		return fmt.Sprintf("❌ Erreur lors de la vérification du paiement : %v\n\nTapez *menu* pour retourner au menu principal.", err)
	} else {
		// Payment exists, update to SUCCESS
		_, err = tx.Exec(`UPDATE payments SET status = 'SUCCESS', updated_at = NOW() WHERE id = $1`, paymentID)
		if err != nil {
			tx.Rollback()
			return fmt.Sprintf("❌ Erreur lors de la mise à jour du paiement : %v\n\nTapez *menu* pour retourner au menu principal.", err)
		}
	}

	// 3. Update ALL tickets status to checked_in with guard condition
	for _, ticketID := range ticketIDs {
		_, err = tx.Exec(`
			UPDATE tickets
			SET status = 'checked_in', checked_in_at = NOW(), payment_validated = TRUE
			WHERE id = $1
			AND booking_id IN (SELECT id FROM bookings WHERE status = 'confirmed')`,
			ticketID)
		if err != nil {
			tx.Rollback()
			return fmt.Sprintf("❌ Erreur lors du check-in : %v\n\nTapez *menu* pour retourner au menu principal.", err)
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Sprintf("❌ Erreur lors de la finalisation : %v\n\nTapez *menu* pour retourner au menu principal.", err)
	}

	// 4. Send WhatsApp notification to ALL passengers
	for i, phone := range passengerPhones {
		if phone != "" {
			name := passengerNames[i]
			if name == "" {
				name = "Passager"
			}
			seat := seatNumbers[i]
			if seat == "" {
				seat = "N/A"
			}
			sendCashValidationNotificationToPassenger(phone, bookingRef, name, origin, destination, seat, totalAmount)
		}
	}

	// Build success message
	var successMsg string
	if isPayAll {
		successMsg = fmt.Sprintf("✅ *Encaissement Cash + Check-in Groupe effectués avec succès !*\n\n📌 *Référence* : %s\n👥 *Nombre de billets* : %d\n🚌 *Trajet* : %s ➔ %s\n💰 *Montant total encaissé* : %.2f USD\n📊 *Statut* : confirmed + checked_in (Tous embarqués)\n\nTapez *menu* pour revenir au menu principal.",
			bookingRef, len(ticketIDs), origin, destination, totalAmount)
	} else {
		seat := ""
		if len(seatNumbers) > 0 {
			seat = seatNumbers[0]
		}
		successMsg = fmt.Sprintf("✅ *Encaissement Cash + Check-in effectués avec succès !*\n\n📌 *Référence* : %s\n👤 *Passager* : %s\n💺 *Siège* : N° %s\n🚌 *Trajet* : %s ➔ %s\n💰 *Montant encaissé* : %.2f USD\n📊 *Statut* : confirmed + checked_in (Embarqué)\n\nTapez *menu* pour revenir au menu principal.",
			bookingRef, passengerName, seat, origin, destination, totalAmount)
	}

	// Reset to menu
	s.Step = StepMenu
	s.PreviousStep = ""

	// Clean up session data
	delete(s.Data, "controller_pay_all")
	delete(s.Data, "controller_ticket_ids")
	delete(s.Data, "controller_passenger_phones")
	delete(s.Data, "controller_passenger_names")
	delete(s.Data, "controller_seat_numbers")
	delete(s.Data, "controller_selected_ticket_id")

	return successMsg
}

// processCashPaymentOnlyUnified processes cash payment without check-in
// Handles both single ticket (isPayAll=false) and group tickets (isPayAll=true)
func processCashPaymentOnlyUnified(s *Session, isPayAll bool) string {
	bookingID := ""
	bookingRef := ""
	passengerName := ""
	origin := ""
	destination := ""
	totalAmount := 0.0
	var ticketIDs []string
	var passengerPhones []string
	var passengerNames []string
	var seatNumbers []string

	if s.Data != nil {
		if id, ok := s.Data["controller_booking_id"].(string); ok {
			bookingID = id
		}
		if ref, ok := s.Data["controller_booking_ref"].(string); ok {
			bookingRef = ref
		}
		if name, ok := s.Data["controller_passenger_name"].(string); ok {
			passengerName = name
		}
		if org, ok := s.Data["controller_origin"].(string); ok {
			origin = org
		}
		if dest, ok := s.Data["controller_destination"].(string); ok {
			destination = dest
		}
		if amount, ok := s.Data["controller_total_amount"].(float64); ok {
			totalAmount = amount
		}

		if isPayAll {
			if ids, ok := s.Data["controller_ticket_ids"].([]string); ok {
				ticketIDs = ids
			}
			if phones, ok := s.Data["controller_passenger_phones"].([]string); ok {
				passengerPhones = phones
			}
			if names, ok := s.Data["controller_passenger_names"].([]string); ok {
				passengerNames = names
			}
			if seats, ok := s.Data["controller_seat_numbers"].([]string); ok {
				seatNumbers = seats
			}
		} else {
			// Single ticket
			if tid, ok := s.Data["controller_ticket_id"].(string); ok {
				ticketIDs = []string{tid}
			}
			if phone, ok := s.Data["controller_passenger_phone"].(string); ok {
				passengerPhones = []string{phone}
			}
			if seat, ok := s.Data["controller_seat_number"].(string); ok {
				seatNumbers = []string{seat}
			}
			passengerNames = []string{passengerName}
		}
	}

	if bookingID == "" {
		return "❌ Erreur : aucune réservation sélectionnée. Tapez *menu* pour retourner au menu principal."
	}

	// Start transaction
	tx, err := DB.Beginx()
	if err != nil {
		return fmt.Sprintf("❌ Erreur lors du début de la transaction : %v\n\nTapez *menu* pour retourner au menu principal.", err)
	}

	// 1. Update booking status to confirmed
	_, err = tx.Exec(`UPDATE bookings SET status = 'confirmed' WHERE id = $1`, bookingID)
	if err != nil {
		tx.Rollback()
		return fmt.Sprintf("❌ Erreur lors de la confirmation de la réservation : %v\n\nTapez *menu* pour retourner au menu principal.", err)
	}

	// 2. Create/validate cash payment record
	var paymentID string
	err = tx.Get(&paymentID, `SELECT id FROM payments WHERE reference = $1`, bookingRef+"_cash")
	if err == sql.ErrNoRows {
		_, err = tx.Exec(`
			INSERT INTO payments (reference, amount, currency, payment_method, status, description)
			VALUES ($1, $2, 'USD', 'CASH', 'SUCCESS', 'Paiement espèces au guichet')
		`, bookingRef+"_cash", totalAmount)
		if err != nil {
			tx.Rollback()
			return fmt.Sprintf("❌ Erreur lors de la création du paiement : %v\n\nTapez *menu* pour retourner au menu principal.", err)
		}
	} else if err != nil {
		tx.Rollback()
		return fmt.Sprintf("❌ Erreur lors de la vérification du paiement : %v\n\nTapez *menu* pour retourner au menu principal.", err)
	} else {
		_, err = tx.Exec(`UPDATE payments SET status = 'SUCCESS', updated_at = NOW() WHERE id = $1`, paymentID)
		if err != nil {
			tx.Rollback()
			return fmt.Sprintf("❌ Erreur lors de la mise à jour du paiement : %v\n\nTapez *menu* pour retourner au menu principal.", err)
		}
	}

	// 3. Update ticket payment_validated to TRUE (but NOT check-in)
	for _, ticketID := range ticketIDs {
		_, err = tx.Exec(`
			UPDATE tickets
			SET payment_validated = TRUE
			WHERE id = $1
			AND booking_id IN (SELECT id FROM bookings WHERE status = 'confirmed')`,
			ticketID)
		if err != nil {
			tx.Rollback()
			return fmt.Sprintf("❌ Erreur lors de la validation du paiement : %v\n\nTapez *menu* pour retourner au menu principal.", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Sprintf("❌ Erreur lors de la finalisation : %v\n\nTapez *menu* pour retourner au menu principal.", err)
	}

	// Send WhatsApp notification to ALL passengers
	for i, phone := range passengerPhones {
		if phone != "" {
			name := passengerNames[i]
			if name == "" {
				name = "Passager"
			}
			seat := seatNumbers[i]
			if seat == "" {
				seat = "N/A"
			}
			sendCashValidationNotificationToPassenger(phone, bookingRef, name, origin, destination, seat, totalAmount)
		}
	}

	// Build success message
	var successMsg string
	if isPayAll {
		successMsg = fmt.Sprintf("✅ *Paiement Groupe encaissé avec succès !*\n\n📌 *Référence* : %s\n👥 *Nombre de billets* : %d\n🚌 *Trajet* : %s ➔ %s\n💰 *Montant total encaissé* : %.2f USD\n📊 *Statut* : confirmed (Tous payés)\n\nLes passagers pourront faire leur check-in ultérieurement.\n\nTapez *menu* pour revenir au menu principal.",
			bookingRef, len(ticketIDs), origin, destination, totalAmount)
	} else {
		seat := ""
		if len(seatNumbers) > 0 {
			seat = seatNumbers[0]
		}
		successMsg = fmt.Sprintf("✅ *Paiement Cash encaissé avec succès !*\n\n📌 *Référence* : %s\n👤 *Passager* : %s\n💺 *Siège* : N° %s\n🚌 *Trajet* : %s ➔ %s\n💰 *Montant encaissé* : %.2f USD\n📊 *Statut* : confirmed (Payé)\n\nLe passager pourra faire son check-in ultérieurement.\n\nTapez *menu* pour revenir au menu principal.",
			bookingRef, passengerName, seat, origin, destination, totalAmount)
	}

	// Return to controller actions menu
	s.PreviousStep = s.Step
	s.Step = StepControllerActions

	return successMsg
}

// processControllerBookingResult processes a booking found by the controller
// Implements smart ticket state detection: pending (cash), confirmed, checked_in
func processControllerBookingResult(s *Session, bookingResult BookingDetailsResult) string {
	// Get passenger name and seat number for all cases
	var passengerName string
	if bookingResult.PassengerNames != "" {
		names := strings.Split(bookingResult.PassengerNames, ",")
		if len(names) > 0 && names[0] != "" {
			passengerName = names[0]
		} else {
			passengerName = "Passager"
		}
	} else if bookingResult.FirstPassengerName != "" {
		passengerName = bookingResult.FirstPassengerName
	} else if bookingResult.PassengerName != "" {
		passengerName = bookingResult.PassengerName
	} else {
		passengerName = "Passager"
	}

	var seatNumber string
	if bookingResult.SeatNumber != "" {
		seatNumber = bookingResult.SeatNumber
	} else if bookingResult.SeatNumbers != "" {
		seats := strings.Split(bookingResult.SeatNumbers, ",")
		if len(seats) > 0 && seats[0] != "" {
			seatNumber = seats[0]
		}
	}

	// Store common data in session
	if s.Data == nil {
		s.Data = make(map[string]interface{})
	}
	s.Data["controller_booking_id"] = bookingResult.ID
	s.Data["controller_booking_ref"] = bookingResult.BookingReference
	s.Data["controller_ticket_id"] = bookingResult.TicketID
	s.Data["controller_passenger_name"] = passengerName
	s.Data["controller_passenger_phone"] = bookingResult.PassengerPhone
	s.Data["controller_seat_number"] = seatNumber
	s.Data["controller_origin"] = bookingResult.Origin
	s.Data["controller_destination"] = bookingResult.Destination
	s.Data["controller_total_amount"] = bookingResult.TotalAmount

	// CASE 3: Already checked_in
	if bookingResult.Status == "checked_in" || bookingResult.PaymentStatus == "checked_in" {
		var b strings.Builder
		b.WriteString(fmt.Sprintf("⚠️ *Ce passager a déjà effectué son Check-in.*\n\n"))
		b.WriteString(fmt.Sprintf("📌 *Référence* : %s\n", bookingResult.BookingReference))
		b.WriteString(fmt.Sprintf("👤 *Passager* : %s\n", passengerName))
		b.WriteString(fmt.Sprintf("📊 *Statut* : Déjà embarqué\n\n"))
		b.WriteString("Tapez *menu* pour revenir au menu principal.")
		return b.String()
	}

	// CASE 2: Already confirmed/paid
	if bookingResult.Status == "confirmed" || bookingResult.Status == "paid" {
		// Booking is already paid - offer direct check-in
		s.PreviousStep = s.Step
		s.Step = StepControllerCheckInConfirm

		var b strings.Builder
		b.WriteString(fmt.Sprintf("ℹ️ Cette réservation (*%s*) est payée/confirmée.\n\n", bookingResult.BookingReference))
		b.WriteString(fmt.Sprintf("👤 *Passager* : %s\n", passengerName))
		b.WriteString(fmt.Sprintf("🚌 *Trajet* : %s ➔ %s\n", bookingResult.Origin, bookingResult.Destination))
		b.WriteString(fmt.Sprintf("📅 *Départ* : %s à %s\n", formatDate(bookingResult.DepartureDate), bookingResult.DepartureTime))
		if seatNumber != "" {
			b.WriteString(fmt.Sprintf("💺 *Siège* : N° %s\n", seatNumber))
		}
		b.WriteString("\nVoulez-vous faire le check-in ?\n\n")
		b.WriteString("1️⃣ Oui (Check-in)\n")
		b.WriteString("2️⃣ Non (Retour)\n")
		return b.String()
	}

	// CASE 1: Pending (cash/unpaid) - offer cash payment + check-in workflow
	if bookingResult.Status == "pending" || bookingResult.Status == "" {
		s.PreviousStep = s.Step
		s.Step = StepControllerActionChoice

		// Mark as single ticket (not pay all)
		s.Data["controller_pay_all"] = false
		s.Data["controller_selected_ticket_id"] = bookingResult.TicketID

		var b strings.Builder
		b.WriteString(fmt.Sprintf("💳 *Action requise pour %s*\n\n", passengerName))
		b.WriteString(fmt.Sprintf("📌 *Réservation* : %s\n", bookingResult.BookingReference))
		b.WriteString(fmt.Sprintf("💺 *Siège* : N° %s\n", seatNumber))
		b.WriteString(fmt.Sprintf("🚌 *Trajet* : %s ➔ %s\n", bookingResult.Origin, bookingResult.Destination))
		b.WriteString(fmt.Sprintf("📅 *Départ* : %s à %s\n", formatDate(bookingResult.DepartureDate), bookingResult.DepartureTime))
		b.WriteString(fmt.Sprintf("\n💰 *Montant total* : %.2f USD\n\n", bookingResult.TotalAmount))
		b.WriteString("Sélectionnez une action :\n")
		b.WriteString("1️⃣ Encaisser le Cash ET valider le Check-in (Recommandé)\n")
		b.WriteString("2️⃣ Encaisser le Cash seulement\n")
		b.WriteString("3️⃣ Annuler / Retour\n")
		return b.String()
	}

	// Fallback for unknown status
	return fmt.Sprintf("❓ Statut de réservation inconnu : %s\n\nTapez *menu* pour retourner au menu principal.", bookingResult.Status)
}

// Helper functions - must be at package level in Go
func extractText(evt *events.Message) string {
	msg := evt.Message
	if msg == nil {
		return ""
	}

	// Regular conversation text
	if msg.GetConversation() != "" {
		return strings.TrimSpace(msg.GetConversation())
	}

	// Extended text (replies, mentions, etc.)
	if msg.GetExtendedTextMessage() != nil {
		return strings.TrimSpace(msg.GetExtendedTextMessage().GetText())
	}

	// Button response messages
	if msg.GetButtonsResponseMessage() != nil {
		btnResp := msg.GetButtonsResponseMessage()
		if btnResp.GetSelectedButtonID() != "" {
			log.Printf("[WHATSAPP-DEBUG] Button response: selectedButtonID=%q, selectedDisplayText=%q",
				btnResp.GetSelectedButtonID(), btnResp.GetSelectedDisplayText())
			// Return the button ID as text for processing
			return strings.TrimSpace(btnResp.GetSelectedButtonID())
		}
		if btnResp.GetSelectedDisplayText() != "" {
			return strings.TrimSpace(btnResp.GetSelectedDisplayText())
		}
	}

	// List response messages
	if msg.GetListResponseMessage() != nil {
		listResp := msg.GetListResponseMessage()
		if listResp.GetSingleSelectReply() != nil && listResp.GetSingleSelectReply().GetSelectedRowID() != "" {
			log.Printf("[WHATSAPP-DEBUG] List response: selectedRowID=%q", listResp.GetSingleSelectReply().GetSelectedRowID())
			return strings.TrimSpace(listResp.GetSingleSelectReply().GetSelectedRowID())
		}
	}

	// Template message responses (e.g., button clicks from template messages)
	if msg.GetTemplateMessage() != nil {
		// TemplateMessage may have different structure depending on whatsmeow version
		// Just log that we received a template message
		log.Printf("[WHATSAPP-DEBUG] Template message received")
	}

	// Reaction messages - treat as text for logging
	if msg.GetReactionMessage() != nil {
		reaction := msg.GetReactionMessage()
		return fmt.Sprintf("[REACTION] %s", reaction.GetText())
	}

	// Poll responses
	if msg.GetPollUpdateMessage() != nil {
		// Poll update message structure may vary
		log.Printf("[WHATSAPP-DEBUG] Poll update message received")
	}

	log.Printf("[WHATSAPP-DEBUG] No extractable text from message type: %T", msg)
	return ""
}

// extractQRFromImage downloads and decodes QR code from an image message
// Returns the decoded QR content, or empty string if not an image or QR decode fails
func extractQRFromImage(evt *events.Message) (string, error) {
	msg := evt.Message
	imageMsg := msg.GetImageMessage()
	if imageMsg == nil {
		return "", fmt.Errorf("not an image message")
	}

	// Download the image
	imageData, err := Client.Download(context.Background(), imageMsg)
	if err != nil {
		return "", fmt.Errorf("failed to download image: %w", err)
	}

	// Decode QR code from image
	qrContent, err := decodeQRCode(imageData)
	if err != nil {
		return "", fmt.Errorf("failed to decode QR code: %w", err)
	}

	return qrContent, nil
}

// decodeQRCode decodes a QR code from image data using tuotoo/qrcode
func decodeQRCode(imageData []byte) (string, error) {
	// Use tuotoo/qrcode to decode
	reader := bytes.NewReader(imageData)
	qrMatrix, err := qrcode.Decode(reader)
	if err != nil {
		return "", err
	}
	if qrMatrix == nil {
		return "", fmt.Errorf("no QR code found in image")
	}
	// Return the decoded content
	return qrMatrix.Content, nil
}

func SendText(to types.JID, text string) {
	log.Printf("[INFO] Sending WhatsApp message to %s: %s", to, text)
	msg := buildTextMessage(text)
	if _, err := Client.SendMessage(context.Background(), to, msg); err != nil {
		log.Printf("[ERROR] Failed to send WhatsApp message to %s: %v", to, err)
	}
}

func getActiveAgencies() ([]Agency, error) {
	var agencies []Agency
	err := DB.Select(&agencies, `SELECT o.id, o.name FROM organizations o WHERE o.is_active = true AND EXISTS (SELECT 1 FROM journeys j WHERE j.organization_id = o.id AND j.departure_date >= CURRENT_DATE AND j.status NOT IN ('cancelled', 'completed')) ORDER BY RANDOM()`)
	if err != nil {
		return nil, err
	}
	return agencies, nil
}

func assignSeats(passengers []PassengerDraft, bookedSeats []string) {
	taken := map[string]bool{}
	for _, s := range bookedSeats {
		taken[s] = true
	}
	seatNum := 1
	for i := range passengers {
		for {
			candidate := fmt.Sprintf("%d", seatNum)
			if !taken[candidate] {
				passengers[i].SeatNumber = candidate
				taken[candidate] = true
				seatNum++
				break
			}
			seatNum++
		}
	}
}

func buildTextMessage(text string) *waE2E.Message {
	return &waE2E.Message{
		Conversation: proto.String(text),
	}
}

func sendTicketCard(to types.JID, ref string, j JourneyResult, t CreatedTicket) {
	imgBytes, err := renderTicketCard(
		j.AgencyName, ref, j.Origin, j.Destination, j.Date, j.Time, j.ArrivalTime,
		t.PassengerName, t.PassengerPhone, t.SeatNumber, t.Price, "Active",
		j.BusRegistration, j.AgencyPhone, t.QRToken,
	)
	if err != nil {
		sendTicketImage(to, t)
		return
	}

	uploaded, err := Client.Upload(context.Background(), imgBytes, whatsmeow.MediaImage)
	if err != nil {
		return
	}

	caption := fmt.Sprintf("🎫 Billet PDF : %s", t.DownloadURL)

	imgMsg := &waE2E.ImageMessage{
		URL:           proto.String(uploaded.URL),
		DirectPath:    proto.String(uploaded.DirectPath),
		MediaKey:      uploaded.MediaKey,
		Mimetype:      proto.String("image/png"),
		FileEncSHA256: uploaded.FileEncSHA256,
		FileSHA256:    uploaded.FileSHA256,
		FileLength:    proto.Uint64(uploaded.FileLength),
		Caption:       proto.String(caption),
	}

	Client.SendMessage(context.Background(), to, &waE2E.Message{ImageMessage: imgMsg})
}

func sendTicketImage(to types.JID, t CreatedTicket) {
	qrPNG, err := qrcodegen.Encode(t.QRToken, qrcodegen.Medium, 400)
	if err != nil {
		return
	}

	uploaded, err := Client.Upload(context.Background(), qrPNG, whatsmeow.MediaImage)
	if err != nil {
		return
	}

	caption := fmt.Sprintf("🎫 %s — Siège %s — %.0f USD\nBillet PDF : %s\nMontrez ce QR code à l'embarquement.",
		t.PassengerName, t.SeatNumber, t.Price, t.DownloadURL)

	imgMsg := &waE2E.ImageMessage{
		URL:           proto.String(uploaded.URL),
		DirectPath:    proto.String(uploaded.DirectPath),
		MediaKey:      uploaded.MediaKey,
		Mimetype:      proto.String("image/png"),
		FileEncSHA256: uploaded.FileEncSHA256,
		FileSHA256:    uploaded.FileSHA256,
		FileLength:    proto.Uint64(uploaded.FileLength),
		Caption:       proto.String(caption),
	}

	Client.SendMessage(context.Background(), to, &waE2E.Message{ImageMessage: imgMsg})
}

// Additional helper functions
var digitEmojis = []string{"0️⃣", "1️⃣", "2️⃣", "3️⃣", "4️⃣", "5️⃣", "6️⃣", "7️⃣", "8️⃣", "9️⃣"}

const pageSize = 10

func numberEmoji(n int) string {
	if n == 10 {
		return "🔟"
	}
	if n >= 0 && n < 10 {
		return digitEmojis[n]
	}
	var b strings.Builder
	for _, c := range strconv.Itoa(n) {
		b.WriteString(digitEmojis[c-'0'])
	}
	return b.String()
}

func formatAgencyList(s *Session) string {
	if len(s.Agencies) == 0 {
		return "Aucune agence active trouvée. Tapez *menu* pour réessayer."
	}

	pageSize := 10
	startIndex := s.AgencyPage * pageSize
	endIndex := startIndex + pageSize
	if endIndex > len(s.Agencies) {
		endIndex = len(s.Agencies)
	}

	var b strings.Builder
	b.WriteString("🏢 *Agences actives* ")
	if len(s.Agencies) > pageSize {
		b.WriteString(fmt.Sprintf("(Page %d/%d)", s.AgencyPage+1, (len(s.Agencies)+pageSize-1)/pageSize))
	}
	b.WriteString(" :\n\n")

	// Continuous global numbering: Page 0 shows 1-10, Page 1 shows 11-20, etc.
	for i := startIndex; i < endIndex; i++ {
		globalNum := i + 1 // 1-based global index
		b.WriteString(fmt.Sprintf("%s %s\n", numberEmoji(globalNum), s.Agencies[i].Name))
	}

	// Add navigation options with fixed numeric codes
	if s.AgencyPage > 0 || endIndex < len(s.Agencies) {
		b.WriteString("\n──────────────────\n")
		if s.AgencyPage > 0 {
			b.WriteString("0️⃣0️⃣ ⬅️ Page précédente (Tapez 00)\n")
		}
		if endIndex < len(s.Agencies) {
			b.WriteString("0️⃣1️⃣ ➡️ Page suivante (Tapez 01)\n")
		}
		b.WriteString("──────────────────\n")
	}

	b.WriteString("\nRépondez avec le numéro de l'agence choisie.")
	return b.String()
}

func formatPaginatedResults(s *Session) string {
	if len(s.Results) == 0 {
		return "Aucun voyage trouvé."
	}

	pageSize := 10
	startIndex := s.VoyagePage * pageSize
	endIndex := startIndex + pageSize
	if endIndex > len(s.Results) {
		endIndex = len(s.Results)
	}

	var b strings.Builder
	b.WriteString("🚌 *Liste des voyages disponibles* ")
	if len(s.Results) > pageSize {
		b.WriteString(fmt.Sprintf("(Page %d/%d)", s.VoyagePage+1, (len(s.Results)+pageSize-1)/pageSize))
	}
	b.WriteString(" :\n\n")

	// Continuous global numbering: Page 0 shows 1-10, Page 1 shows 11-20, etc.
	for i := startIndex; i < endIndex; i++ {
		r := s.Results[i]
		globalNum := i + 1 // 1-based global index
		departDateFormatted := formatDate(r.Date)
		arriveDateFormatted := formatDate(r.ArrivalDate)
		departTimeFormatted := formatTime(r.Time)
		arriveTimeFormatted := formatTime(r.ArrivalTime)
		duration := calculateDuration(r.Time, r.ArrivalTime)
		priceCDF := r.Price * 2850

		b.WriteString(fmt.Sprintf("%s %s → %s\n• Agence : %s\n• 🚀 Départ : %s à %s\n• 🏁 Arrivée : %s à %s\n• ⏱️ Durée estimée : %s\n• 💰 Prix : %.0f USD / %.0f CDF\n\n",
			numberEmoji(globalNum), r.Origin, r.Destination, r.AgencyName,
			departDateFormatted, departTimeFormatted,
			arriveDateFormatted, arriveTimeFormatted,
			duration,
			r.Price, priceCDF))
	}

	// Add navigation options with fixed numeric codes
	if s.VoyagePage > 0 || endIndex < len(s.Results) {
		b.WriteString("\n──────────────────\n")
		if s.VoyagePage > 0 {
			b.WriteString("0️⃣0️⃣ ⬅️ Page précédente (Tapez 00)\n")
		}
		if endIndex < len(s.Results) {
			b.WriteString("0️⃣1️⃣ ➡️ Page suivante (Tapez 01)\n")
		}
		b.WriteString("──────────────────\n")
	}

	b.WriteString("\nTapez le numéro du voyage choisi (numérotation continue).")
	return b.String()
}

func formatResults(results []JourneyResult) string {
	var b strings.Builder
	b.WriteString("🚌 Liste des voyages disponibles :\n\n")
	for i, r := range results {
		departDateFormatted := formatDate(r.Date)
		arriveDateFormatted := formatDate(r.ArrivalDate)
		departTimeFormatted := formatTime(r.Time)
		arriveTimeFormatted := formatTime(r.ArrivalTime)
		duration := calculateDuration(r.Time, r.ArrivalTime)
		priceCDF := r.Price * 2850

		b.WriteString(fmt.Sprintf("%s %s → %s\n• Agence : %s\n• 🚀 Départ : %s à %s\n• 🏁 Arrivée : %s à %s\n• ⏱️ Durée estimée : %s\n• 💰 Prix : %.0f USD / %.0f CDF\n\n",
			numberEmoji(i+1), r.Origin, r.Destination, r.AgencyName,
			departDateFormatted, departTimeFormatted,
			arriveDateFormatted, arriveTimeFormatted,
			duration,
			r.Price, priceCDF))
	}
	b.WriteString("Tapez le numéro du voyage choisi.")
	return b.String()
}

func buildConfirmSummary(s *Session) string {
	var b strings.Builder
	b.WriteString("Récapitulatif :\n")
	for _, p := range s.Passengers {
		b.WriteString(fmt.Sprintf("- %s (%s)\n", p.Name, valueOr(p.Phone, "sans téléphone")))
	}
	baseAmountUSD := s.SelectedJourney.Price * float64(len(s.Passengers))
	totalAmountUSD := baseAmountUSD
	totalAmountCDF := baseAmountUSD * 2850
	b.WriteString(fmt.Sprintf("\nMontant total à payer : %.2f USD / %.2f CDF\n\n", totalAmountUSD, totalAmountCDF))
	b.WriteString("\nConfirmez-vous la réservation ? (oui/non)")
	return b.String()
}

func valueOr(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

// isNumeric checks if a string contains only digits
func isNumeric(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return len(s) > 0
}

func parseDate(input string) string {
	input = strings.TrimSpace(input)
	lower := strings.ToLower(input)

	for _, sep := range []string{"/", "-", "."} {
		if strings.Contains(input, sep) {
			parts := strings.Split(input, sep)
			if len(parts) == 3 {
				day, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
				month, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
				year, err3 := strconv.Atoi(strings.TrimSpace(parts[2]))
				if err1 == nil && err2 == nil && err3 == nil && day >= 1 && day <= 31 && month >= 1 && month <= 12 && year >= 2020 {
					return fmt.Sprintf("%04d-%02d-%02d", year, month, day)
				}
			}
		}
	}

	fields := strings.Fields(lower)
	if len(fields) == 3 {
		day, err1 := strconv.Atoi(fields[0])
		month, ok := frenchMonths[fields[1]]
		year, err2 := strconv.Atoi(fields[2])
		if err1 == nil && ok && err2 == nil && day >= 1 && day <= 31 && year >= 2020 {
			return fmt.Sprintf("%04d-%02d-%02d", year, month, day)
		}
	}

	return ""
}

var frenchMonths = map[string]int{
	"janvier": 1, "février": 2, "fevrier": 2, "mars": 3, "avril": 4,
	"mai": 5, "juin": 6, "juillet": 7, "août": 8, "aout": 8,
	"septembre": 9, "octobre": 10, "novembre": 11, "décembre": 12, "decembre": 12,
}

// sendCashValidationNotificationToPassenger sends a WhatsApp notification to the passenger
// when a controller validates their cash payment
func sendCashValidationNotificationToPassenger(passengerPhone, bookingRef, passengerName, origin, destination, seatNumber string, amount float64) {
	if passengerPhone == "" {
		log.Printf("[CASH VALIDATION] No passenger phone available for notification")
		return
	}

	cleanedPhone := CleanPhoneNumber(passengerPhone)
	if len(cleanedPhone) != 12 {
		log.Printf("[CASH VALIDATION] Invalid passenger phone format: %s", passengerPhone)
		return
	}

	jid := types.NewJID(cleanedPhone, "s.whatsapp.net")

	message := fmt.Sprintf("✅ *Paiement Cash Validé par le Contrôleur !*\n\n📌 *Référence* : %s\n👤 *Passager* : %s\n🚌 *Trajet* : %s ➔ %s\n💰 *Montant* : %.2f USD\n📊 *Statut* : Confirmé & Payé\n\nVotre billet est maintenant validé pour l'embarquement.",
		bookingRef, passengerName, origin, destination, amount)

	SendText(jid, message)
	log.Printf("[CASH VALIDATION] Notification sent to passenger %s for booking %s", cleanedPhone, bookingRef)
}

// createAgency creates a new organization (agency) in the database
// Returns the organization ID on success
func createAgency(name, managerName, address, contactPhone string) (string, error) {
	var orgID string

	// Generate a slug from the name for uniqueness
	slug := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	// Add a random suffix to ensure uniqueness
	slug = fmt.Sprintf("%s-%d", slug, time.Now().UnixNano()%10000)

	err := DB.QueryRow(`
		INSERT INTO organizations (id, name, slug, is_active, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, $2, true, NOW(), NOW())
		RETURNING id
	`, name, slug).Scan(&orgID)

	if err != nil {
		return "", fmt.Errorf("échec de la création de l'organisation : %w", err)
	}

	// Note: In a full implementation, you might also want to create a user account
	// for the agency manager with role 'owner' or 'agent' linked to this organization.
	// This would require additional fields like email and password.

	return orgID, nil
}
