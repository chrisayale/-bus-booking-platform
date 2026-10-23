// THIS IS A TEST TO SEE IF THE FILE IS BEING UPDATED CORRECTLY

package whatsapp
import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
	"unicode"

	"github.com/bus-booking/BE-bus-booking-platform/internal/payments"
	"github.com/bus-booking/BE-bus-booking-platform/internal/utils"
	"github.com/jmoiron/sqlx"
	"database/sql"
	"go.mau.fi/whatsmeow/types"
	"github.com/bus-booking/BE-bus-booking-platform/internal/models"
)

// CleanPhoneNumber removes all non-digit characters and ensures the number starts with the DRC country code (243).
func CleanPhoneNumber(phone string) string {
	// Remove all non-digit characters
	var digits string
	for _, char := range phone {
		if unicode.IsDigit(char) {
			digits += string(char)
		}
	}

	// If the number starts with '0', assume it's a local DRC number and replace the leading '0' with '243'
	if len(digits) > 0 && digits[0] == '0' {
		digits = "243" + digits[1:]
	}

	// If the number does not start with '243', prepend '243' (assuming it's a local number without the leading zero)
	if !strings.HasPrefix(digits, "243") {
		digits = "243" + digits
	}

	return digits
}

// FlexPay est le provider de paiement utilisé pour le bot WhatsApp.
var FlexPay *payments.FlexPayProvider

// pendingBookingMetadata est sérialisé en JSON dans payments.metadata
type pendingBookingMetadata struct {
	JourneyID   string           `json:"journey_id"`
	GuestName   string           `json:"guest_name"`
	GuestEmail  string           `json:"guest_email"`
	GuestPhone  string           `json:"guest_phone"`
	Passengers  []PassengerDraft `json:"passengers"`
	PaymentType string           `json:"payment_type"` // "carte" ou "mobile money"
}

// PendingBookingMetadata is the exported version for use in other packages
type PendingBookingMetadata struct {
	JourneyID   string           `json:"journey_id"`
	GuestName   string           `json:"guest_name"`
	GuestEmail  string           `json:"guest_email"`
	GuestPhone  string           `json:"guest_phone"`
	Passengers  []PassengerDraft `json:"passengers"`
	PaymentType string           `json:"payment_type"` // "carte" ou "mobile money"
	// RecipientJID is the exact JID/LID of the WhatsApp user for sending notifications
	RecipientJID string           `json:"recipient_jid,omitempty"`
}

