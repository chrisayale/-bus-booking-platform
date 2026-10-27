package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

// Organization represents the bus company
type Organization struct {
	ID            string    `db:"id" json:"id"`
	Name          string    `db:"name" json:"name"`
	Email         string    `db:"email" json:"email"`
	Phone         *string   `db:"phone" json:"phone,omitempty"`
	LogoURL       *string   `db:"logo_url" json:"logo_url,omitempty"`
	IsActive      bool      `db:"is_active" json:"is_active"`
	ReferenceCode *string   `db:"reference_code" json:"reference_code,omitempty"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
}

// User represents both owner and passenger accounts
type User struct {
	ID                         string     `db:"id" json:"id"`
	Email                      string     `db:"email" json:"email"`
	PasswordHash               string     `db:"password_hash" json:"-"`
	FullName                   string     `db:"full_name" json:"full_name"`
	Phone                      *string    `db:"phone" json:"phone,omitempty"`
	Role                       string     `db:"role" json:"role"`
	OrganizationID             *string    `db:"organization_id" json:"organization_id,omitempty"`
	Organizations              []Organization `db:"-" json:"organizations,omitempty"`
	IsVerified                 bool       `db:"is_verified" json:"is_verified"`
	ApprovalStatus             string     `db:"approval_status" json:"approval_status"`
	VerificationToken          *string    `db:"verification_token" json:"-"`
	VerificationTokenExpiresAt *time.Time `db:"verification_token_expires_at" json:"-"`
	ResetToken                 *string    `db:"reset_token" json:"-"`
	ResetTokenExpiresAt        *time.Time `db:"reset_token_expires_at" json:"-"`
	CreatedAt                  time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt                  time.Time  `db:"updated_at" json:"updated_at"`
	Pin                        *string    `db:"pin" json:"pin,omitempty"` // PIN for agents (plaintext, for display)
	IsActive      bool      `db:"is_active" json:"is_active"`
}

// Bus represents a vehicle in the fleet
type Bus struct {
	ID                 string          `db:"id" json:"id"`
	OrganizationID     string          `db:"organization_id" json:"organization_id"`
	RegistrationNumber string          `db:"registration_number" json:"registration_number"`
	Capacity           int             `db:"capacity" json:"capacity"`
	SeatConfiguration  json.RawMessage `db:"seat_configuration" json:"seat_configuration"`
	IsActive           bool            `db:"is_active" json:"is_active"`
	ImageURL           *string         `db:"image_url" json:"image_url,omitempty"`
	DriverName         *string         `db:"driver_name" json:"driver_name,omitempty"`
	CreatedAt          time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt          time.Time       `db:"updated_at" json:"updated_at"`
	AvailableSeats     *int            `db:"-" json:"available_seats,omitempty"`
}

// SeatConfig is a typed helper for seat_configuration JSON
type SeatConfig struct {
	Rows    int    `json:"rows"`
	Columns int    `json:"columns"`
	Layout  string `json:"layout"`
}

// Route represents an origin → destination path
type Route struct {
	ID             string    `db:"id" json:"id"`
	OrganizationID string    `db:"organization_id" json:"organization_id"`
	Name           string    `db:"name" json:"name"`
	Origin         string    `db:"origin" json:"origin"`
	Destination    string    `db:"destination" json:"destination"`
	BasePrice      float64   `db:"base_price" json:"base_price"`
	IsActive       bool      `db:"is_active" json:"is_active"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at" json:"updated_at"`
	Stops          []Stop    `db:"-" json:"stops,omitempty"`
}

// Stop represents an intermediary stop on a route
type Stop struct {
	ID                     string    `db:"id" json:"id"`
	RouteID                string    `db:"route_id" json:"route_id"`
	StopOrder              int       `db:"stop_order" json:"stop_order"`
	Location               string    `db:"location" json:"location"`
	ArrivalOffsetMinutes   *int      `db:"arrival_offset_minutes" json:"arrival_offset_minutes,omitempty"`
	DepartureOffsetMinutes *int      `db:"departure_offset_minutes" json:"departure_offset_minutes,omitempty"`
	StopPrice              float64   `db:"stop_price" json:"stop_price"`
	CreatedAt              time.Time `db:"created_at" json:"created_at"`
}

