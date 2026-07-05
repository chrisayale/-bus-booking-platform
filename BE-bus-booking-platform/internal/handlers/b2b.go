package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"time"

	"github.com/bus-booking/BE-bus-booking-platform/internal/models"
	"github.com/bus-booking/BE-bus-booking-platform/internal/services"
	"github.com/bus-booking/BE-bus-booking-platform/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

type B2BHandler struct {
	DB          *sqlx.DB
	StorageSvc  services.StorageService
}

func NewB2BHandler(db *sqlx.DB, storageSvc services.StorageService) *B2BHandler {
	return &B2BHandler{DB: db, StorageSvc: storageSvc}
}

// ======================== BUSES ========================

func (h *B2BHandler) CreateBus(c *gin.Context) {
	var req models.CreateBusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orgID := c.GetString("org_id")
	if orgID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "no organization found"})
		return
	}

	seatCfg := req.SeatConfiguration
	if seatCfg == "" {
		cfg, _ := json.Marshal(models.SeatConfig{Rows: 10, Columns: 4, Layout: "2+2"})
		seatCfg = string(cfg)
	}

	var bus models.Bus
	err := h.DB.QueryRowx(`
		INSERT INTO buses (organization_id, registration_number, capacity, seat_configuration, image_url, driver_name)
		VALUES ($1, $2, $3, $4::jsonb, $5, $6)
		RETURNING id, organization_id, registration_number, capacity, seat_configuration, is_active, image_url, driver_name, created_at, updated_at`,
		orgID, req.RegistrationNumber, req.Capacity, seatCfg, models.NullString(req.ImageURL), models.NullString(req.DriverName),
	).StructScan(&bus)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create bus: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, bus)
}

func (h *B2BHandler) ListBuses(c *gin.Context) {
	orgID := c.GetString("org_id")
	var buses []models.Bus
	err := h.DB.Select(&buses, `SELECT * FROM buses WHERE organization_id = $1 ORDER BY created_at DESC`, orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list buses"})
		return
	}
	c.JSON(http.StatusOK, buses)
}

func (h *B2BHandler) GetBus(c *gin.Context) {
	id := c.Param("id")
	orgID := c.GetString("org_id")
	var bus models.Bus
	err := h.DB.Get(&bus, `SELECT * FROM buses WHERE id = $1 AND organization_id = $2`, id, orgID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "bus not found"})
		return
	}
	c.JSON(http.StatusOK, bus)
}

func (h *B2BHandler) UpdateBus(c *gin.Context) {
	id := c.Param("id")
	orgID := c.GetString("org_id")

	var req models.UpdateBusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// If seat_configuration is not provided, pass an empty string and let COALESCE
	// in the SQL preserve the existing column value. Never apply a hardcoded default.
	seatCfg := req.SeatConfiguration

	var bus models.Bus
	err := h.DB.QueryRowx(`
		UPDATE buses SET
			registration_number=$1,
			capacity=$2,
			seat_configuration=COALESCE(NULLIF($3, ''), seat_configuration)::jsonb,
			image_url=$4,
			driver_name=$5,
			updated_at=NOW()
		WHERE id=$6 AND organization_id=$7
		RETURNING id, organization_id, registration_number, capacity, seat_configuration, is_active, image_url, driver_name, created_at, updated_at`,
		req.RegistrationNumber, req.Capacity, seatCfg, models.NullString(req.ImageURL), models.NullString(req.DriverName), id, orgID,
	).StructScan(&bus)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "bus not found"})
		return
	}
	c.JSON(http.StatusOK, bus)
}

