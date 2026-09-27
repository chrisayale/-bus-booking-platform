package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/bus-booking/BE-bus-booking-platform/internal/models"
	"github.com/bus-booking/BE-bus-booking-platform/internal/services"
	"github.com/bus-booking/BE-bus-booking-platform/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

type B2CHandler struct {
	DB            *sqlx.DB
	JWTSecret     string
	QRSecret      string
	JWTExpiry     time.Duration
	RefreshExpiry time.Duration
	EmailService  *services.EmailService
}

func NewB2CHandler(db *sqlx.DB, jwtSecret, qrSecret string, jwtExpiry, refreshExpiry time.Duration, emailSvc *services.EmailService) *B2CHandler {
	return &B2CHandler{
		DB: db, JWTSecret: jwtSecret, QRSecret: qrSecret,
		JWTExpiry: jwtExpiry, RefreshExpiry: refreshExpiry, EmailService: emailSvc,
	}
}

// ======================== AUTH ========================

func (h *B2CHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if email exists
	var count int
	if err := h.DB.Get(&count, `SELECT COUNT(*) FROM users WHERE email=$1`, req.Email); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check email"})
		return
	}
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		return
	}

		hash, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	role := "passenger"
	approvalStatus := "approved"
	var orgID *string

	if req.Role == "owner" {
		if req.AgencyName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "agency_name is required for owner registration"})
			return
		}
		role = "owner"
		approvalStatus = "pending"

		var newOrgID string
		err = h.DB.QueryRowx(`
			INSERT INTO organizations (name, email, phone)
			VALUES ($1, $2, $3)
			RETURNING id`,
			req.AgencyName, req.Email, models.NullString(req.Phone),
		).Scan(&newOrgID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create organization"})
			return
		}
		orgID = &newOrgID
	}

	var user models.User
	err = h.DB.QueryRowx(`
		INSERT INTO users (email, password_hash, full_name, phone, role, organization_id, approval_status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, email, full_name, phone, role, organization_id, is_verified, approval_status, created_at, updated_at`,
		req.Email, hash, req.FullName, models.NullString(req.Phone), role, orgID, approvalStatus,
	).StructScan(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	msg := "user created"
	if approvalStatus == "pending" {
		msg = "account created, pending approval"
	}
	c.JSON(http.StatusCreated, gin.H{"message": msg, "user": user})
}

func (h *B2CHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	err := h.DB.Get(&user, `SELECT * FROM users WHERE email=$1`, req.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
		return
	}

		if !utils.CheckPassword(req.Password, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
		if user.Role == "owner" && user.ApprovalStatus == "pending" {
		c.JSON(http.StatusForbidden, gin.H{"error": "your agency account is pending approval"})
		return
	}
	if user.Role == "owner" && user.ApprovalStatus == "rejected" {
		c.JSON(http.StatusForbidden, gin.H{"error": "your agency account was rejected"})
		return
	}
	if user.Role == "owner" && user.OrganizationID != nil {
		var orgActive bool
		if err := h.DB.Get(&orgActive, `SELECT is_active FROM organizations WHERE id = $1`, *user.OrganizationID); err == nil && !orgActive {
			c.JSON(http.StatusForbidden, gin.H{"error": "your agency account has been deactivated"})
			return
		}
	}
	orgID := ""
	if user.OrganizationID != nil {
		orgID = *user.OrganizationID
	}

	accessToken, err := utils.GenerateToken(h.JWTSecret, user.ID, user.Email, user.Role, orgID, h.JWTExpiry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	refreshToken, err := utils.GenerateToken(h.JWTSecret, user.ID, user.Email, user.Role, orgID, h.RefreshExpiry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate refresh token"})
		return
	}

	c.JSON(http.StatusOK, models.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
	})
}

func (h *B2CHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	claims, err := utils.ValidateToken(h.JWTSecret, req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}

	accessToken, err := utils.GenerateToken(h.JWTSecret, claims.UserID, claims.Email, claims.Role, claims.OrgID, h.JWTExpiry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"access_token": accessToken})
}

// ======================== SEARCH ========================

