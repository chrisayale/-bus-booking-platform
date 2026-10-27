package handlers

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"path/filepath"
	"time"

	"github.com/bus-booking/BE-bus-booking-platform/internal/models"
	"github.com/bus-booking/BE-bus-booking-platform/internal/services"
	"github.com/bus-booking/BE-bus-booking-platform/internal/utils"
	"github.com/bus-booking/BE-bus-booking-platform/internal/whatsapp"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	waTypes "go.mau.fi/whatsmeow/types"
)

type B2BHandler struct {
	DB            *sqlx.DB
	StorageSvc    services.StorageService
	JWTSecret     string
	JWTExpiry     time.Duration
	RefreshExpiry time.Duration
}

func NewB2BHandler(db *sqlx.DB, storageSvc services.StorageService, jwtSecret string, jwtExpiry, refreshExpiry time.Duration) *B2BHandler {
	return &B2BHandler{DB: db, StorageSvc: storageSvc, JWTSecret: jwtSecret, JWTExpiry: jwtExpiry, RefreshExpiry: refreshExpiry}
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
		c.JSON(http.StatusNotFound, gin.H{"error": "route not found or already processed"})
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
	err := h.DB.Select(&journeys, `
		SELECT id, organization_id, route_id, bus_id,
			departure_date::text, departure_time::text,
			arrival_date::text, arrival_time::text,
			status, created_at, updated_at
		FROM journeys WHERE organization_id = $1 ORDER BY departure_date DESC, departure_time DESC`, orgID)
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
	userID := c.GetString("user_id")
	id := c.Param("id")
	orgID := c.GetString("org_id")

	// Verify user belongs to this organization
	var memberCount int
	if err := h.DB.Get(&memberCount, `SELECT COUNT(*) FROM user_organizations WHERE user_id = $1 AND organization_id = $2`, userID, orgID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify organization membership"})
		return
	}
	if memberCount == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied: user is not a member of this organization"})
		return
	}

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

// ======================== PLATFORM ADMIN ========================

func (h *B2BHandler) ListPendingOwners(c *gin.Context) {
	var users []models.User
	err := h.DB.Select(&users, `
		SELECT id, email, full_name, phone, role, organization_id, is_verified, approval_status, created_at, updated_at
		FROM users WHERE role = 'owner' AND approval_status = 'pending'
		ORDER BY created_at DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list pending owners"})
		return
	}
	c.JSON(http.StatusOK, users)
}

func (h *B2BHandler) ApproveOwner(c *gin.Context) {
	id := c.Param("id")
	// Get owner's organization_id first
	var orgID string
	err := h.DB.Get(&orgID, `SELECT organization_id FROM users WHERE id = $1 AND role = 'owner'`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to find owner organization"})
		return
	}
	// Update user: approve, verify, activate
	_, err = h.DB.Exec(`UPDATE users SET approval_status = 'approved', is_verified = true, is_active = true, updated_at = NOW() WHERE id = $1 AND role = 'owner'`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to approve owner"})
		return
	}
	// Also activate the organization
	if orgID != "" {
		_, err = h.DB.Exec(`UPDATE organizations SET is_active = true, updated_at = NOW() WHERE id = $1`, orgID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to activate organization"})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"message": "owner approved"})
}

func (h *B2BHandler) RejectOwner(c *gin.Context) {
	id := c.Param("id")
	_, err := h.DB.Exec(`UPDATE users SET approval_status = 'rejected', updated_at = NOW() WHERE id = $1 AND role = 'owner'`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reject owner"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "owner rejected"})
}

func (h *B2BHandler) ListAgencies(c *gin.Context) {
	var orgs []models.Organization
	err := h.DB.Select(&orgs, `SELECT id, name, email, phone, logo_url, is_active, created_at, updated_at FROM organizations ORDER BY created_at DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list agencies"})
		return
	}
	c.JSON(http.StatusOK, orgs)
}