func (h *B2BHandler) DeleteBus(c *gin.Context) {
	id := c.Param("id")
	orgID := c.GetString("org_id")
	result, err := h.DB.Exec(`DELETE FROM buses WHERE id = $1 AND organization_id = $2`, id, orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete bus"})
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "bus not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "bus deleted"})
}

// ======================== ROUTES ========================

func (h *B2BHandler) CreateRoute(c *gin.Context) {
	var req models.CreateRouteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orgID := c.GetString("org_id")

	tx, err := h.DB.Beginx()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "transaction failed"})
		return
	}
	defer tx.Rollback()

	var route models.Route
	err = tx.QueryRowx(`
		INSERT INTO routes (organization_id, name, origin, destination, base_price)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, organization_id, name, origin, destination, base_price, is_active, created_at, updated_at`,
		orgID, req.Name, req.Origin, req.Destination, req.BasePrice,
	).StructScan(&route)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create route: " + err.Error()})
		return
	}

	for _, stop := range req.Stops {
		var s models.Stop
		err := tx.QueryRowx(`
			INSERT INTO route_stops (route_id, stop_order, location, arrival_offset_minutes, departure_offset_minutes, stop_price)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id, route_id, stop_order, location, arrival_offset_minutes, departure_offset_minutes, stop_price, created_at`,
			route.ID, stop.StopOrder, stop.Location, stop.ArrivalOffsetMinutes, stop.DepartureOffsetMinutes, stop.StopPrice,
		).StructScan(&s)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create stop: " + err.Error()})
			return
		}
		route.Stops = append(route.Stops, s)
	}

	tx.Commit()
	c.JSON(http.StatusCreated, route)
}

func (h *B2BHandler) ListRoutes(c *gin.Context) {
	orgID := c.GetString("org_id")
	var routes []models.Route
	err := h.DB.Select(&routes, `SELECT * FROM routes WHERE organization_id = $1 ORDER BY created_at DESC`, orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list routes"})
		return
	}

	// Fetch stops for each route
	for i := range routes {
		var stops []models.Stop
		h.DB.Select(&stops, `SELECT * FROM route_stops WHERE route_id = $1 ORDER BY stop_order`, routes[i].ID)
		routes[i].Stops = stops
	}

	c.JSON(http.StatusOK, routes)
}

func (h *B2BHandler) GetRoute(c *gin.Context) {
	id := c.Param("id")
	orgID := c.GetString("org_id")
	var route models.Route
	err := h.DB.Get(&route, `SELECT * FROM routes WHERE id = $1 AND organization_id = $2`, id, orgID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "route not found"})
		return
	}
	var stops []models.Stop
	h.DB.Select(&stops, `SELECT * FROM route_stops WHERE route_id = $1 ORDER BY stop_order`, route.ID)
	route.Stops = stops
	c.JSON(http.StatusOK, route)
}

func (h *B2BHandler) DeleteRoute(c *gin.Context) {
	id := c.Param("id")
	orgID := c.GetString("org_id")
	result, err := h.DB.Exec(`DELETE FROM routes WHERE id = $1 AND organization_id = $2`, id, orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete route"})
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "route not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "route deleted"})
}

// ======================== JOURNEYS ========================

func (h *B2BHandler) CreateJourney(c *gin.Context) {
	var req models.CreateJourneyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orgID := c.GetString("org_id")

	// Verify bus and route belong to this org
	var count int
	h.DB.Get(&count, `SELECT COUNT(*) FROM buses WHERE id=$1 AND organization_id=$2`, req.BusID, orgID)
	if count == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bus not found in your organization"})
		return
	}
	h.DB.Get(&count, `SELECT COUNT(*) FROM routes WHERE id=$1 AND organization_id=$2`, req.RouteID, orgID)
	if count == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "route not found in your organization"})
		return
	}

	var journey models.Journey
	err := h.DB.QueryRowx(`
		INSERT INTO journeys (organization_id, route_id, bus_id, departure_date, departure_time, arrival_date, arrival_time)
		VALUES ($1, $2, $3, $4, $5::time, $6, $7::time)
		RETURNING id, organization_id, route_id, bus_id, departure_date, departure_time, arrival_date, arrival_time, status, created_at, updated_at`,
		orgID, req.RouteID, req.BusID, req.DepartureDate, req.DepartureTime, req.ArrivalDate, req.ArrivalTime,
	).StructScan(&journey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create journey: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, journey)
}