// InitiateOnlinePayment initie un paiement par carte bancaire via l'API REST FlexPay
func InitiateOnlinePayment(journeyID string, journey JourneyResult, passengers []PassengerDraft, phone string, currency string, recipientJID string, flexPayPhone string) (redirectURL string, reference string, err error) {
	if FlexPay == nil {
		return "", "", fmt.Errorf("FlexPay non configuré")
	}

	if DB == nil {
		return "", "", fmt.Errorf("DB non configuré")
	}

	baseAmountUSD := journey.Price * float64(len(passengers))

	var totalAmount float64
	var currencyCode string
	if currency == "CDF" {
		totalAmount = baseAmountUSD * 2850
		currencyCode = "CDF"
	} else {
		totalAmount = baseAmountUSD
		currencyCode = "USD"
	}

	reference, err = utils.GenerateReference()
	if err != nil {
		return "", "", err
	}

	meta := PendingBookingMetadata{
		JourneyID:   journeyID,
		GuestName:   passengers[0].Name,
		GuestEmail:  fmt.Sprintf("whatsapp+%s@ayaletrans.local", phone),
		GuestPhone:  CleanPhoneNumber(phone),
		Passengers:  passengers,
		PaymentType: "carte",
		RecipientJID: recipientJID,
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return "", "", err
	}

	description := "Billet(s) bus - " + reference

	if _, err = payments.CreatePendingPayment(DB, reference, totalAmount, currencyCode, payments.MethodCard, description, string(metaJSON)); err != nil {
		return "", "", err
	}

	// Appel à l'API REST FlexPay pour les paiements par carte (type "2") via le provider
	req := payments.CardPaymentRequest{
		Merchant:    FlexPay.Merchant,
		Type:        "2", // Type "2" pour les paiements par carte
		Reference:   reference,
		Amount:      fmt.Sprintf("%.2f", totalAmount),
		Currency:    currencyCode,
		CallbackURL: AppURL + "/api/payments/flexpay/callback",
		ApproveURL:  AppURL + "/api/payments/flexpay/approve",
		CancelURL:   AppURL + "/api/payments/flexpay/cancel",
		DeclineURL:  AppURL + "/api/payments/flexpay/decline",
	}
	// Log the URL values for debugging
	log.Printf("[FLEXPAY] Debug URLs - Callback: %s, Approve: %s, Cancel: %s, Decline: %s",
		AppURL+"/api/payments/flexpay/callback",
		AppURL+"/api/payments/flexpay/approve",
		AppURL+"/api/payments/flexpay/cancel",
		AppURL+"/api/payments/flexpay/decline")
	// Additional debug logging for AppURL
	log.Printf("[FLEXPAY] Debug AppURL value: %s", AppURL)

	// Log the exact JSON payload sent to FlexPay for inspection
	data, err := json.Marshal(req)
	if err != nil {
		return "", "", err
	}
	log.Printf("[FLEXPAY] Sending payload for reference %s: %s", reference, string(data))

	// Utiliser la méthode CreateCardPayment du provider
	flexPayResp, err := FlexPay.CreateCardPayment(req)
	if err != nil {
		return "", "", err
	}

	// Log the request payload sent to FlexPay for debugging (response logged in CreateCardPayment)
	log.Printf("[FLEXPAY] Request payload for reference %s: %s", reference, string(data))

	var result struct {
		Code        string `json:"code"`
		Message     string `json:"message"`
		URL         string `json:"url,omitempty"`
		OrderNumber string `json:"orderNumber,omitempty"`
	}
	result.Code = flexPayResp.Code
	result.Message = flexPayResp.Message
	result.URL = flexPayResp.URL
	result.OrderNumber = flexPayResp.OrderNumber

	if result.Code != "0" {
		return "", "", fmt.Errorf("FlexPay a refusé la demande: %s", result.Message)
	}

	if err := payments.SetOrderNumber(DB, reference, result.OrderNumber); err != nil {
		return "", "", err
	}

	// Start background goroutine to monitor payment timeout (5 minutes)
	ctx := context.Background()
	go StartOrderStatusPoller(ctx, DB, FlexPay, reference, result.OrderNumber)

	return result.URL, reference, nil
}