func (h *B2CHandler) SearchTrips(c *gin.Context) {
	var req models.SearchRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

		var journeys []models.Journey
	err := h.DB.Select(&journeys, `
		SELECT j.id, j.organization_id, j.route_id, j.bus_id,
			j.departure_date::text, j.departure_time::text,
			j.arrival_date::text, j.arrival_time::text,
			j.status, j.created_at, j.updated_at
		FROM journeys j
		JOIN routes r ON j.route_id = r.id
		WHERE r.origin ILIKE $1
		  AND r.destination ILIKE $2
		  AND j.departure_date = $3
		  AND j.status NOT IN ('cancelled', 'completed')
		ORDER BY j.departure_time`,
		"%"+req.Origin+"%", "%"+req.Destination+"%", req.Date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "search failed"})
		return
	}

	// Enrich with route, bus, and available seats
	for i := range journeys {
		var route models.Route
		if err := h.DB.Get(&route, `SELECT * FROM routes WHERE id=$1`, journeys[i].RouteID); err == nil {
			var stops []models.Stop
			h.DB.Select(&stops, `SELECT * FROM route_stops WHERE route_id=$1 ORDER BY stop_order`, route.ID)
			route.Stops = stops
			journeys[i].Route = &route
		}

				var bus models.Bus
		if err := h.DB.Get(&bus, `SELECT * FROM buses WHERE id=$1`, journeys[i].BusID); err == nil {
			journeys[i].Bus = &bus
		}

		var org models.Organization
		if err := h.DB.Get(&org, `SELECT * FROM organizations WHERE id=$1`, journeys[i].OrganizationID); err == nil {
			journeys[i].Organization = &org
		}

		// Count booked seats and store remaining as available_seats (don't mutate Capacity)
		var bookedCount int
		h.DB.Get(&bookedCount, `
			SELECT COUNT(*) FROM tickets WHERE journey_id=$1 AND status IN ('active', 'checked_in')`,
			journeys[i].ID)
		if journeys[i].Bus != nil {
			avail := journeys[i].Bus.Capacity - bookedCount
			journeys[i].Bus.AvailableSeats = &avail
		}
	}

	c.JSON(http.StatusOK, journeys)
}

// ======================== BOOKING ========================

func (h *B2CHandler) GetJourneySeats(c *gin.Context) {
	journeyID := c.Param("id")

	// Verify journey exists
	var journey models.Journey
	err := h.DB.Get(&journey, `SELECT * FROM journeys WHERE id=$1`, journeyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "journey not found"})
		return
	}

	// Get booked seat numbers
	var bookedSeats []string
	err = h.DB.Select(&bookedSeats, `
		SELECT seat_number FROM tickets
		WHERE journey_id=$1 AND status IN ('active', 'checked_in')
		ORDER BY seat_number`, journeyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch seats"})
		return
	}

	if bookedSeats == nil {
		bookedSeats = []string{}
	}

	c.JSON(http.StatusOK, gin.H{
		"booked_seats": bookedSeats,
	})
}

func (h *B2CHandler) BookTickets(c *gin.Context) {
	var req models.BookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// For guest checkout, require email
	userID := c.GetString("user_id")
	if userID == "" && (req.GuestEmail == nil || *req.GuestEmail == "") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "guest email is required for guest checkout"})
		return
	}

	tx, err := h.DB.Beginx()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "transaction failed"})
		return
	}
	defer tx.Rollback()

	// Lock journey to prevent overbooking
	var journey models.Journey
	err = tx.Get(&journey, `SELECT * FROM journeys WHERE id=$1 FOR UPDATE`, req.JourneyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "journey not found"})
		return
	}

	// Check each seat availability
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

	// Calculate total price
	basePrice := 0.0
	tx.Get(&basePrice, `SELECT base_price FROM routes WHERE id=$1`, journey.RouteID)
	totalAmount := basePrice * float64(len(req.Seats))

	// Create booking
	bookingRef, err := utils.GenerateReference()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate booking reference"})
		return
	}
	var booking models.Booking
	err = tx.QueryRowx(`
		INSERT INTO bookings (user_id, booking_reference, guest_email, guest_name, total_amount, status)
		VALUES ($1, $2, $3, $4, $5, 'confirmed')
		RETURNING id, user_id, booking_reference, guest_email, guest_name, total_amount, status, created_at, updated_at`,
		models.NullString(userID), bookingRef, req.GuestEmail, req.GuestName, totalAmount,
	).StructScan(&booking)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create booking"})
		return
	}

	// Create tickets
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

		var downloadToken *string
		var downloadExpiry *time.Time
		if req.GuestEmail != nil {
			tok, tokErr := utils.GenerateDownloadToken()
			if tokErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate download token"})
				return
			}
			downloadToken = &tok
			expiry := time.Now().Add(7 * 24 * time.Hour)
			downloadExpiry = &expiry
		}

		var ticket models.Ticket
		err = tx.QueryRowx(`
			INSERT INTO tickets (booking_id, journey_id, seat_number, passenger_name, passenger_phone,
				boarding_stop_id, dropoff_stop_id, price, qr_token, download_token, download_token_expires_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)            RETURNING id, booking_id, journey_id, seat_number, passenger_name, passenger_phone,
                boarding_stop_id, dropoff_stop_id, price, qr_token, status, created_at, updated_at`,
			booking.ID, req.JourneyID, seat.SeatNumber, seat.PassengerName,
			models.NullString(seat.PassengerPhone),
			models.NullString(seat.BoardingStopID), models.NullString(seat.DropoffStopID),
			basePrice, qrToken, downloadToken, downloadExpiry,
		).StructScan(&ticket)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create ticket: " + err.Error()})
			return
		}
		booking.Tickets = append(booking.Tickets, ticket)
	}

	tx.Commit()

	c.JSON(http.StatusCreated, booking)
}

