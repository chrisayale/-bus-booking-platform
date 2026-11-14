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
	tomorrowStr := now.AddDate(0, 0, 1).Format("2006-01-02")
	log.Printf("Adding journeys for TOMORROW at 23:00: %s", tomorrowStr)

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

		// Create 1 journey at 23:00
		busID := busIDs[0] // Use first bus

		depTime := "23:00:00"
		// Travel time based on price (roughly 1 hour per $10, min 2 hours)
		travelHours := 3
		arrHour := (23 + travelHours) % 24 // 02:00 next day
		arrTime := fmt.Sprintf("%02d:00:00", arrHour)
		arrDate := now.AddDate(0, 0, 2).Format("2006-01-02") // Day after tomorrow

		// Status is scheduled since 23:00 is in the future
		status := "scheduled"

		_, err = tx.Exec(`
			INSERT INTO journeys (id, organization_id, route_id, bus_id, departure_date, departure_time, arrival_date, arrival_time, status, created_at, updated_at)
			VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
		`, orgID, routeID, busID, tomorrowStr, depTime, arrDate, arrTime, status)
		if err != nil {
			log.Fatalf("Failed to insert journey: %v", err)
		}
		journeyCount++
		log.Printf("  ✓ %s → %s | %s | %s | Bus: %s | LATE NIGHT", route.Origin, route.Dest, depTime, status, busID[:8])
	}

	if err = tx.Commit(); err != nil {
		log.Fatalf("Failed to commit: %v", err)
	}

	log.Printf("\n✅ Added %d journeys for TOMORROW (%s) at 23:00", journeyCount, tomorrowStr)
}