// InitiateMobileMoneyPayment initie un paiement Mobile Money via FlexPay (Push USSD)
func InitiateMobileMoneyPayment(journeyID string, journey JourneyResult, passengers []PassengerDraft, mobileMoneyPhone string, guestPhone string, currency string, recipientJID string) (redirectURL string, reference string, err error) {
	log.Printf("[DEBUG] Entrée dans InitiateMobileMoneyPayment pour le numéro %s", mobileMoneyPhone)

	if FlexPay == nil {
		log.Printf("[ERROR] FlexPay est nil dans le package whatsapp !")
		return "", "", fmt.Errorf("FlexPay non configuré")
	}

	if DB == nil {
		log.Printf("[ERROR] La base de données (DB) est nil dans le package whatsapp !")
		return "", "", fmt.Errorf("DB non configuré")
	}

	log.Printf("[DEBUG] DB et FlexPay OK, calcul des montants...")

	baseAmountUSD := journey.Price * float64(len(passengers))

	var totalAmount float64
	var currencyCode string
	if currency == "CDF" {
		totalAmount = baseAmountUSD * 2850
		currencyCode = "CDF"
	} else {
		totalAmount = baseAmountUSD
		currencyCode = "USD"
	}

	reference, err = utils.GenerateReference()
	if err != nil {
		log.Printf("[ERROR] Échec de génération de la référence: %v", err)
		return "", "", err
	}

	meta := PendingBookingMetadata{
		JourneyID:   journeyID,
		GuestName:   passengers[0].Name,
		GuestEmail:  fmt.Sprintf("whatsapp+%s@ayaletrans.local", guestPhone),
		GuestPhone:  CleanPhoneNumber(guestPhone),
		Passengers:  passengers,
		PaymentType: "mobile money",
		RecipientJID: recipientJID,
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		log.Printf("[ERROR] Échec du marshal metadata: %v", err)
		return "", "", err
	}

	description := "Billet(s) bus - " + reference

	log.Printf("[DEBUG] Enregistrement du paiement en attente dans la DB (ref: %s)...", reference)
	if _, err = payments.CreatePendingPayment(DB, reference, totalAmount, currencyCode, "mobile money", description, string(metaJSON)); err != nil {
		log.Printf("[ERROR] Échec CreatePendingPayment: %v", err)
		return "", "", err
	}

	log.Printf("[DEBUG] Appel de l'API FlexPay CreateMobilePayment...")

	// Clean and validate the mobile money phone number
	originalPhone := mobileMoneyPhone
	cleanedPhone := CleanPhoneNumber(mobileMoneyPhone)
	log.Printf("[DEBUG] Numéro de téléphone original: %s, numéro nettoyé: %s", originalPhone, cleanedPhone)

	res, err := FlexPay.CreateMobilePayment(payments.MobilePaymentRequest{
		Reference:   reference,
		Phone:       cleanedPhone, // Use cleaned phone number to ensure correct format
		Amount:      fmt.Sprintf("%.2f", totalAmount),
		Currency:    currencyCode,
		CallbackURL: AppURL + "/api/payments/flexpay/callback",
	})
	if err != nil {
		log.Printf("[ERROR] Échec de l'appel HTTP FlexPay: %v", err)
		return "", "", err
	}

	log.Printf("[DEBUG] Réponse FlexPay reçue, code: %s, message: %s", res.Code, res.Message)
	if res.Code != "0" {
		return "", "", fmt.Errorf("FlexPay a refusé la demande Mobile Money: %s", res.Message)
	}

	if res.OrderNumber != "" {
		if err := payments.SetOrderNumber(DB, reference, res.OrderNumber); err != nil {
			log.Printf("[ERROR] Échec SetOrderNumber: %v", err)
			return "", "", err
		}
	}

	// Start background goroutine to monitor payment timeout (5 minutes)
	ctx := context.Background()
	go StartOrderStatusPoller(ctx, DB, FlexPay, reference, res.OrderNumber)

	log.Printf("[DEBUG] InitiateMobileMoneyPayment terminé avec succès pour la référence %s", reference)
	return res.Message, reference, nil
}

