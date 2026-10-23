package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	// Database connection from environment or default
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:postgres@localhost:5432/bus_booking?sslmode=disable"
	}

	db, err := sqlx.Connect("postgres", databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	log.Println("Connected to database. Starting seed...")

	// Seed data
	if err := seedData(db); err != nil {
		log.Fatalf("Seed failed: %v", err)
	}

	log.Println("Seed completed successfully!")
}

func seedData(db *sqlx.DB) error {
	rand.Seed(time.Now().UnixNano())

	// Clear existing data (optional - comment out if you want to keep existing)
	// if err := clearData(db); err != nil {
	//     return err
	// }

	// 1. Create 20 agencies
	log.Println("Creating 20 agencies...")
	agencies, err := createAgencies(db, 20)
	if err != nil {
		return fmt.Errorf("failed to create agencies: %w", err)
	}
	log.Printf("Created %d agencies", len(agencies))

	// 2. Create routes for each agency
	log.Println("Creating routes for each agency...")
	routes, err := createRoutes(db, agencies)
	if err != nil {
		return fmt.Errorf("failed to create routes: %w", err)
	}
	log.Printf("Created %d routes", len(routes))

	// 3. Create buses for each agency (at least 5 per agency)
	log.Println("Creating buses for each agency...")
	buses, err := createBuses(db, agencies)
	if err != nil {
		return fmt.Errorf("failed to create buses: %w", err)
	}
	log.Printf("Created %d buses", len(buses))

	// 4. Create route stops
	log.Println("Creating route stops...")
	if err := createRouteStops(db, routes); err != nil {
		return fmt.Errorf("failed to create route stops: %w", err)
	}
	log.Println("Created route stops")

	// 5. Create journeys for each agency on the specified routes
	log.Println("Creating journeys...")
	journeys, err := createJourneys(db, agencies, routes, buses)
	if err != nil {
		return fmt.Errorf("failed to create journeys: %w", err)
	}
	log.Printf("Created %d journeys", len(journeys))

	return nil
}

type Agency struct {
	ID    string `db:"id"`
	Name  string `db:"name"`
	Email string `db:"email"`
}

func createAgencies(db *sqlx.DB, count int) ([]Agency, error) {
	agencies := make([]Agency, 0, count)

	for i := 1; i <= count; i++ {
		name := fmt.Sprintf("Agence Express %d", i)
		email := fmt.Sprintf("agence-express-%d@busbooking.com", i)
		phone := fmt.Sprintf("+24399%07d", 1000000+i)

		var agency Agency
		err := db.QueryRowx(`
			INSERT INTO organizations (name, email, phone, is_active)
			VALUES ($1, $2, $3, true)
			ON CONFLICT (email) DO UPDATE SET name = EXCLUDED.name, phone = EXCLUDED.phone, is_active = true
			RETURNING id, name, email
		`, name, email, phone).StructScan(&agency)

		if err != nil {
			if err == sql.ErrNoRows {
				// Agency already exists, fetch it
				err = db.Get(&agency, `SELECT id, name, email FROM organizations WHERE email = $1`, email)
				if err != nil {
					return nil, fmt.Errorf("failed to fetch existing agency %s: %w", email, err)
				}
			} else {
				return nil, fmt.Errorf("failed to create agency %s: %w", name, err)
			}
		}
		agencies = append(agencies, agency)
	}

	return agencies, nil
}

type Route struct {
	ID             string  `db:"id"`
	OrganizationID string  `db:"organization_id"`
	Name           string  `db:"name"`
	Origin         string  `db:"origin"`
	Destination    string  `db:"destination"`
	BasePrice      float64 `db:"base_price"`
}

func createRoutes(db *sqlx.DB, agencies []Agency) ([]Route, error) {
	routes := make([]Route, 0)

	// Define the three routes from Goma
	routeConfigs := []struct {
		name        string
		origin      string
		destination string
		basePrice   float64
	}{
		{"Goma - Beni", "Goma", "Beni", 30.0},
		{"Goma - Butembo", "Goma", "Butembo", 35.0},
		{"Goma - Uvira", "Goma", "Uvira", 40.0},
	}

	for _, agency := range agencies {
		for _, rc := range routeConfigs {
			var route Route
			err := db.QueryRowx(`
				INSERT INTO routes (organization_id, name, origin, destination, base_price, is_active)
				VALUES ($1, $2, $3, $4, $5, true)
				ON CONFLICT DO NOTHING
				RETURNING id, organization_id, name, origin, destination, base_price
			`, agency.ID, rc.name, rc.origin, rc.destination, rc.basePrice).StructScan(&route)

			if err != nil {
				if err == sql.ErrNoRows {
					// Route might already exist, fetch it
					err = db.Get(&route, `
						SELECT id, organization_id, name, origin, destination, base_price
						FROM routes
						WHERE organization_id = $1 AND origin = $2 AND destination = $3
					`, agency.ID, rc.origin, rc.destination)
					if err != nil {
						return nil, fmt.Errorf("failed to fetch existing route: %w", err)
					}
				} else {
					return nil, fmt.Errorf("failed to create route %s for agency %s: %w", rc.name, agency.Name, err)
				}
			}
			routes = append(routes, route)
		}
	}

	return routes, nil
}