// ======================== TICKET MANAGEMENT ========================

func (h *B2CHandler) UpdateTicket(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("user_id")

	var req struct {
		PassengerName  *string `json:"passenger_name"`
		PassengerPhone *string `json:"passenger_phone"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify ticket belongs to this user's booking
	var ticket models.Ticket
	err := h.DB.Get(&ticket, `
		SELECT t.* FROM tickets t
		JOIN bookings b ON t.booking_id = b.id
		WHERE t.id = $1 AND b.user_id = $2 AND t.status = 'active'`, id, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found or cannot be edited"})
		return
	}

	// Build dynamic update
	name := ticket.PassengerName
	phone := ticket.PassengerPhone
	if req.PassengerName != nil {
		name = *req.PassengerName
	}
	if req.PassengerPhone != nil {
		phone = req.PassengerPhone
	}

	_, err = h.DB.Exec(`
		UPDATE tickets SET passenger_name=$1, passenger_phone=$2, updated_at=NOW()
		WHERE id=$3`, name, phone, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update ticket"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ticket updated"})
}

func (h *B2CHandler) CancelTicket(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("user_id")

	// Verify ticket belongs to this user's booking
	var ticket models.Ticket
	err := h.DB.Get(&ticket, `
		SELECT t.* FROM tickets t
		JOIN bookings b ON t.booking_id = b.id
		WHERE t.id = $1 AND b.user_id = $2 AND t.status IN ('active', 'checked_in', 'missed')`, id, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found, already cancelled, or cannot be cancelled"})
		return
	}

	_, err = h.DB.Exec(`UPDATE tickets SET status='cancelled', updated_at=NOW() WHERE id=$1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel ticket"})
		return
	}

	// If all tickets in the booking are cancelled, auto-cancel the booking too
	var remainingCount int
	h.DB.Get(&remainingCount, `SELECT COUNT(*) FROM tickets WHERE booking_id=$1 AND status NOT IN ('cancelled', 'refunded')`, ticket.BookingID)
	if remainingCount == 0 {
		h.DB.Exec(`UPDATE bookings SET status='cancelled', updated_at=NOW() WHERE id=$1`, ticket.BookingID)
	}

	c.JSON(http.StatusOK, gin.H{"message": "ticket cancelled"})
}