func (h *B2BHandler) ListJourneys(c *gin.Context) {
	orgID := c.GetString("org_id")
	var journeys []models.Journey
	err := h.DB.Select(&journeys, `SELECT * FROM journeys WHERE organization_id = $1 ORDER BY departure_date DESC, departure_time DESC`, orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list journeys"})
		return
	}

	// Enrich with route and bus data
	for i := range journeys {
		var route models.Route
		if err := h.DB.Get(&route, `SELECT id, name, origin, destination FROM routes WHERE id=$1`, journeys[i].RouteID); err == nil {
			journeys[i].Route = &route
		}
		var bus models.Bus
		if err := h.DB.Get(&bus, `SELECT id, registration_number, capacity FROM buses WHERE id=$1`, journeys[i].BusID); err == nil {
			journeys[i].Bus = &bus
		}
	}

	c.JSON(http.StatusOK, journeys)
}

func (h *B2BHandler) UpdateJourneyStatus(c *gin.Context) {
	id := c.Param("id")
	orgID := c.GetString("org_id")

	var req struct {
		Status string `json:"status" binding:"required,oneof=scheduled boarding departed completed cancelled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.DB.Exec(`UPDATE journeys SET status=$1, updated_at=NOW() WHERE id=$2 AND organization_id=$3`,
		req.Status, id, orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update journey"})
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "journey not found"})
		return
	}

	// If journey departed, mark unscanned tickets as missed
	if req.Status == "departed" {
		h.DB.Exec(`
			UPDATE tickets SET status='missed', updated_at=NOW()
			WHERE journey_id=$1 AND status='active'`, id)
	}

	c.JSON(http.StatusOK, gin.H{"message": "journey status updated"})
}

// ======================== IMAGE UPLOAD ========================

func (h *B2BHandler) UploadBusImage(c *gin.Context) {
	id := c.Param("id")
	orgID := c.GetString("org_id")

	// Verify bus belongs to this org
	var bus models.Bus
	if err := h.DB.Get(&bus, `SELECT * FROM buses WHERE id=$1 AND organization_id=$2`, id, orgID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "bus not found"})
		return
	}

	// Limit upload size to 10MB
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 10<<20)

	file, header, err := c.Request.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no image file provided or file too large"})
		return
	}
	defer file.Close()

	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = ".jpg"
	}

	fileName := "bus-" + id + ext
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
		return
	}

	// Upload via storage service using bytes.Reader (safe, no goroutine leak)
	ctx := c.Request.Context()
	imageURL, err := h.StorageSvc.Upload(ctx, fileName, io.NopCloser(bytes.NewReader(fileBytes)))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload image: " + err.Error()})
		return
	}

	// Update bus record
	_, err = h.DB.Exec(`UPDATE buses SET image_url=$1, updated_at=NOW() WHERE id=$2 AND organization_id=$3`, imageURL, id, orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update bus"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"image_url": imageURL})
}

// ======================== ADMIN MANAGEMENT ========================

func (h *B2BHandler) CreateAdminUser(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required,min=6"`
		FullName string `json:"full_name" binding:"required"`
		Phone    string `json:"phone"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if email exists
	var count int
	h.DB.Get(&count, `SELECT COUNT(*) FROM users WHERE email=$1`, req.Email)
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		return
	}

	hash, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	// Get the creating admin's org ID to assign the new admin to the same org
	creatingOrgID := c.GetString("org_id")

	var user models.User
	err = h.DB.QueryRowx(`
		INSERT INTO users (email, password_hash, full_name, phone, role, organization_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, email, full_name, phone, role, organization_id, is_verified, created_at, updated_at`,
		req.Email, hash, req.FullName, models.NullString(req.Phone), "owner", models.NullString(creatingOrgID),
	).StructScan(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create admin user"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "admin user created", "user": user})
}

func (h *B2BHandler) ListAdminUsers(c *gin.Context) {
	orgID := c.GetString("org_id")

	var users []models.User
	err := h.DB.Select(&users, `
		SELECT id, email, full_name, phone, role, organization_id, is_verified, created_at, updated_at
		FROM users WHERE organization_id = $1 AND role = 'owner'
		ORDER BY created_at DESC`, orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list admin users"})
		return
	}

	c.JSON(http.StatusOK, users)
}

func (h *B2BHandler) DeleteAdminUser(c *gin.Context) {
	userID := c.Param("id")
	orgID := c.GetString("org_id")
	currentUserID := c.GetString("user_id")

	// Prevent deleting yourself
	if userID == currentUserID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete your own account"})
		return
	}

	// Verify the target user is an owner in the same org
	var targetUser models.User
	err := h.DB.Get(&targetUser, `
		SELECT id, email, full_name, phone, role, organization_id, is_verified, created_at, updated_at
		FROM users WHERE id = $1 AND organization_id = $2 AND role = 'owner'`, userID, orgID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "admin user not found in your organization"})
		return
	}

	// Count remaining admins in this org to prevent deleting the last one
	var adminCount int
	h.DB.Get(&adminCount, `SELECT COUNT(*) FROM users WHERE organization_id = $1 AND role = 'owner'`, orgID)
	if adminCount <= 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete the only admin account in the organization"})
		return
	}

	// Soft-remove: delete the user (or set role to passenger? We'll delete since user asked for removal)
	result, err := h.DB.Exec(`DELETE FROM users WHERE id = $1 AND organization_id = $2 AND role = 'owner'`, userID, orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete admin user"})
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "admin user not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "admin user deleted successfully"})
}

// ======================== QR SCANNER ========================

func (h *B2BHandler) ScanTicket(c *gin.Context) {
	var req models.QRScanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orgID := c.GetString("org_id")

	// Validate QR token
	var ticket models.Ticket
	err := h.DB.Get(&ticket, `SELECT * FROM tickets WHERE qr_token = $1`, req.QRToken)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "invalid ticket"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "scan failed"})
		return
	}

	// Validate journey belongs to this org
	var journey models.Journey
	err = h.DB.Get(&journey, `SELECT * FROM journeys WHERE id = $1 AND organization_id = $2`, req.JourneyID, orgID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "ticket belongs to another company"})
		return
	}

	// Check ticket belongs to this journey
	if ticket.JourneyID != req.JourneyID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ticket is for a different journey"})
		return
	}

	// Check ticket status
	switch ticket.Status {
	case "checked_in":
		c.JSON(http.StatusConflict, gin.H{"error": "ticket already scanned"})
		return
	case "missed":
		c.JSON(http.StatusBadRequest, gin.H{"error": "ticket is expired (missed journey)"})
		return
	case "cancelled":
		c.JSON(http.StatusBadRequest, gin.H{"error": "ticket is cancelled"})
		return
	}

	// Check if journey already departed (support both "HH:MM:SS" and "HH:MM" formats)
	departureTimeStr := journey.DepartureDate + "T" + journey.DepartureTime
	// Try with the time as-is first (handles HH:MM:SS from PG), then append :00 (handles HH:MM from HTML input)
	departureTime, err := time.Parse(time.RFC3339, departureTimeStr + "Z")
	if err != nil {
		departureTime, err = time.Parse(time.RFC3339, departureTimeStr + ":00Z")
	}
	if err == nil && time.Now().After(departureTime.Add(30*time.Minute)) {
		// More than 30 min after departure? Too late
		if journey.Status == "departed" || journey.Status == "completed" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ticket is expired (journey already departed)"})
			return
		}
	}

	// Mark as checked in
	_, err = h.DB.Exec(`UPDATE tickets SET status='checked_in', checked_in_at=NOW(), updated_at=NOW() WHERE id=$1`, ticket.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check in ticket"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "check-in successful",
		"passenger":  ticket.PassengerName,
		"seat":       ticket.SeatNumber,
		"status":     "checked_in",
	})
}