// Journey represents a specific bus trip on a route
type Journey struct {
	ID             string        `db:"id" json:"id"`
	OrganizationID string        `db:"organization_id" json:"organization_id"`
	RouteID        string        `db:"route_id" json:"route_id"`
	BusID          string        `db:"bus_id" json:"bus_id"`
	DepartureDate  string        `db:"departure_date" json:"departure_date"`
	DepartureTime  string        `db:"departure_time" json:"departure_time"`
	ArrivalDate    string        `db:"arrival_date" json:"arrival_date"`
	ArrivalTime    string        `db:"arrival_time" json:"arrival_time"`
	Status         string        `db:"status" json:"status"`
	CreatedAt      time.Time     `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time     `db:"updated_at" json:"updated_at"`
	Route          *Route        `db:"-" json:"route,omitempty"`
	Bus            *Bus          `db:"-" json:"bus,omitempty"`
	Organization   *Organization `db:"-" json:"organization,omitempty"`
}

// Booking represents a ticket purchase
type Booking struct {
	ID               string     `db:"id" json:"id"`
	UserID           *string    `db:"user_id" json:"user_id,omitempty"`
	BookingReference string     `db:"booking_reference" json:"booking_reference"`
	GuestEmail       *string    `db:"guest_email" json:"guest_email,omitempty"`
	GuestName        *string    `db:"guest_name" json:"guest_name,omitempty"`
	TotalAmount      float64    `db:"total_amount" json:"total_amount"`
	Status           string     `db:"status" json:"status"`
	PaymentMethod    string     `db:"payment_method" json:"payment_method,omitempty"`
	ExpiresAt        *time.Time `db:"expires_at" json:"expires_at,omitempty"`
	FirstPassengerName string   `db:"first_passenger_name" json:"-"`
	CreatedAt        time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time  `db:"updated_at" json:"updated_at"`
	Tickets          []Ticket   `db:"-" json:"tickets,omitempty"`
	// Additional fields for controller payment validation queries
	JourneyID         string   `db:"journey_id" json:"-"`
	DepartureDate     string   `db:"departure_date" json:"-"`
	DepartureTime     string   `db:"departure_time" json:"-"`
	ArrivalDate       string   `db:"arrival_date" json:"-"`
	ArrivalTime       string   `db:"arrival_time" json:"-"`
	RouteID           string   `db:"route_id" json:"-"`
	Origin            string   `db:"origin" json:"-"`
	Destination       string   `db:"destination" json:"-"`
	BasePrice         float64  `db:"base_price" json:"-"`
	OrganizationID    string   `db:"organization_id" json:"-"`
	OrganizationName  string   `db:"organization_name" json:"-"`
	TicketIDs         string   `db:"ticket_ids" json:"-"`
	PassengerNames    string   `db:"passenger_names" json:"-"`
	PassengerPhones   string   `db:"passenger_phones" json:"-"`
	SeatNumbers       string   `db:"seat_numbers" json:"-"`
}

// Ticket represents a single seat on a journey
type Ticket struct {
	ID                     string     `db:"id" json:"id"`
	BookingID              string     `db:"booking_id" json:"booking_id"`
	JourneyID              string     `db:"journey_id" json:"journey_id"`
	SeatNumber             string     `db:"seat_number" json:"seat_number"`
	PassengerName          string     `db:"passenger_name" json:"passenger_name"`
	PassengerPhone         *string    `db:"passenger_phone" json:"passenger_phone,omitempty"`
	BoardingStopID         *string    `db:"boarding_stop_id" json:"boarding_stop_id,omitempty"`
	DropoffStopID          *string    `db:"dropoff_stop_id" json:"dropoff_stop_id,omitempty"`
	Price                  float64    `db:"price" json:"price"`
	QRToken                string     `db:"qr_token" json:"qr_token"`
	Status                 string     `db:"status" json:"status"`
	CheckedInAt            *time.Time `db:"checked_in_at" json:"checked_in_at,omitempty"`
	DownloadToken          *string    `db:"download_token" json:"-"`
	DownloadTokenExpiresAt *time.Time `db:"download_token_expires_at" json:"-"`
	CreatedAt              time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt              time.Time  `db:"updated_at" json:"updated_at"`
	Journey                *Journey   `db:"-" json:"journey,omitempty"`
}

// Request/Response types

type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type RegisterRequest struct {
	Email      string `json:"email" binding:"required"`
	Password   string `json:"password" binding:"required,min=6"`
	FullName   string `json:"full_name" binding:"required"`
	Phone      string `json:"phone"`
	Role       string `json:"role"`
	AgencyName string `json:"agency_name"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         User   `json:"user"`
}