// ListPendingAgencies returns all pending agency requests awaiting approval
func (h *B2BHandler) ListPendingAgencies(c *gin.Context) {
	var pendingAgencies []struct {
		ID              string    `db:"id" json:"id"`
		AgencyName      string    `db:"agency_name" json:"agency_name"`
		ManagerName     string    `db:"manager_name" json:"manager_name"`
		Address         string    `db:"address" json:"address"`
		ContactPhone    string    `db:"contact_phone" json:"contact_phone"`
		Email           string    `db:"email" json:"email"`
		DefaultPassword string    `db:"default_password" json:"default_password"`
		Status          string    `db:"status" json:"status"`
		CreatedAt       time.Time `db:"created_at" json:"created_at"`
		UpdatedAt       time.Time `db:"updated_at" json:"updated_at"`
	}
	err := h.DB.Select(&pendingAgencies, `SELECT id, agency_name, manager_name, address, contact_phone, email, default_password, status, created_at, updated_at FROM pending_agencies WHERE status = 'pending' ORDER BY created_at DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list pending agencies"})
		return
	}
	c.JSON(http.StatusOK, pendingAgencies)
}

// ApproveAgency approves a pending agency request and creates the organization
func (h *B2BHandler) ApproveAgency(c *gin.Context) {
	id := c.Param("id")

	// Start transaction
	tx, err := h.DB.Beginx()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// Get the pending agency
	var pendingAgency struct {
		ID              string `db:"id"`
		AgencyName      string `db:"agency_name"`
		ManagerName     string `db:"manager_name"`
		Address         string `db:"address"`
		ContactPhone    string `db:"contact_phone"`
		Email           string `db:"email"`
		DefaultPassword string `db:"default_password"`
		Status          string `db:"status"`
	}

	err = tx.Get(&pendingAgency, `SELECT id, agency_name, manager_name, address, contact_phone, email, default_password, status FROM pending_agencies WHERE id = $1`, id)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "pending agency not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch pending agency"})
		}
		return
	}

	// Check if already processed
	if pendingAgency.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agency request has already been processed"})
		return
	}

	// Hash the default password
	passwordHash, err := utils.HashPassword(pendingAgency.DefaultPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	// Generate a unique 6-digit reference code for the agency
	referenceCode, err := generateUniqueReferenceCode(tx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate reference code"})
		return
	}

	// Create organization record with reference code
	var organization models.Organization
	err = tx.QueryRowx(`
		INSERT INTO organizations (name, email, phone, is_active, reference_code)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, email, phone, logo_url, is_active, reference_code, created_at, updated_at`,
		pendingAgency.AgencyName,
		pendingAgency.Email,
		pendingAgency.ContactPhone,
		true, // Set as active upon approval
		referenceCode,
	).StructScan(&organization)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create organization"})
		return
	}

	// Create owner user record for the agency manager
	var user models.User
	err = tx.QueryRowx(`
		INSERT INTO users (email, password_hash, full_name, phone, role, organization_id, is_verified, approval_status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, email, full_name, phone, role, organization_id, is_verified, approval_status, created_at, updated_at`,
		pendingAgency.Email,
		passwordHash,
		pendingAgency.ManagerName,
		pendingAgency.ContactPhone,
		"owner",
		organization.ID,
		true,       // is_verified
		"approved", // approval_status
	).StructScan(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create owner user"})
		return
	}

	// Update pending agency status to approved
	_, err = tx.Exec(`UPDATE pending_agencies SET status = 'approved', updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update pending agency status"})
		return
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit transaction"})
		return
	}

	// Send WhatsApp notification about successful approval with reference code and credentials
	go func() {
		// Format the WhatsApp message with reference code, email and password
		message := fmt.Sprintf("Votre compte a été activé avec succès !\n\n🏢 Code agence : %s\n📧 Email : %s\n🔑 Mot de passe : %s\n\nVeuillez choisir une option :\n1. Changer le mot de passe\n2. Pas maintenant\n\nRépondez avec le numéro de votre choix.", referenceCode, pendingAgency.Email, pendingAgency.DefaultPassword)

		// Send the WhatsApp message using the helper function
		jid := waTypes.NewJID(pendingAgency.ContactPhone, waTypes.DefaultUserServer)
		whatsapp.SendText(jid, message)

		// Set up the FSM state for handling the response
		whatsapp.SetSessionStep(pendingAgency.ContactPhone, whatsapp.StepPasswordChangeMenu)
	}()

	c.JSON(http.StatusOK, gin.H{
		"message":      "agency approved successfully",
		"organization": organization,
		"user":         user,
	})
}

// generateUniqueReferenceCode generates a unique 6-digit reference code
func generateUniqueReferenceCode(tx *sqlx.Tx) (string, error) {
	const maxAttempts = 10
	for i := 0; i < maxAttempts; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(1000000))
		if err != nil {
			return "", err
		}
		code := fmt.Sprintf("%06d", n.Int64())
		var exists bool
		err = tx.Get(&exists, `SELECT EXISTS(SELECT 1 FROM organizations WHERE reference_code = $1)`, code)
		if err != nil {
			return "", err
		}
		if !exists {
			return code, nil
		}
	}
	return "", fmt.Errorf("failed to generate unique reference code after %d attempts", maxAttempts)
}

// RejectAgency rejects a pending agency request
func (h *B2BHandler) RejectAgency(c *gin.Context) {
	id := c.Param("id")

	// Update the pending agency status to rejected
	result, err := h.DB.Exec(`UPDATE pending_agencies SET status = 'rejected', updated_at = NOW() WHERE id = $1 AND status = 'pending'`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reject agency"})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "pending agency not found or already processed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "agency rejected"})
}

