package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

type AnalyticsHandler struct {
	DB *sqlx.DB
}

func NewAnalyticsHandler(db *sqlx.DB) *AnalyticsHandler {
	return &AnalyticsHandler{DB: db}
}

type DashboardStats struct {
	TotalBuses     int              `json:"total_buses"`
	ActiveBuses    int              `json:"active_buses"`
	TotalRoutes    int              `json:"total_routes"`
	TotalJourneys  int              `json:"total_journeys"`
	UpcomingJourneys int           `json:"upcoming_journeys"`
	TotalBookings  int              `json:"total_bookings"`
	TotalRevenue   float64          `json:"total_revenue"`
	ActiveTickets  int              `json:"active_tickets"`
	CheckedIn      int              `json:"checked_in"`
	MissedTickets  int              `json:"missed_tickets"`
	CanceledTickets int            `json:"canceled_tickets"`
	SeatOccupancy  float64          `json:"seat_occupancy"` // percentage
	MonthlyRevenue []MonthlyRevenue `json:"monthly_revenue"`
	PopularRoutes  []PopularRoute   `json:"popular_routes"`
	RecentJourneys []JourneySummary `json:"recent_journeys"`
}

type MonthlyRevenue struct {
	Month  string  `json:"month"`
	Year   int     `json:"year"`
	Amount float64 `json:"amount"`
}

type PopularRoute struct {
	RouteName    string `db:"route_name" json:"route_name"`
	Origin       string `db:"origin" json:"origin"`
	Destination  string `db:"destination" json:"destination"`
	TicketCount  int    `db:"ticket_count" json:"ticket_count"`
}

type JourneySummary struct {
	ID            string `db:"id" json:"id"`
	DepartureDate string `db:"departure_date" json:"departure_date"`
	DepartureTime string `db:"departure_time" json:"departure_time"`
	Status        string `db:"status" json:"status"`
	BookedSeats   int    `db:"booked_seats" json:"booked_seats"`
	TotalSeats    int    `db:"total_seats" json:"total_seats"`
}

func (h *AnalyticsHandler) GetDashboardStats(c *gin.Context) {
	orgID := c.GetString("org_id")
	stats := DashboardStats{}

	// Total & active buses
	h.DB.Get(&stats.TotalBuses, `SELECT COUNT(*) FROM buses WHERE organization_id=$1`, orgID)
	h.DB.Get(&stats.ActiveBuses, `SELECT COUNT(*) FROM buses WHERE organization_id=$1 AND is_active=true`, orgID)

	// Routes
	h.DB.Get(&stats.TotalRoutes, `SELECT COUNT(*) FROM routes WHERE organization_id=$1`, orgID)

	// Journeys
	h.DB.Get(&stats.TotalJourneys, `SELECT COUNT(*) FROM journeys WHERE organization_id=$1`, orgID)
	h.DB.Get(&stats.UpcomingJourneys, `SELECT COUNT(*) FROM journeys WHERE organization_id=$1 AND status IN ('scheduled', 'boarding') AND departure_date >= CURRENT_DATE`, orgID)

	// Bookings & revenue
	h.DB.Get(&stats.TotalBookings, `
		SELECT COUNT(*) FROM bookings b
		JOIN tickets t ON t.booking_id = b.id
		JOIN journeys j ON t.journey_id = j.id
		WHERE j.organization_id=$1`, orgID)
	h.DB.Get(&stats.TotalRevenue, `
		SELECT COALESCE(SUM(t.price), 0) FROM tickets t
		JOIN journeys j ON t.journey_id = j.id
		WHERE j.organization_id=$1 AND t.status IN ('active', 'checked_in')`, orgID)

	// Ticket status breakdown
	h.DB.Get(&stats.ActiveTickets, `
		SELECT COUNT(*) FROM tickets t
		JOIN journeys j ON t.journey_id = j.id
		WHERE j.organization_id=$1 AND t.status='active'`, orgID)
	h.DB.Get(&stats.CheckedIn, `
		SELECT COUNT(*) FROM tickets t
		JOIN journeys j ON t.journey_id = j.id
		WHERE j.organization_id=$1 AND t.status='checked_in'`, orgID)
	h.DB.Get(&stats.MissedTickets, `
		SELECT COUNT(*) FROM tickets t
		JOIN journeys j ON t.journey_id = j.id
		WHERE j.organization_id=$1 AND t.status='missed'`, orgID)
	h.DB.Get(&stats.CanceledTickets, `
		SELECT COUNT(*) FROM tickets t
		JOIN journeys j ON t.journey_id = j.id
		WHERE j.organization_id=$1 AND t.status='cancelled'`, orgID)

	// Seat occupancy (percentage of booked seats across all journeys)
	var totalCapacity, totalBooked int
	h.DB.Get(&totalCapacity, `
		SELECT COALESCE(SUM(b.capacity), 0) FROM journeys j
		JOIN buses b ON j.bus_id = b.id
		WHERE j.organization_id=$1 AND j.status NOT IN ('cancelled')`, orgID)
	h.DB.Get(&totalBooked, `
		SELECT COUNT(*) FROM tickets t
		JOIN journeys j ON t.journey_id = j.id
		WHERE j.organization_id=$1 AND t.status IN ('active', 'checked_in')`, orgID)
	if totalCapacity > 0 {
		stats.SeatOccupancy = float64(totalBooked) / float64(totalCapacity) * 100
	}

	// Monthly revenue (last 6 months)
	h.DB.Select(&stats.MonthlyRevenue, `
		SELECT
			TO_CHAR(t.created_at, 'Mon') as month,
			EXTRACT(YEAR FROM t.created_at)::int as year,
			COALESCE(SUM(t.price), 0) as amount
		FROM tickets t
		JOIN journeys j ON t.journey_id = j.id
		WHERE j.organization_id=$1
		  AND t.created_at >= CURRENT_DATE - INTERVAL '6 months'
		  AND t.status IN ('active', 'checked_in')
		GROUP BY year, EXTRACT(MONTH FROM t.created_at), month
		ORDER BY year, EXTRACT(MONTH FROM t.created_at)`, orgID)

	// Popular routes
	h.DB.Select(&stats.PopularRoutes, `
		SELECT
			r.name as route_name,
			r.origin,
			r.destination,
			COUNT(t.id) as ticket_count
		FROM tickets t
		JOIN journeys j ON t.journey_id = j.id
		JOIN routes r ON j.route_id = r.id
		WHERE j.organization_id=$1 AND t.status IN ('active', 'checked_in')
		GROUP BY r.id, r.name, r.origin, r.destination
		ORDER BY ticket_count DESC
		LIMIT 5`, orgID)

	// Recent journeys
	h.DB.Select(&stats.RecentJourneys, `
		SELECT
			j.id,
			j.departure_date,
			j.departure_time,
			j.status,
			(SELECT COUNT(*) FROM tickets WHERE journey_id=j.id AND status IN ('active', 'checked_in')) as booked_seats,
			(SELECT b.capacity FROM buses b WHERE b.id=j.bus_id) as total_seats
		FROM journeys j
		WHERE j.organization_id=$1
		ORDER BY j.departure_date DESC, j.departure_time DESC
		LIMIT 6`, orgID)

	if stats.RecentJourneys == nil {
		stats.RecentJourneys = []JourneySummary{}
	}
	if stats.MonthlyRevenue == nil {
		stats.MonthlyRevenue = []MonthlyRevenue{}
	}
	if stats.PopularRoutes == nil {
		stats.PopularRoutes = []PopularRoute{}
	}

	c.JSON(http.StatusOK, stats)
}