// StartOrderStatusPoller démarre une goroutine qui vérifie l'état d'un paiement toutes les 5 secondes pendant 5 minutes
// En cas de succès, il met à jour la base de données (la finalisation de la réservation sera gérée par le webhook FlexPay)
func StartOrderStatusPoller(ctx context.Context, db *sqlx.DB, flexPay *payments.FlexPayProvider, reference string, orderNumber string) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	cleanupOldPendingPayments()

	// 5 minutes = 60 intervalles de 5 secondes
	maxAttempts := 60
	attempts := 0

	for {
		// Check if payment is still pending before each attempt
		var currentStatus string
		err := db.Get(&currentStatus, `SELECT status FROM payments WHERE order_number = $1`, orderNumber)
		if err != nil || currentStatus != payments.StatusPending {
			// Payment is no longer pending (success, failed, cancelled, expired, etc.)
			log.Printf("[FLEXPAY] Payment %s is no longer pending (status: %s), stopping poller", orderNumber, currentStatus)
			return
		}

		select {
		case <-ctx.Done():
			log.Printf("[FLEXPAY] Arrêt du poller pour la référence %s (contexte annulé)", reference)
			return
		case <-ticker.C:
			attempts++
			log.Printf("[FLEXPAY] Vérification du statut de paiement (tentative %d/%d) pour la référence %s", attempts, maxAttempts, reference)

			// Check if we have exceeded the maximum attempts
			if attempts >= maxAttempts {
				log.Printf("[FLEXPAY] Timeout atteint après %d tentatives pour la référence %s", maxAttempts, reference)

				// Mettre à jour la base de données en échec (timeout)
				if err := payments.MarkFailed(db, orderNumber, ""); err != nil {
					log.Printf("[FLEXPAY] Erreur lors de la mise à jour du statut en échec pour la référence %s: %v", reference, err)
				}
				// Envoyer un message WhatsApp pour informer l'utilisateur du timeout via la fonction centralisée
				handlePaymentExpiration(reference, orderNumber)
				return
			}

			// Vérifier le statut du paiement
			checkResp, err := flexPay.CheckOrderStatus(orderNumber)
			if err != nil {
				log.Printf("[FLEXPAY] Erreur lors de la vérification du statut pour la référence %s: %v", reference, err)
				continue
			}

			// Vérifier si la réponse est valide
			if checkResp == nil {
				log.Printf("[FLEXPAY] Réponse nil reçue pour la référence %s", reference)
				continue
			}
			if checkResp.Transaction == nil {
				log.Printf("[FLEXPAY] Réponse reçue avec transaction nil pour la référence %s", reference)
				continue
			}

			// Analyser le statut du paiement
			status := checkResp.Transaction.Status
			log.Printf("[FLEXPAY] Statut reçu pour la référence %s: %s (0: Succès, 1: Échec, 2: En attente, 3: Non trouvé)", reference, status)

			switch status {
			case "0": // Succès
				log.Printf("[FLEXPAY] Paiement réussi pour la référence %s", reference)

				// Mettre à jour la base de données
				if err := payments.MarkSuccess(db, orderNumber, checkResp.Transaction.Reference, ""); err != nil {
					log.Printf("[FLEXPAY] Erreur lors de la mise à jour du statut en succès pour la référence %s: %v", reference, err)
				}
				// La finalisation de la réservation sera gérée par le webhook FlexPay
				return

			case "1": // Échec
				log.Printf("[FLEXPAY] Paiement échoué pour la référence %s", reference)

				// Mettre à jour la base de données
				if err := payments.MarkFailed(db, orderNumber, ""); err != nil {
					log.Printf("[FLEXPAY] Erreur lors de la mise à jour du statut en échec pour la référence %s: %v", reference, err)
				}
				return

			case "3": // Non trouvé
				log.Printf("[FLEXPAY] Paiement non trouvé pour la référence %s", reference)

				// Mettre à jour la base de données en échec
				if err := payments.MarkFailed(db, orderNumber, ""); err != nil {
					log.Printf("[FLEXPAY] Erreur lors de la mise à jour du statut en échec pour la référence %s: %v", reference, err)
				}
				return

			case "2": // En attente - continuer à attendre
				// Rien à faire, on continue la boucle
			}
		}
	}
}

// cleanupOldPendingPayments met à jour tous les paiements PENDING plus anciens que 300 secondes en EXPIRED
func cleanupOldPendingPayments() {
	_, err := DB.Exec(`
		UPDATE payments
		SET status = $1, updated_at = NOW()
		WHERE status = $2
		AND created_at < NOW() - INTERVAL '300 seconds'
	`, payments.StatusExpired, payments.StatusPending)
	if err != nil {
		log.Printf("[PAYMENT TIMEOUT WORKER] Cleanup error: %v", err)
	}
}

