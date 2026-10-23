package whatsapp

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/bus-booking/BE-bus-booking-platform/internal/models"
	"github.com/bus-booking/BE-bus-booking-platform/internal/utils"
	"github.com/jmoiron/sqlx"
)

var DB *sqlx.DB
var QRSecret string
var AppURL string

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func enrichJourneys(journeys []models.Journey) []JourneyResult {
	var results []JourneyResult
	for _, j := range journeys {
		var route models.Route
		if err := DB.Get(&route, `SELECT * FROM routes WHERE id=$1`, j.RouteID); err != nil {
			continue
		}
		var org models.Organization
		DB.Get(&org, `SELECT * FROM organizations WHERE id=$1`, j.OrganizationID)
		var bus models.Bus
		DB.Get(&bus, `SELECT * FROM buses WHERE id=$1`, j.BusID)

		results = append(results, JourneyResult{
			ID:              j.ID,
			Origin:          route.Origin,
			Destination:     route.Destination,
			Date:            j.DepartureDate,
			Time:            j.DepartureTime,
			ArrivalTime:     j.ArrivalTime,
			Price:           route.BasePrice,
			AgencyName:      org.Name,
			AgencyPhone:     derefStr(org.Phone),
			BusRegistration: bus.RegistrationNumber,
		})
	}
	return results
}

func SearchJourneys(origin, destination, date string, agencyID string) ([]JourneyResult, error) {
	var journeys []models.Journey
	query := `
		SELECT j.id, j.organization_id, j.route_id, j.bus_id,
			j.departure_date::text, j.departure_time::text,
			j.arrival_date::text, j.arrival_time::text,
			j.status, j.created_at, j.updated_at
		FROM journeys j
		JOIN routes r ON j.route_id = r.id
		JOIN organizations o ON j.organization_id = o.id
		WHERE r.origin ILIKE $1
		  AND r.destination ILIKE $2
		  AND j.departure_date = $3
		  AND j.status NOT IN ('cancelled', 'completed')
	`

	var args []interface{}
	args = append(args, "%"+origin+"%", "%"+destination+"%", date)

	if agencyID != "" {
		query += " AND o.id = $4"
		args = append(args, agencyID)
	}

	query += " ORDER BY j.departure_time"

	err := DB.Select(&journeys, query, args...)
	if err != nil {
		return nil, err
	}
	return enrichJourneys(journeys), nil
}

func SearchUpcomingJourneys(origin, destination string, limit int) ([]JourneyResult, error) {
	var journeys []models.Journey
	err := DB.Select(&journeys, `
		SELECT j.id, j.organization_id, j.route_id, j.bus_id,
			j.departure_date::text, j.departure_time::text,
			j.arrival_date::text, j.arrival_time::text,
			j.status, j.created_at, j.updated_at
		FROM journeys j
		JOIN routes r ON j.route_id = r.id
		WHERE r.origin ILIKE $1
		  AND r.destination ILIKE $2
		  AND j.departure_date >= CURRENT_DATE
		  AND j.status NOT IN ('cancelled', 'completed')
		ORDER BY j.departure_date, j.departure_time
		LIMIT $3`,
		"%"+origin+"%", "%"+destination+"%", limit)
	if err != nil {
		return nil, err
	}
	return enrichJourneys(journeys), nil
}

func SearchJourneysOnDate(date string, limit int) ([]JourneyResult, error) {
	var journeys []models.Journey
	err := DB.Select(&journeys, `
		SELECT j.id, j.organization_id, j.route_id, j.bus_id,
			j.departure_date::text, j.departure_time::text,
			j.arrival_date::text, j.arrival_time::text,
			j.status, j.created_at, j.updated_at
		FROM journeys j
		WHERE j.departure_date = $1
		  AND j.status NOT IN ('cancelled', 'completed')
		ORDER BY j.departure_time
		LIMIT $2`,
		date, limit)
	if err != nil {
		return nil, err
	}
	return enrichJourneys(journeys), nil
}

type CreatedTicket struct {
	ID              string `db:"id"`
	Destination     string `db:"destination"`
	SeatNumber     string `db:"seat_number"`
	PassengerName  string `db:"passenger_name"`
	PassengerPhone string `db:"passenger_phone"`
	Price          float64 `db:"price"`
	DownloadURL    string `db:"-"`
	QRToken        string `db:"qr_token"`
	DownloadToken  string `db:"download_token"`
	// Additional fields for ticket details display
	AgencyName     string `db:"agency_name"`
	Status         string `db:"status"`
	DepartureCity  string `db:"departure_city"`
	ArrivalCity    string `db:"arrival_city"`
	DepartureTime  string `db:"departure_time"`
	ArrivalTime    string `db:"arrival_time"`
	Date           string `db:"date"`
	ArrivalDate    string `db:"arrival_date"`
	PaymentMethod  string `db:"payment_method"`
	BookingReference string `db:"booking_reference"`
	// New fields for ticket list display
	Origin         string `db:"origin"`
	TicketStatus   string `db:"ticket_status"`
	BookingStatus  string `db:"booking_status"`
}