type Bus struct {
	ID             string `db:"id"`
	OrganizationID string `db:"organization_id"`
	RegNumber      string `db:"registration_number"`
	Capacity       int    `db:"capacity"`
}

func createBuses(db *sqlx.DB, agencies []Agency) ([]Bus, error) {
	buses := make([]Bus, 0)

	// Generate unique registration numbers
	regCounter := 1
	for _, agency := range agencies {
		// Create 5-8 buses per agency
		busCount := 5 + rand.Intn(4) // 5 to 8 buses

		for i := 0; i < busCount; i++ {
			regNumber := fmt.Sprintf("EXP-%03d-%02d", regCounter/100, regCounter%100)
			regCounter++
			capacity := 30 + rand.Intn(21) // 30 to 50 seats

			var bus Bus
			err := db.QueryRowx(`
				INSERT INTO buses (organization_id, registration_number, capacity, is_active)
				VALUES ($1, $2, $3, true)
				ON CONFLICT (organization_id, registration_number) DO UPDATE SET capacity = EXCLUDED.capacity, is_active = true
				RETURNING id, organization_id, registration_number, capacity
			`, agency.ID, regNumber, capacity).StructScan(&bus)

			if err != nil {
				if err == sql.ErrNoRows {
					err = db.Get(&bus, `
						SELECT id, organization_id, registration_number, capacity
						FROM buses
						WHERE organization_id = $1 AND registration_number = $2
					`, agency.ID, regNumber)
					if err != nil {
						return nil, fmt.Errorf("failed to fetch existing bus: %w", err)
					}
				} else {
					return nil, fmt.Errorf("failed to create bus %s for agency %s: %w", regNumber, agency.Name, err)
				}
			}
			buses = append(buses, bus)
		}
	}

	return buses, nil
}

func createRouteStops(db *sqlx.DB, routes []Route) error {
	// Define intermediate stops for each route
	routeStops := map[string][]struct {
		stopOrder              int
		location               string
		arrivalOffsetMinutes   int
		departureOffsetMinutes int
		stopPrice              float64
	}{
		"Goma - Beni": {
			{1, "Rutshuru", 120, 135, 10.0},
			{2, "Kiwanja", 240, 255, 20.0},
			{3, "Bambo", 360, 375, 25.0},
		},
		"Goma - Butembo": {
			{1, "Rutshuru", 120, 135, 10.0},
			{2, "Kiwanja", 240, 255, 20.0},
			{3, "Beni", 360, 375, 25.0},
			{4, "Lubero", 480, 495, 30.0},
		},
		"Goma - Uvira": {
			{1, "Sake", 60, 75, 8.0},
			{2, "Minova", 180, 195, 15.0},
			{3, "Kalehe", 300, 315, 22.0},
			{4, "Bukavu", 420, 435, 30.0},
			{5, "Kamanyola", 540, 555, 35.0},
		},
	}

	for _, route := range routes {
		stops, exists := routeStops[route.Name]
		if !exists {
			continue
		}

		for _, stop := range stops {
			_, err := db.Exec(`
				INSERT INTO route_stops (route_id, stop_order, location, arrival_offset_minutes, departure_offset_minutes, stop_price)
				VALUES ($1, $2, $3, $4, $5, $6)
				ON CONFLICT (route_id, stop_order) DO UPDATE SET
					location = EXCLUDED.location,
					arrival_offset_minutes = EXCLUDED.arrival_offset_minutes,
					departure_offset_minutes = EXCLUDED.departure_offset_minutes,
					stop_price = EXCLUDED.stop_price
			`, route.ID, stop.stopOrder, stop.location, stop.arrivalOffsetMinutes, stop.departureOffsetMinutes, stop.stopPrice)

			if err != nil {
				return fmt.Errorf("failed to create stop %s for route %s: %w", stop.location, route.Name, err)
			}
		}
	}

	return nil
}

