package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/bus-booking/BE-bus-booking-platform/internal/models"
	"github.com/bus-booking/BE-bus-booking-platform/internal/services"
	"github.com/bus-booking/BE-bus-booking-platform/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

type GuestHandler struct {
	DB           *sqlx.DB
	QRSecret     string
	EmailService *services.EmailService
	AppURL       string
}

func NewGuestHandler(db *sqlx.DB, qrSecret string, emailSvc *services.EmailService, appURL string) *GuestHandler {
	return &GuestHandler{DB: db, QRSecret: qrSecret, EmailService: emailSvc, AppURL: appURL}
}

type GuestCheckoutRequest struct {
	JourneyID  string            `json:"journey_id" binding:"required"`
	GuestEmail string            `json:"guest_email" binding:"required"`
	GuestName  string            `json:"guest_name" binding:"required"`
	Seats      []models.BookSeat `json:"seats" binding:"required,min=1"`
}

func (h *GuestHandler) GuestCheckout(c *gin.Context) {
	var req GuestCheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx, err := h.DB.Beginx()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "transaction failed"})
		return
	}
	defer tx.Rollback()

	// Lock journey
	var journey models.Journey
	err = tx.Get(&journey, `SELECT * FROM journeys WHERE id=$1 FOR UPDATE`, req.JourneyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "journey not found"})
		return
	}

	// Check seat availability
	for _, seat := range req.Seats {
		var count int
		tx.Get(&count, `
			SELECT COUNT(*) FROM tickets WHERE journey_id=$1 AND seat_number=$2 AND status IN ('active', 'checked_in')`,
			req.JourneyID, seat.SeatNumber)
		if count > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "seat " + seat.SeatNumber + " is already booked"})
			return
		}
	}

	// Calculate price
	var basePrice float64
	tx.Get(&basePrice, `SELECT base_price FROM routes WHERE id=$1`, journey.RouteID)
	totalAmount := basePrice * float64(len(req.Seats))

	// Fetch route details for email
	var route models.Route
	tx.Get(&route, `SELECT * FROM routes WHERE id=$1`, journey.RouteID)

	// Create booking
	bookingRef, err := utils.GenerateReference()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate booking reference"})
		return
	}
	var booking models.Booking
	err = tx.QueryRowx(`
		INSERT INTO bookings (booking_reference, guest_email, guest_name, total_amount, status)
		VALUES ($1, $2, $3, $4, 'confirmed')
		RETURNING id, booking_reference, guest_email, guest_name, total_amount, status, created_at, updated_at`,
		bookingRef, req.GuestEmail, req.GuestName, totalAmount,
	).StructScan(&booking)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create booking"})
		return
	}

	// Create tickets with download tokens
	var ticketInfos []services.TicketInfo
	for _, seat := range req.Seats {
		qrToken, err := utils.GenerateQRToken(
			booking.ID+"-"+seat.SeatNumber,
			req.JourneyID,
			seat.SeatNumber,
			h.QRSecret,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate QR"})
			return
		}

		tok, tokErr := utils.GenerateDownloadToken()
		if tokErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate download token"})
			return
		}
		expiry := time.Now().Add(7 * 24 * time.Hour)

		var ticket models.Ticket
		err = tx.QueryRowx(`
			INSERT INTO tickets (booking_id, journey_id, seat_number, passenger_name, passenger_phone,
				boarding_stop_id, dropoff_stop_id, price, qr_token, download_token, download_token_expires_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)            RETURNING id, booking_id, journey_id, seat_number, passenger_name,passenger_phone,
                boarding_stop_id, dropoff_stop_id, price, qr_token, status, created_at, updated_at`,
			booking.ID, req.JourneyID, seat.SeatNumber, seat.PassengerName,
			models.NullString(seat.PassengerPhone),
			models.NullString(seat.BoardingStopID), models.NullString(seat.DropoffStopID),
			basePrice, qrToken, tok, expiry,
		).StructScan(&ticket)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create ticket"})
			return
		}
		booking.Tickets = append(booking.Tickets, ticket)

		ticketInfos = append(ticketInfos, services.TicketInfo{
			PassengerName: seat.PassengerName,
			SeatNumber:    seat.SeatNumber,
			Date:          journey.DepartureDate,
			Time:          journey.DepartureTime,
			Origin:        route.Origin,
			Destination:   route.Destination,
			Price:         fmt.Sprintf("%.0f", basePrice),
		})
	}

	tx.Commit()

	// Send email with ticket invoice & download link
	if h.EmailService != nil && len(booking.Tickets) > 0 && booking.Tickets[0].DownloadToken != nil {
		downloadLink := fmt.Sprintf("%s/tickets/download/%s", h.AppURL, *booking.Tickets[0].DownloadToken)
		if err := h.EmailService.SendTicketInvoice(req.GuestEmail, req.GuestName, bookingRef, ticketInfos, downloadLink); err != nil {
			// Log but don't fail the request — booking is already confirmed
			c.JSON(http.StatusCreated, gin.H{
				"booking":      booking,
				"email_status": "failed",
				"warning":      "booking confirmed but email could not be sent. Configure SMTP in your .env file.",
			})
			return
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"booking":      booking,
		"email_status": "sent",
	})
}