func CreateGuestBooking(journeyID, guestName, guestEmail string, passengers []PassengerDraft, appURL, paymentMethod string) ([]CreatedTicket, string, error) {
	tx, err := DB.Beginx()
	if err != nil {
		return nil, "", err
	}
	defer tx.Rollback()

	var journey models.Journey
	if err := tx.Get(&journey, `SELECT * FROM journeys WHERE id=$1 FOR UPDATE`, journeyID); err != nil {
		return nil, "", err
	}

	for _, p := range passengers {
		var count int
		tx.Get(&count, `SELECT COUNT(*) FROM tickets WHERE journey_id=$1 AND seat_number=$2 AND status IN ('active', 'checked_in')`, journeyID, p.SeatNumber)
		if count > 0 {
			return nil, "", fmt.Errorf("le siège %s est déjà pris", p.SeatNumber)
		}
	}

	var basePrice float64
	tx.Get(&basePrice, `SELECT base_price FROM routes WHERE id=$1`, journey.RouteID)
	var destination string
	tx.Get(&destination, `SELECT destination FROM routes WHERE id=$1`, journey.RouteID)
	totalAmount := basePrice * float64(len(passengers))

	bookingRef, err := utils.GenerateReference()
	if err != nil {
		return nil, "", err
	}

	// Determine booking status based on payment method
	bookingStatus := "confirmed"
	if paymentMethod == "cash" {
		bookingStatus = "pending"
	}

	var bookingID string
	err = tx.QueryRowx(`
		INSERT INTO bookings (booking_reference, guest_email, guest_name, total_amount, status, payment_method)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`,
		bookingRef, guestEmail, guestName, totalAmount, bookingStatus, paymentMethod,
	).Scan(&bookingID)
	if err != nil {
		return nil, "", err
	}

	var tickets []CreatedTicket
	for _, p := range passengers {
		qrToken, err := utils.GenerateQRToken(bookingID+"-"+p.SeatNumber, journeyID, p.SeatNumber, QRSecret)
		if err != nil {
			return nil, "", err
		}
		tok, err := utils.GenerateDownloadToken()
		if err != nil {
			return nil, "", err
		}
		expiry := time.Now().Add(7 * 24 * time.Hour)

		_, err = tx.Exec(`
			INSERT INTO tickets (booking_id, journey_id, seat_number, passenger_name, passenger_phone, price, qr_token, download_token, download_token_expires_at, payment_validated)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
			bookingID, journeyID, p.SeatNumber, p.Name, models.NullString(p.Phone), basePrice, qrToken, tok, expiry, false,
		)
		if err != nil {
			return nil, "", err
		}

		tickets = append(tickets, CreatedTicket{
			ID:             "",
			Destination:    destination,
			SeatNumber:     p.SeatNumber,
			PassengerName:  p.Name,
			PassengerPhone: p.Phone,
			Price:          basePrice,
			DownloadURL:    fmt.Sprintf("%s/tickets/download/%s", appURL, tok),
			QRToken:        qrToken,
			DownloadToken:  tok,
		})
	}

	if err := tx.Commit(); err != nil {
		return nil, "", err
	}

	return tickets, bookingRef, nil
}

func GetBookedSeats(journeyID string) ([]string, error) {
	var seats []string
	err := DB.Select(&seats, `SELECT seat_number FROM tickets WHERE journey_id=$1 AND status IN ('active', 'checked_in')`, journeyID)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	return seats, nil
}

// GetTicketByDownloadToken retrieves a ticket by its download token.
func GetTicketByDownloadToken(downloadToken string) (*CreatedTicket, error) {
	var t CreatedTicket
	err := DB.Get(&t, `SELECT seat_number, passenger_name, passenger_phone, price, download_token, qr_token FROM tickets WHERE download_token=$1`, downloadToken)
	if err != nil {
		return nil, err
	}
	// Construct DownloadURL using AppURL
	t.DownloadURL = fmt.Sprintf("%s/tickets/download/%s", AppURL, t.DownloadToken)
	return &t, nil
}


// UpdateTicketPaymentStatus marks the ticket as payment validated (cash payment).
func UpdateTicketPaymentStatus(downloadToken string) error {
	_, err := DB.Exec(`UPDATE tickets SET payment_validated = TRUE WHERE download_token=$1`, downloadToken)
	return err
}

// UpdateTicketCheckinStatus marks the ticket as checked in.
func UpdateTicketCheckinStatus(downloadToken string) error {
	_, err := DB.Exec(`UPDATE tickets SET status = 'checked_in', checked_in_at = NOW() WHERE download_token=$1`, downloadToken)
	return err
}

type TicketDetail struct {
	OrganizationID string `db:"organization_id"`
	JourneyID      string `db:"journey_id"`
}

func GetTicketDetailsByDownloadToken(downloadToken string) (*TicketDetail, error) {
	var td TicketDetail
	// First check if the ticket exists with this download token
	var exists bool
	err := DB.Get(&exists, `SELECT EXISTS(SELECT 1 FROM tickets WHERE download_token=$1)`, downloadToken)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("no ticket found with download token: %s", downloadToken)
	}

	// Get the ticket details
	err = DB.Get(&td, `SELECT t.journey_id as journey_id, j.organization_id as organization_id FROM tickets t JOIN journeys j ON t.journey_id = j.id WHERE t.download_token=$1`, downloadToken)
	if err != nil {
		return nil, err
	}
	return &td, nil
}

// Fixed version of GetTicketDetailsByDownloadToken that properly retrieves ticket details
func GetTicketDetailsByDownloadTokenFixed(downloadToken string) (*TicketDetail, error) {
	var td TicketDetail
	// Get journey_id from ticket
	err := DB.Get(&td, `SELECT t.journey_id FROM tickets WHERE download_token=$1`, downloadToken)
	if err != nil {
		return nil, err
	}

	// Get organization_id from journey
	var orgID string
	err = DB.Get(&orgID, `SELECT j.organization_id FROM journeys j WHERE j.id = $1`, td.JourneyID)
	if err != nil {
		return nil, err
	}
	td.OrganizationID = orgID

	return &td, nil
}