func (h *B2CHandler) DeleteBooking(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("user_id")

	var booking models.Booking
	err := h.DB.Get(&booking, `SELECT * FROM bookings WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "booking not found"})
		return
	}

	// Only allow deleting cancelled bookings
	if booking.Status != "cancelled" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only cancelled bookings can be deleted"})
		return
	}

	// Tickets cascade on delete via FK
	_, err = h.DB.Exec(`DELETE FROM bookings WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete booking"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "booking deleted"})
}

func (h *B2CHandler) GetUserBookings(c *gin.Context) {
	userID := c.GetString("user_id")
	var bookings []models.Booking
	err := h.DB.Select(&bookings, `
		SELECT * FROM bookings WHERE user_id=$1 ORDER BY created_at DESC`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch bookings"})
		return
	}

	for i := range bookings {
		var tickets []models.Ticket
		h.DB.Select(&tickets, `
			SELECT t.* FROM tickets t
			WHERE t.booking_id=$1`, bookings[i].ID)

		// Enrich tickets with journey & route data (1 query for the booking's journey)
				if len(tickets) > 0 {
						var journey models.Journey
			if err := h.DB.Get(&journey, `
				SELECT id, organization_id, route_id, bus_id,
					departure_date::text, departure_time::text,
					arrival_date::text, arrival_time::text,
					status, created_at, updated_at
				FROM journeys WHERE id=$1`, tickets[0].JourneyID); err == nil {
				var route models.Route
				if err := h.DB.Get(&route, `SELECT * FROM routes WHERE id=$1`, journey.RouteID); err == nil {
					journey.Route = &route
				}
				var bus models.Bus
				if err := h.DB.Get(&bus, `SELECT * FROM buses WHERE id=$1`, journey.BusID); err == nil {
					journey.Bus = &bus
				}
				var org models.Organization
				if err := h.DB.Get(&org, `SELECT * FROM organizations WHERE id=$1`, journey.OrganizationID); err == nil {
					journey.Organization = &org
				}
				for j := range tickets {
					tickets[j].Journey = &journey
				}
			}
		}

		bookings[i].Tickets = tickets
	}
	c.JSON(http.StatusOK, bookings)
}

func (h *B2CHandler) DownloadTicket(c *gin.Context) {
	token := c.Param("token")

	var ticket models.Ticket
	err := h.DB.Get(&ticket, `SELECT * FROM tickets WHERE download_token=$1`, token)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "invalid download token"})
		return
	}

	if ticket.DownloadTokenExpiresAt != nil && time.Now().After(*ticket.DownloadTokenExpiresAt) {
		c.JSON(http.StatusGone, gin.H{"error": "download link expired"})
		return
	}

	// One-time use: invalidate the download token
	h.DB.Exec(`UPDATE tickets SET download_token=NULL, download_token_expires_at=NULL, updated_at=NOW() WHERE id=$1`, ticket.ID)

	c.JSON(http.StatusOK, gin.H{
		"ticket":    ticket,
		"qr_token":  ticket.QRToken,
		"journey_id": ticket.JourneyID,
	})
}

func (h *B2CHandler) CancelBooking(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("user_id")

	var booking models.Booking
	err := h.DB.Get(&booking, `SELECT * FROM bookings WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "booking not found"})
		return
	}

	if booking.Status != "confirmed" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "booking cannot be cancelled"})
		return
	}

	_, err = h.DB.Exec(`UPDATE bookings SET status='cancelled', updated_at=NOW() WHERE id=$1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel booking"})
		return
	}

	h.DB.Exec(`UPDATE tickets SET status='cancelled', updated_at=NOW() WHERE booking_id=$1`, id)

	c.JSON(http.StatusOK, gin.H{"message": "booking cancelled"})
}

// ======================== EMAIL VERIFICATION ========================

func (h *B2CHandler) SendVerificationEmail(c *gin.Context) {
	userID := c.GetString("user_id")
	email := c.GetString("email")

	var user models.User
	if err := h.DB.Get(&user, `SELECT * FROM users WHERE id=$1`, userID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if user.IsVerified {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email already verified"})
		return
	}

	token, err := services.GenerateSecureToken(32)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	expiry := time.Now().Add(24 * time.Hour)
	_, err = h.DB.Exec(`UPDATE users SET verification_token=$1, verification_token_expires_at=$2, updated_at=NOW() WHERE id=$3`,
		token, expiry, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save token"})
		return
	}

	if h.EmailService != nil {
		if err := h.EmailService.SendVerificationEmail(email, user.FullName, token); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send email"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "verification email sent"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "verification token saved",
		"warning": "SMTP is not configured. Set SMTP_HOST and SMTP_PORT in your .env file to send emails.",
	})
}

