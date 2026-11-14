// +build ignore

package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

const (
	DBHost     = "localhost"
	DBPort     = 5433
	DBUser     = "postgres"
	DBPassword = "postgres"
	DBName     = "bus_booking"
)

var db *sql.DB

// Main routes per agency (index matches agency order)
var mainRoutes = []struct {
	AgencyIdx int
	RouteName string
	Origin    string
	Dest      string
	BasePrice float64
}{
	{0, "Goma - Bukavu Express", "Goma", "Bukavu", 25.0},       // Kivu Travel
	{1, "Kinshasa - Matadi Express", "Kinshasa", "Matadi", 15.0}, // Trans Kinshasa
	{2, "Lubumbashi - Likasi", "Lubumbashi", "Likasi", 12.0},     // Katanga Express
	{3, "Goma - Rutshuru", "Goma", "Rutshuru", 15.0},             // Goma Voyages
	{4, "Matadi - Kinshasa Express", "Matadi", "Kinshasa", 15.0}, // Matadi Lines
	{5, "Kinshasa - Kananga", "Kinshasa", "Kananga", 40.0},       // Congo Connect
	{6, "Mbandaka - Gemena", "Mbandaka", "Gemena", 20.0},         // Equateur Transit
	{7, "Kananga - Tshikapa", "Kananga", "Tshikapa", 18.0},       // Kasaï Express
	{8, "Kisangani - Isiro", "Kisangani", "Isiro", 35.0},         // Orientale Bus
	{9, "Beni - Butembo", "Beni", "Butembo", 12.0},               // Grand Nord Transport
}

// 3 schedules per route: Morning, Midday, Evening
var todaySchedules = []struct {
	DepHour   int
	DepMin    int
	Label     string
	TravelHrs int
}{
	{6, 30, "MORNING", 3},
	{12, 0, "MIDDAY", 3},
	{18, 30, "EVENING", 3},
}

func main() {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		DBHost, DBPort, DBUser, DBPassword, DBName)

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatalf("Failed to ping: %v", err)
	}
	log.Println("Connected to database")

	now := time.Now()
	todayStr := now.Format("2006-01-02")
	log.Printf("Adding journeys for TODAY: %s", todayStr)

	tx, err := db.Begin()
	if err != nil {
		log.Fatalf("Failed to start transaction: %v", err)
	}

	journeyCount := 0

	for _, route := range mainRoutes {
		// Get organization ID
		var orgID string
		err = tx.QueryRow(`SELECT id FROM organizations WHERE reference_code = $1`,
			fmt.Sprintf("1000%02d", route.AgencyIdx+1)).Scan(&orgID)
		if err != nil {
			log.Fatalf("Failed to get org for agency %d: %v", route.AgencyIdx, err)
		}

		// Get route ID
		var routeID string
		err = tx.QueryRow(`SELECT id FROM routes WHERE organization_id = $1 AND name = $2`,
			orgID, route.RouteName).Scan(&routeID)
		if err != nil {
			log.Fatalf("Failed to get route %s: %v", route.RouteName, err)
		}

		// Get bus IDs for this agency
		busRows, err := tx.Query(`SELECT id FROM buses WHERE organization_id = $1 ORDER BY created_at`, orgID)
		if err != nil {
			log.Fatalf("Failed to get buses: %v", err)
		}
		var busIDs []string
		for busRows.Next() {
			var bid string
			busRows.Scan(&bid)
			busIDs = append(busIDs, bid)
		}
		busRows.Close()

		if len(busIDs) == 0 {
			log.Printf("No buses for agency %d, skipping", route.AgencyIdx)
			continue
		}

		// Create 3 journeys (morning, midday, evening) on 2 different buses
		for i, sched := range todaySchedules {
			busID := busIDs[i%len(busIDs)]

			depTime := fmt.Sprintf("%02d:%02d:00", sched.DepHour, sched.DepMin)
			arrHour := (sched.DepHour + sched.TravelHrs) % 24
			arrTime := fmt.Sprintf("%02d:%02d:00", arrHour, sched.DepMin)
			arrDate := todayStr

			var status string
			nowHour, nowMin := now.Hour(), now.Minute()
			if sched.DepHour < nowHour || (sched.DepHour == nowHour && sched.DepMin <= nowMin) {
				if sched.DepHour < nowHour || (sched.DepHour == nowHour && sched.DepMin+30 <= nowMin) {
					status = "departed"
				} else {
					status = "boarding"
				}
			} else {
				status = "scheduled"
			}

			_, err = tx.Exec(`
				INSERT INTO journeys (id, organization_id, route_id, bus_id, departure_date, departure_time, arrival_date, arrival_time, status, created_at, updated_at)
				VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
			`, orgID, routeID, busID, todayStr, depTime, arrDate, arrTime, status)
			if err != nil {
				log.Fatalf("Failed to insert journey: %v", err)
			}
			journeyCount++
			log.Printf("  ✓ %s → %s | %s | %s | Bus: %s | %s", route.Origin, route.Dest, depTime, status, busID[:8], sched.Label)
		}
	}

	if err = tx.Commit(); err != nil {
		log.Fatalf("Failed to commit: %v", err)
	}

	log.Printf("\n✅ Added %d journeys for TODAY (%s)", journeyCount, todayStr)
	log.Println("Schedules: 6:30 (Morning), 12:00 (Midday), 18:30 (Evening) per agency")
}