// handlePaymentExpiration gère l'expiration d'un paiement en envoyant une notification
// et en mettant à jour l'état de la session. Cette fonction est centralisée pour éviter
// la duplication du code entre StartOrderStatusPoller et StartPaymentTimeoutWorker.
func handlePaymentExpiration(reference string, orderNumber string) {
	// Get the payment to extract metadata for notification
	payment, err := payments.GetByOrderNumber(DB, orderNumber)
	if err != nil || !payment.Metadata.Valid || payment.Metadata.String == "" {
		log.Printf("[PAYMENT EXPIRATION] Could not retrieve payment metadata for reference %s: %v", reference, err)
		return
	}

	var pendingMeta PendingBookingMetadata
	if err := json.Unmarshal([]byte(payment.Metadata.String), &pendingMeta); err != nil {
		log.Printf("[PAYMENT EXPIRATION] Failed to unmarshal metadata for reference %s: %v", reference, err)
		return
	}

	// Use RecipientJID if available (exact JID/LID from user's session)
	var jid types.JID
	if pendingMeta.RecipientJID != "" {
		jidStr := strings.TrimSpace(pendingMeta.RecipientJID)
		if jidStr != "" {
			parsedJID, err := types.ParseJID(jidStr)
			if err == nil {
				jid = parsedJID
			} else {
				log.Printf("[PAYMENT EXPIRATION] Failed to parse RecipientJID '%s': %v", pendingMeta.RecipientJID, err)
			}
		}
	}
	// Fallback to building JID from phone number if RecipientJID is not available or invalid
	if jid.IsEmpty() {
		cleanedPhone := CleanPhoneNumber(pendingMeta.GuestPhone)
		// Validate cleanedPhone: must be exactly 12 digits for DRC format (243xxxxxxxxx)
		if len(cleanedPhone) == 12 {
			// Check session state and send expiration message only if not already sent
			s := getSession(cleanedPhone)
			if !s.NotificationSent {
				// If this is a retry that has expired/failed, immediately cancel and return to main menu
				if s.HasRetried {
					// Reset session completely
					resetSession(cleanedPhone)
					// Send specific cancellation message for retry failure
					message := "⌛ Le délai de la relance a expiré. La transaction est annulée."
					jid = types.NewJID(cleanedPhone, "s.whatsapp.net")
					SendText(jid, message)
					return
				}

				// Normal expiration handling (first timeout)
				s.NotificationSent = true
				s.Step = StepPaymentExpired
				s.PaymentError = "Le temps alloué pour le paiement est expiré"
				saveSession(cleanedPhone, s)
				jid = types.NewJID(cleanedPhone, "s.whatsapp.net")
				var message string
				if s.HasRetried {
					message = "😔 Le temps alloué pour le paiement est expiré. Votre transaction a été automatiquement annulée.\n\n❌ Nombre maximum de relances atteint. La réservation a été annulée."
				} else {
					message = "😔 Le temps alloué pour le paiement est expiré. Votre transaction a été automatiquement annulée.\n\nChoisissez une option :\n1️⃣ Relancer le paiement\n2️⃣ Annuler / Retour au menu principal"
				}
				SendText(jid, message)
			}
		} else {
			log.Printf("[PAYMENT EXPIRATION] Invalid cleanedPhone for JID creation: %s", cleanedPhone)
		}
	}

	if !jid.IsEmpty() {
		// Check session state and send expiration message only if not already sent
		// We need to get the session again since we might have built the jid from RecipientJID
		var phone string
		if pendingMeta.RecipientJID != "" && !jid.IsEmpty() {
			// Extract phone from JID (format: 243xxxxxxxxx@s.whatsapp.net)
			jidStr := jid.String()
			if atIdx := strings.Index(jidStr, "@"); atIdx > 0 {
				phone = jidStr[:atIdx]
			}
		} else {
			phone = CleanPhoneNumber(pendingMeta.GuestPhone)
		}

		if phone != "" {
			s := getSession(phone)
			if !s.NotificationSent {
				// If this is a retry that has expired/failed, immediately cancel and return to main menu
				if s.HasRetried {
					// Reset session completely
					resetSession(phone)
					// Send specific cancellation message for retry failure
					message := "⌛ Le délai de la relance a expiré. La transaction est annulée."
					SendText(jid, message)
					return
				}

				// Normal expiration handling (first timeout)
				s.NotificationSent = true
				s.Step = StepPaymentExpired
				s.PaymentError = "Le temps alloué pour le paiement est expiré"
				saveSession(phone, s)
				message := "😔 Le temps alloué pour le paiement est expiré. Votre transaction a été automatiquement annulée.\n\nChoisissez une option :\n1️⃣ Relancer le paiement\n2️⃣ Annuler / Retour au menu principal"
				if s.HasRetried {
					message = "😔 Le temps alloué pour le paiement est expiré. Votre transaction a été automatiquement annulée.\n\n❌ Nombre maximum de relances atteint. La réservation a été annulée."
				}
				SendText(jid, message)
			}
		}
	}
}