func (h *B2CHandler) VerifyEmail(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing token"})
		return
	}

	var user models.User
	err := h.DB.Get(&user, `SELECT * FROM users WHERE verification_token=$1`, token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired verification token"})
		return
	}

	if user.VerificationTokenExpiresAt != nil && time.Now().After(*user.VerificationTokenExpiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "verification token expired"})
		return
	}

	_, err = h.DB.Exec(`UPDATE users SET is_verified=true, verification_token=NULL, verification_token_expires_at=NULL, updated_at=NOW() WHERE id=$1`,
		user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify email"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "email verified successfully"})
}

// ======================== PASSWORD RESET ========================

func (h *B2CHandler) ForgotPassword(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	err := h.DB.Get(&user, `SELECT * FROM users WHERE email=$1`, req.Email)
	if err != nil {
		// Don't reveal if email exists
		c.JSON(http.StatusOK, gin.H{"message": "if the email exists, a reset link has been sent"})
		return
	}

	token, err := services.GenerateSecureToken(32)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	expiry := time.Now().Add(1 * time.Hour)
	_, err = h.DB.Exec(`UPDATE users SET reset_token=$1, reset_token_expires_at=$2, updated_at=NOW() WHERE id=$3`,
		token, expiry, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save token"})
		return
	}

	if h.EmailService != nil {
		if err := h.EmailService.SendPasswordResetEmail(user.Email, user.FullName, token); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send email"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "if the email exists, a reset link has been sent"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "reset link saved",
		"warning": "SMTP is not configured. Set SMTP_HOST and SMTP_PORT in your .env file to send emails.",
	})
}

func (h *B2CHandler) ResetPassword(c *gin.Context) {
	var req struct {
		Token       string `json:"token" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	err := h.DB.Get(&user, `SELECT * FROM users WHERE reset_token=$1`, req.Token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired reset token"})
		return
	}

	if user.ResetTokenExpiresAt != nil && time.Now().After(*user.ResetTokenExpiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "reset token expired"})
		return
	}

	hash, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	_, err = h.DB.Exec(`UPDATE users SET password_hash=$1, reset_token=NULL, reset_token_expires_at=NULL, updated_at=NOW() WHERE id=$2`,
		hash, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reset password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password reset successfully"})
}

// ======================== USER SETTINGS ========================

func (h *B2CHandler) GetProfile(c *gin.Context) {
	userID := c.GetString("user_id")
	var user models.User
	err := h.DB.Get(&user, `SELECT id, email, full_name, phone, role, is_verified, created_at FROM users WHERE id=$1`, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *B2CHandler) UpdateProfile(c *gin.Context) {
	userID := c.GetString("user_id")
	var req struct {
		FullName string `json:"full_name"`
		Phone    string `json:"phone"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := h.DB.Exec(`UPDATE users SET full_name=COALESCE(NULLIF($1, ''), full_name), phone=COALESCE(NULLIF($2, ''), phone), updated_at=NOW() WHERE id=$3`,
		req.FullName, req.Phone, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update profile"})
		return
	}

	var user models.User
	h.DB.Get(&user, `SELECT id, email, full_name, phone, role, is_verified, created_at FROM users WHERE id=$1`, userID)
	c.JSON(http.StatusOK, user)
}

func (h *B2CHandler) ChangePassword(c *gin.Context) {
	userID := c.GetString("user_id")
	var req struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	err := h.DB.Get(&user, `SELECT * FROM users WHERE id=$1`, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if !utils.CheckPassword(req.CurrentPassword, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "current password is incorrect"})
		return
	}

	hash, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	_, err = h.DB.Exec(`UPDATE users SET password_hash=$1, updated_at=NOW() WHERE id=$2`, hash, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password changed successfully"})
}

func (h *B2CHandler) DeleteAccount(c *gin.Context) {
	userID := c.GetString("user_id")
	role := c.GetString("role")

	// Prevent deleting the last owner account
	if role == "owner" {
		var count int
		h.DB.Get(&count, `SELECT COUNT(*) FROM users WHERE role='owner'`)
		if count <= 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete the only owner account"})
			return
		}
	}

	// Archive or delete bookings first
	h.DB.Exec(`UPDATE bookings SET status='cancelled', user_id=NULL WHERE user_id=$1`, userID)
	h.DB.Exec(`UPDATE tickets SET status='cancelled' FROM bookings WHERE bookings.id=tickets.booking_id AND bookings.user_id=$1`, userID)

	_, err := h.DB.Exec(`DELETE FROM users WHERE id=$1`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete account"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "account deleted successfully"})
}