type Journey struct {
	ID             string    `db:"id"`
	OrganizationID string    `db:"organization_id"`
	RouteID        string    `db:"route_id"`
	BusID          string    `db:"bus_id"`
	DepartureDate  time.Time `db:"departure_date"`
	DepartureTime  string    `db:"departure_time"`
	ArrivalDate    time.Time `db:"arrival_date"`
	ArrivalTime    string    `db:"arrival_time"`
}

func createJourneys(db *sqlx.DB, agencies []Agency, routes []Route, buses []Bus) ([]Journey, error) {
	journeys := make([]Journey, 0)

	// Group routes by agency
	routesByAgency := make(map[string][]Route)
	for _, route := range routes {
		routesByAgency[route.OrganizationID] = append(routesByAgency[route.OrganizationID], route)
	}

	// Group buses by agency
	busesByAgency := make(map[string][]Bus)
	for _, bus := range buses {
		busesByAgency[bus.OrganizationID] = append(busesByAgency[bus.OrganizationID], bus)
	}

	// Create journeys for the next 7 days
	startDate := time.Now().Truncate(24 * time.Hour)
	for dayOffset := 0; dayOffset < 7; dayOffset++ {
		departureDate := startDate.AddDate(0, 0, dayOffset)

		for _, agency := range agencies {
			agencyRoutes := routesByAgency[agency.ID]
			agencyBuses := busesByAgency[agency.ID]

			if len(agencyRoutes) == 0 || len(agencyBuses) == 0 {
				continue
			}

			// For each route, create 2-3 journeys per day with different times
			for _, route := range agencyRoutes {
				// Pick a random bus from this agency
				bus := agencyBuses[rand.Intn(len(agencyBuses))]

				// Create 2-3 journeys per route per day
				journeyCount := 2 + rand.Intn(2) // 2 or 3
				for j := 0; j < journeyCount; j++ {
					// Departure times spread throughout the day
					departureHour := 6 + j*5 + rand.Intn(3) // 6-7, 11-12, 16-17 roughly
					departureTime := fmt.Sprintf("%02d:%02d:00", departureHour, rand.Intn(2)*30)

					// Calculate arrival time based on route
					var travelHours int
					switch route.Name {
					case "Goma - Beni":
						travelHours = 6 + rand.Intn(2) // 6-7 hours
					case "Goma - Butembo":
						travelHours = 8 + rand.Intn(2) // 8-9 hours
					case "Goma - Uvira":
						travelHours = 9 + rand.Intn(2) // 9-10 hours
					default:
						travelHours = 6
					}

					arrivalHour := departureHour + travelHours
					arrivalTime := fmt.Sprintf("%02d:%02d:00", arrivalHour, rand.Intn(2)*30)

					// Arrival date might be next day for long journeys
					arrivalDate := departureDate
					if arrivalHour >= 24 {
						arrivalHour -= 24
						arrivalDate = departureDate.AddDate(0, 0, 1)
						arrivalTime = fmt.Sprintf("%02d:%02d:00", arrivalHour, rand.Intn(2)*30)
					}

					var journey Journey
					err := db.QueryRowx(`
						INSERT INTO journeys (organization_id, route_id, bus_id, departure_date, departure_time, arrival_date, arrival_time, status)
						VALUES ($1, $2, $3, $4, $5, $6, $7, 'scheduled')
						RETURNING id, organization_id, route_id, bus_id, departure_date, departure_time, arrival_date, arrival_time
					`, agency.ID, route.ID, bus.ID, departureDate.Format("2006-01-02"), departureTime, arrivalDate.Format("2006-01-02"), arrivalTime).StructScan(&journey)

					if err != nil {
						return nil, fmt.Errorf("failed to create journey for agency %s route %s: %w", agency.Name, route.Name, err)
					}
					journeys = append(journeys, journey)
				}
			}
		}
	}

	return journeys, nil
}

func clearData(db *sqlx.DB) error {
	log.Println("Clearing existing data...")
	// Delete in order to respect foreign keys
	tables := []string{
		"tickets",
		"bookings",
		"journeys",
		"route_stops",
		"routes",
		"buses",
		"organizations",
	}

	for _, table := range tables {
		if _, err := db.Exec(fmt.Sprintf("DELETE FROM %s", table)); err != nil {
			return fmt.Errorf("failed to clear table %s: %w", table, err)
		}
		log.Printf("Cleared table: %s", table)
	}

	return nil
}