func (h *B2BHandler) SetAgencyActive(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		IsActive bool `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Update organization is_active
	_, err := h.DB.Exec(`UPDATE organizations SET is_active = $1, updated_at = NOW() WHERE id = $2`, req.IsActive, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update agency status"})
		return
	}
	// Also update the owner user's is_active, is_verified, and approval_status
	// When activating an agency, the owner should be fully verified and approved
	if req.IsActive {
		_, err = h.DB.Exec(`UPDATE users SET is_active = true, is_verified = true, approval_status = 'approved', updated_at = NOW() WHERE organization_id = $1 AND role = 'owner'`, id)
	} else {
		_, err = h.DB.Exec(`UPDATE users SET is_active = false, updated_at = NOW() WHERE organization_id = $1 AND role = 'owner'`, id)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update owner user status"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "agency status updated"})
}

type TicketWithDetails struct {
	TicketID           string  `db:"ticket_id" json:"ticket_id"`
	PassengerName      string  `db:"passenger_name" json:"passenger_name"`
	PassengerPhone     *string `db:"passenger_phone" json:"passenger_phone,omitempty"`
	SeatNumber         string  `db:"seat_number" json:"seat_number"`
	Price              float64 `db:"price" json:"price"`
	TicketStatus       string  `db:"ticket_status" json:"ticket_status"`
	JourneyID          string  `db:"journey_id" json:"journey_id"`
	DepartureDate      string  `db:"departure_date" json:"departure_date"`
	DepartureTime      string  `db:"departure_time" json:"departure_time"`
	BusID              string  `db:"bus_id" json:"bus_id"`
	RegistrationNumber string  `db:"registration_number" json:"registration_number"`
	Origin             string  `db:"origin" json:"origin"`
	Destination        string  `db:"destination" json:"destination"`
	BookingReference   string  `db:"booking_reference" json:"booking_reference"`
	BookingStatus      string  `db:"booking_status" json:"booking_status"`
	BookedAt           string  `db:"booked_at" json:"booked_at"`
}

func (h *B2BHandler) ListOrganizationTickets(c *gin.Context) {
	userID := c.GetString("user_id")
	orgID := c.GetString("org_id")
	busID := c.Query("bus_id")
	journeyID := c.Query("journey_id")

	// Verify user belongs to this organization
	var memberCount int
	if err := h.DB.Get(&memberCount, `SELECT COUNT(*) FROM user_organizations WHERE user_id = $1 AND organization_id = $2`, userID, orgID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify organization membership"})
		return
	}
	if memberCount == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied: user is not a member of this organization"})
		return
	}

	query := `
		SELECT t.id AS ticket_id, t.passenger_name, t.passenger_phone, t.seat_number, t.price,
			t.status AS ticket_status, j.id AS journey_id, j.departure_date::text, j.departure_time::text,
			b.id AS bus_id, b.registration_number, r.origin, r.destination,
			bk.booking_reference, bk.status AS booking_status, bk.created_at::text AS booked_at
		FROM tickets t
		JOIN journeys j ON t.journey_id = j.id
		JOIN buses b ON j.bus_id = b.id
		JOIN routes r ON j.route_id = r.id
		JOIN bookings bk ON t.booking_id = bk.id
		WHERE j.organization_id = $1`

	args := []interface{}{orgID}
	if busID != "" {
		args = append(args, busID)
		query += fmt.Sprintf(" AND b.id = $%d", len(args))
	}
	if journeyID != "" {
		args = append(args, journeyID)
		query += fmt.Sprintf(" AND j.id = $%d", len(args))
	}
	query += " ORDER BY bk.created_at DESC"

	var tickets []TicketWithDetails
	if err := h.DB.Select(&tickets, query, args...); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tickets"})
		return
	}
	c.JSON(http.StatusOK, tickets)
}

func (h *B2BHandler) ImpersonateOwner(c *gin.Context) {
	orgID := c.Param("id")

	var user models.User
	err := h.DB.Get(&user, `
		SELECT * FROM users
		WHERE organization_id = $1 AND role = 'owner' AND approval_status = 'approved'
		ORDER BY created_at ASC LIMIT 1`, orgID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no approved owner found for this agency"})
		return
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
	userID := c.GetString("user_id")
	var req models.QRScanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orgID := c.GetString("org_id")

	// Verify user belongs to this organization
	var memberCount int
	if err := h.DB.Get(&memberCount, `SELECT COUNT(*) FROM user_organizations WHERE user_id = $1 AND organization_id = $2`, userID, orgID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify organization membership"})
		return
	}
	if memberCount == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied: user is not a member of this organization"})
		return
	}

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
	departureTime, err := time.Parse(time.RFC3339, departureTimeStr+"Z")
	if err != nil {
		departureTime, err = time.Parse(time.RFC3339, departureTimeStr+":00Z")
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
		"message":   "check-in successful",
		"passenger": ticket.PassengerName,
		"seat":      ticket.SeatNumber,
		"status":    "checked_in",
	})
}