type SearchRequest struct {
	Origin      string `json:"origin" form:"origin" binding:"required"`
	Destination string `json:"destination" form:"destination" binding:"required"`
	Date        string `json:"date" form:"date" binding:"required"`
}

type BookRequest struct {
	JourneyID  string     `json:"journey_id" binding:"required"`
	Seats      []BookSeat `json:"seats" binding:"required,min=1"`
	GuestEmail *string    `json:"guest_email"`
	GuestName  *string    `json:"guest_name"`
}

type BookSeat struct {
	SeatNumber     string `json:"seat_number" binding:"required"`
	PassengerName  string `json:"passenger_name" binding:"required"`
	PassengerPhone string `json:"passenger_phone"`
	BoardingStopID string `json:"boarding_stop_id"`
	DropoffStopID  string `json:"dropoff_stop_id"`
}

type CreateBusRequest struct {
	RegistrationNumber string `json:"registration_number" binding:"required"`
	Capacity           int    `json:"capacity" binding:"required,min=1"`
	SeatConfiguration  string `json:"seat_configuration"`
	ImageURL           string `json:"image_url"`
	DriverName         string `json:"driver_name"`
}

type UpdateBusRequest struct {
	RegistrationNumber string `json:"registration_number" binding:"required"`
	Capacity           int    `json:"capacity" binding:"required,min=1"`
	SeatConfiguration  string `json:"seat_configuration"`
	ImageURL           string `json:"image_url"`
	DriverName         string `json:"driver_name"`
}

type CreateRouteRequest struct {
	Name        string       `json:"name" binding:"required"`
	Origin      string       `json:"origin" binding:"required"`
	Destination string       `json:"destination" binding:"required"`
	BasePrice   float64      `json:"base_price" binding:"required,min=0"`
	Stops       []CreateStop `json:"stops"`
}

type CreateStop struct {
	Location               string  `json:"location" binding:"required"`
	StopOrder              int     `json:"stop_order" binding:"required"`
	ArrivalOffsetMinutes   *int    `json:"arrival_offset_minutes"`
	DepartureOffsetMinutes *int    `json:"departure_offset_minutes"`
	StopPrice              float64 `json:"stop_price"`
}

type CreateJourneyRequest struct {
	RouteID       string `json:"route_id" binding:"required"`
	BusID         string `json:"bus_id" binding:"required"`
	DepartureDate string `json:"departure_date" binding:"required"`
	DepartureTime string `json:"departure_time" binding:"required"`
	ArrivalDate   string `json:"arrival_date" binding:"required"`
	ArrivalTime   string `json:"arrival_time" binding:"required"`
}

type QRScanRequest struct {
	QRToken   string `json:"qr_token" binding:"required"`
	JourneyID string `json:"journey_id" binding:"required"`
}

// Helper for nullable string
func NullString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// NullStringToPtr handles sql.NullString conversion
func NullStringToPtr(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}