func CompleteBookingFromPayment(p *payments.Payment) error {
	if !p.Metadata.Valid || p.Metadata.String == "" {
		return fmt.Errorf("paiement %s sans metadata, impossible de créer la réservation", p.Reference)
	}

	var meta pendingBookingMetadata
	if err := json.Unmarshal([]byte(p.Metadata.String), &meta); err != nil {
		return fmt.Errorf("metadata invalide: %w", err)
	}

	cleanedPhone := CleanPhoneNumber(meta.GuestPhone)
	jid := types.NewJID(cleanedPhone, "s.whatsapp.net")

	tickets, ref, err := CreateGuestBooking(meta.JourneyID, meta.GuestName, meta.GuestEmail, meta.Passengers, AppURL)
	if err != nil {
		SendText(jid, fmt.Sprintf(
			"⚠️ Votre paiement a été confirmé, mais un problème est survenu lors de la création de votre billet : %s\nContactez le support. Référence paiement : %s",
			err.Error(), p.Reference))
		return err
	}

	var mj models.Journey
	err = DB.Get(&mj, `
		SELECT id, organization_id, route_id, bus_id,
			departure_date::text, departure_time::text,
			arrival_date::text, arrival_time::text,
			status, created_at, updated_at
		FROM journeys WHERE id=$1`, meta.JourneyID)
	if err == nil {
		_ = enrichJourneys([]models.Journey{mj})
	}

	// Enrich the journey to get full details for ticket cards
	var journeyResult JourneyResult
	if err == nil {
		var route models.Route
		if err := DB.Get(&route, `SELECT * FROM routes WHERE id=$1`, mj.RouteID); err == nil {
			var org models.Organization
			DB.Get(&org, `SELECT * FROM organizations WHERE id=$1`, mj.OrganizationID)
			var bus models.Bus
			DB.Get(&bus, `SELECT * FROM buses WHERE id=$1`, mj.BusID)

			journeyResult = JourneyResult{
				ID:              mj.ID,
				Origin:          route.Origin,
				Destination:     route.Destination,
				Date:            mj.DepartureDate,
				Time:            mj.DepartureTime,
				ArrivalTime:     mj.ArrivalTime,
				Price:           route.BasePrice,
				AgencyName:      org.Name,
				AgencyPhone:     derefStr(org.Phone),
				BusRegistration: bus.RegistrationNumber,
			}
		}
	}

	// Send ticket cards for all passengers
	if len(tickets) > 0 && journeyResult.ID != "" {
		sendTicketCard(jid, ref, journeyResult, tickets[0])

		// Send additional tickets for multiple passengers
		for i := 1; i < len(tickets); i++ {
			sendTicketCard(jid, ref, journeyResult, tickets[i])
		}
	}

	var amountStr string
	if p.Amount != 0 {
		amountStr = fmt.Sprintf("%.2f", p.Amount)
	} else {
		amountStr = "0.00"
	}
	var dateHeure string
	if !p.CreatedAt.IsZero() {
		dateHeure = p.CreatedAt.Format("02/01/2006 15:04:05")
	} else {
		dateHeure = "date inconnue"
	}
	var statut string
	if p.Status == "0" {
		statut = "Confirmé & Validé"
	} else {
		statut = fmt.Sprintf("Statut %s", p.Status)
	}

	confirmationMsg := fmt.Sprintf("✅ PAIEMENT CONFIRMÉ !\n\n🧾 REÇU DE FACTURATION :\n- ID Transaction : %s\n- Mode de paiement : %s\n- Montant : %s\n- Date & Heure : %s\n- Statut : %s\n\n🎟️ Vos billet(s) de bus sont désormais OFFICIELLEMENT CONFIRMÉ(S).\n⚠️ Veuillez présenter ce(s) billet(s) ou votre code de confirmation lors de l'embarquement pour vérification.\n\n🚌 Toute l'équipe d'AyaleTrans vous souhaite un très bon voyage !\n\n📞 Assistance Client :\n- Tél & WhatsApp : +243974645174",
		p.Reference, meta.PaymentType, amountStr, dateHeure, statut)

	SendText(jid, confirmationMsg)
	return nil
}
func StartPaymentTimeoutWorker() {
	go func() {
		ticker := time.NewTicker(10 * time.Second) // Vérifier toutes les 10 secondes
		defer ticker.Stop()
		cleanupOldPendingPayments()

		for {
			select {
			case <-ticker.C:
				// Récupérer tous les paiements en attente depuis plus de 300 secondes
				var expiredPayments []payments.Payment
				err := DB.Select(&expiredPayments, `
					SELECT * FROM payments
					WHERE status = $1
					AND created_at < NOW() - INTERVAL '300 seconds'
					ORDER BY created_at
					LIMIT 50
				`, payments.StatusPending)

				if err != nil {
					log.Printf("[PAYMENT TIMEOUT WORKER] Erreur lors de la recherche des paiements expirés: %v", err)
					continue
				}

				// Traiter chaque paiement expiré
				for _, payment := range expiredPayments {
					log.Printf("[PAYMENT TIMEOUT WORKER] Traitement du paiement expiré: %s", payment.Reference)

					// Double-check that payment is still pending before processing
					var currentStatus string
					err := DB.Get(&currentStatus, `SELECT status FROM payments WHERE id = $1`, payment.ID)
					if err != nil || currentStatus != payments.StatusPending {
						log.Printf("[PAYMENT TIMEOUT WORKER] Payment %s is no longer pending (status: %s), skipping", payment.Reference, currentStatus)
						continue
					}

					// Update payment status to expired
					var result sql.Result

					if payment.OrderNumber.Valid {
						result, err = DB.Exec(
							"UPDATE payments SET status = $1, callback_data = $2, updated_at = NOW() WHERE order_number = $3 AND status = $4",
							payments.StatusExpired, "", payment.OrderNumber.String, payments.StatusPending,
						)
					} else {
						// Fallback: update by ID if OrderNumber is not valid
						result, err = DB.Exec(
							"UPDATE payments SET status = $1, callback_data = $2, updated_at = NOW() WHERE id = $3 AND status = $4",
							payments.StatusExpired, "", payment.ID, payments.StatusPending,
						)
					}

					if err != nil {
						log.Printf("[PAYMENT TIMEOUT WORKER] Erreur lors de la mise à jour du statut en expiré pour la référence %s: %v", payment.Reference, err)
						continue
					}

					rowsAffected, err := result.RowsAffected()
					if err != nil {
						log.Printf("[PAYMENT TIMEOUT WORKER] Erreur lors de la récupération du nombre de lignes affectées: %v", err)
						continue
					}
					if rowsAffected == 0 {
						log.Printf("[PAYMENT TIMEOUT WORKER] Paiement %s n'est plus en statut pending (déjà traité)", payment.Reference)
						continue
					}

					// Database update successful, now attempt to send expiration message with fallback
					handlePaymentExpiration(payment.Reference, payment.OrderNumber.String)
				}
			}
		}
	}()
}
