// +build ignore

package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"math"
	"math/big"
	"time"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

// Database configuration
const (
	DBHost     = "localhost"
	DBPort     = 5433
	DBUser     = "postgres"
	DBPassword = "postgres"
	DBName     = "bus_booking"
)

var db *sql.DB

// DRC Agencies data
type Agency struct {
	Name           string
	Email          string
	Phone          string
	ReferenceCode  string
	ManagerName    string
	Address        string
	Password       string
}

// DRC Routes data
type Route struct {
	Name        string
	Origin      string
	Destination string
	BasePrice   float64
	Stops       []Stop
}

type Stop struct {
	Location string
	Price    float64
}

// Agency configurations
var agencies = []Agency{
	{
		Name:           "Kivu Travel",
		Email:          "contact@kivutravel.cd",
		Phone:          "+243 99 12 34 567",
		ReferenceCode:  "100001",
		ManagerName:    "Jean-Pierre Mutombo",
		Address:        "Boulevard Kanyamuhanga, Goma",
		Password:       "kivu2024!",
	},
	{
		Name:           "Trans Kinshasa",
		Email:          "info@transkinshasa.cd",
		Phone:          "+243 81 23 45 678",
		ReferenceCode:  "100002",
		ManagerName:    "Marie-Claire Ilunga",
		Address:        "Avenue de la Justice, Kinshasa",
		Password:       "kinshasa2024!",
	},
	{
		Name:           "Katanga Express",
		Email:          "booking@katangaexpress.cd",
		Phone:          "+243 97 34 56 789",
		ReferenceCode:  "100003",
		ManagerName:    "Antoine Kasongo",
		Address:        "Avenue M'Siri, Lubumbashi",
		Password:       "katanga2024!",
	},
	{
		Name:           "Goma Voyages",
		Email:          "reservation@gomavoyages.cd",
		Phone:          "+243 99 45 67 890",
		ReferenceCode:  "100004",
		ManagerName:    "Grace Nsimire",
		Address:        "Rue du Lac, Goma",
		Password:       "goma2024!",
	},
	{
		Name:           "Matadi Lines",
		Email:          "contact@matadilines.cd",
		Phone:          "+243 82 56 78 901",
		ReferenceCode:  "100005",
		ManagerName:    "Paul Mbemba",
		Address:        "Avenue du Port, Matadi",
		Password:       "matadi2024!",
	},
	{
		Name:           "Congo Connect",
		Email:          "info@congoconnect.cd",
		Phone:          "+243 98 67 89 012",
		ReferenceCode:  "100006",
		ManagerName:    "Chantal Makengo",
		Address:        "Boulevard du 30 Juin, Kinshasa",
		Password:       "congo2024!",
	},
	{
		Name:           "Equateur Transit",
		Email:          "booking@equateurtransit.cd",
		Phone:          "+243 84 78 90 123",
		ReferenceCode:  "100007",
		ManagerName:    "Robert Lokonda",
		Address:        "Avenue des Martyrs, Mbandaka",
		Password:       "equateur2024!",
	},
	{
		Name:           "Kasaï Express",
		Email:          "contact@kasaiexpress.cd",
		Phone:          "+243 99 89 01 234",
		ReferenceCode:  "100008",
		ManagerName:    "Julie Tshisekedi",
		Address:        "Rue de l'Indépendance, Kananga",
		Password:       "kasai2024!",
	},
	{
		Name:           "Orientale Bus",
		Email:          "info@orientalebus.cd",
		Phone:          "+243 85 90 12 345",
		ReferenceCode:  "100009",
		ManagerName:    "David Asani",
		Address:        "Avenue de la Paix, Kisangani",
		Password:       "orientale2024!",
	},
	{
		Name:           "Grand Nord Transport",
		Email:          "reservation@grandnord.cd",
		Phone:          "+243 97 01 23 456",
		ReferenceCode:  "100010",
		ManagerName:    "Sophie Kahindo",
		Address:        "Rue du Marché, Beni",
		Password:       "nord2024!",
	},
}

// Routes per agency (10 routes each)
var agencyRoutes = map[int][]Route{
	0: { // Kivu Travel - Nord Kivu / Sud Kivu
		{Name: "Goma - Bukavu Express", Origin: "Goma", Destination: "Bukavu", BasePrice: 25.0, Stops: []Stop{{Location: "Kalehe", Price: 12.0}, {Location: "Minova", Price: 18.0}}},
		{Name: "Goma - Beni", Origin: "Goma", Destination: "Beni", BasePrice: 35.0, Stops: []Stop{{Location: "Rutshuru", Price: 15.0}, {Location: "Kiwanja", Price: 25.0}}},
		{Name: "Goma - Butembo", Origin: "Goma", Destination: "Butembo", BasePrice: 30.0, Stops: []Stop{{Location: "Rutshuru", Price: 12.0}, {Location: "Bambu", Price: 22.0}}},
		{Name: "Bukavu - Uvira", Origin: "Bukavu", Destination: "Uvira", BasePrice: 20.0, Stops: []Stop{{Location: "Kamanyola", Price: 10.0}}},
		{Name: "Goma - Bunia", Origin: "Goma", Destination: "Bunia", BasePrice: 55.0, Stops: []Stop{{Location: "Rutshuru", Price: 15.0}, {Location: "Beni", Price: 35.0}, {Location: "Komo", Price: 45.0}}},
		{Name: "Bukavu - Kindu", Origin: "Bukavu", Destination: "Kindu", BasePrice: 45.0, Stops: []Stop{{Location: "Shabunda", Price: 22.0}}},
		{Name: "Goma - Kisangani", Origin: "Goma", Destination: "Kisangani", BasePrice: 85.0, Stops: []Stop{{Location: "Beni", Price: 35.0}, {Location: "Butembo", Price: 50.0}}},
		{Name: "Bukavu - Walikale", Origin: "Bukavu", Destination: "Walikale", BasePrice: 30.0, Stops: []Stop{{Location: "Muzimu", Price: 15.0}}},
		{Name: "Goma - Sake", Origin: "Goma", Destination: "Sake", BasePrice: 10.0, Stops: []Stop{{Location: "Mugunga", Price: 5.0}}},
		{Name: "Bukavu - Shabunda", Origin: "Bukavu", Destination: "Shabunda", BasePrice: 25.0, Stops: []Stop{{Location: "Mwana", Price: 12.0}}},
	},
	1: { // Trans Kinshasa - Kinshasa / Bas-Congo
		{Name: "Kinshasa - Matadi Express", Origin: "Kinshasa", Destination: "Matadi", BasePrice: 15.0, Stops: []Stop{{Location: "Kisantu", Price: 8.0}, {Location: "Mbanza-Ngungu", Price: 12.0}}},
		{Name: "Kinshasa - Kikwit", Origin: "Kinshasa", Destination: "Kikwit", BasePrice: 30.0, Stops: []Stop{{Location: "Kenge", Price: 15.0}, {Location: "Idiofa", Price: 22.0}}},
		{Name: "Kinshasa - Bandundu", Origin: "Kinshasa", Destination: "Bandundu", BasePrice: 25.0, Stops: []Stop{{Location: "Kikwit", Price: 12.0}, {Location: "Mongala", Price: 18.0}}},
		{Name: "Kinshasa - Mbanza-Ngungu", Origin: "Kinshasa", Destination: "Mbanza-Ngungu", BasePrice: 12.0, Stops: []Stop{{Location: "Kisantu", Price: 6.0}}},
		{Name: "Matadi - Boma", Origin: "Matadi", Destination: "Boma", BasePrice: 10.0, Stops: []Stop{{Location: "Muanda", Price: 5.0}}},
		{Name: "Kinshasa - Tshikapa", Origin: "Kinshasa", Destination: "Tshikapa", BasePrice: 45.0, Stops: []Stop{{Location: "Kikwit", Price: 15.0}, {Location: "Mweka", Price: 30.0}}},
		{Name: "Kinshasa - Mbuji-Mayi", Origin: "Kinshasa", Destination: "Mbuji-Mayi", BasePrice: 55.0, Stops: []Stop{{Location: "Kananga", Price: 30.0}}},
		{Name: "Kinshasa - Idiofa", Origin: "Kinshasa", Destination: "Idiofa", BasePrice: 28.0, Stops: []Stop{{Location: "Kenge", Price: 14.0}}},
		{Name: "Matadi - Moanda", Origin: "Matadi", Destination: "Moanda", BasePrice: 8.0, Stops: []Stop{}},
		{Name: "Kinshasa - Kisantu", Origin: "Kinshasa", Destination: "Kisantu", BasePrice: 8.0, Stops: []Stop{}},
	},
	2: { // Katanga Express - Haut-Katanga / Lualaba
		{Name: "Lubumbashi - Likasi", Origin: "Lubumbashi", Destination: "Likasi", BasePrice: 12.0, Stops: []Stop{{Location: "Kipushi", Price: 6.0}}},
		{Name: "Lubumbashi - Kolwezi", Origin: "Lubumbashi", Destination: "Kolwezi", BasePrice: 25.0, Stops: []Stop{{Location: "Kambove", Price: 12.0}, {Location: "Mutshatsha", Price: 18.0}}},
		{Name: "Lubumbashi - Kasumbalesa", Origin: "Lubumbashi", Destination: "Kasumbalesa", BasePrice: 8.0, Stops: []Stop{}},
		{Name: "Likasi - Kalemie", Origin: "Likasi", Destination: "Kalemie", BasePrice: 35.0, Stops: []Stop{{Location: "Kamina", Price: 18.0}}},
		{Name: "Kolwezi - Mutshatsha", Origin: "Kolwezi", Destination: "Mutshatsha", BasePrice: 15.0, Stops: []Stop{{Location: "Kambove", Price: 8.0}}},
		{Name: "Lubumbashi - Kamina", Origin: "Lubumbashi", Destination: "Kamina", BasePrice: 30.0, Stops: []Stop{{Location: "Kambove", Price: 12.0}, {Location: "Kaniama", Price: 22.0}}},
		{Name: "Likasi - Sakania", Origin: "Likasi", Destination: "Sakania", BasePrice: 10.0, Stops: []Stop{}},
		{Name: "Kolwezi - Dilolo", Origin: "Kolwezi", Destination: "Dilolo", BasePrice: 20.0, Stops: []Stop{{Location: "Mutshatsha", Price: 10.0}}},
		{Name: "Lubumbashi - Mokambo", Origin: "Lubumbashi", Destination: "Mokambo", BasePrice: 15.0, Stops: []Stop{{Location: "Kipushi", Price: 8.0}}},
		{Name: "Likasi - Kolwezi", Origin: "Likasi", Destination: "Kolwezi", BasePrice: 20.0, Stops: []Stop{{Location: "Kambove", Price: 10.0}}},
	},
	3: { // Goma Voyages - Nord Kivu
		{Name: "Goma - Rutshuru", Origin: "Goma", Destination: "Rutshuru", BasePrice: 15.0, Stops: []Stop{}},
		{Name: "Goma - Kiwanja", Origin: "Goma", Destination: "Kiwanja", BasePrice: 25.0, Stops: []Stop{{Location: "Rutshuru", Price: 15.0}}},
		{Name: "Goma - Kitchanga", Origin: "Goma", Destination: "Kitchanga", BasePrice: 20.0, Stops: []Stop{{Location: "Sake", Price: 10.0}}},
		{Name: "Goma - Walikale", Origin: "Goma", Destination: "Walikale", BasePrice: 30.0, Stops: []Stop{{Location: "Sake", Price: 10.0}, {Location: "Muzimu", Price: 20.0}}},
		{Name: "Rutshuru - Bunagana", Origin: "Rutshuru", Destination: "Bunagana", BasePrice: 8.0, Stops: []Stop{}},
		{Name: "Goma - Pinga", Origin: "Goma", Destination: "Pinga", BasePrice: 35.0, Stops: []Stop{{Location: "Kitchanga", Price: 20.0}}},
		{Name: "Sake - Minova", Origin: "Sake", Destination: "Minova", BasePrice: 10.0, Stops: []Stop{}},
		{Name: "Goma - Masisi", Origin: "Goma", Destination: "Masisi", BasePrice: 18.0, Stops: []Stop{{Location: "Sake", Price: 10.0}}},
		{Name: "Rutshuru - Ishasha", Origin: "Rutshuru", Destination: "Ishasha", BasePrice: 12.0, Stops: []Stop{}},
		{Name: "Goma - Birambizo", Origin: "Goma", Destination: "Birambizo", BasePrice: 22.0, Stops: []Stop{{Location: "Kitchanga", Price: 15.0}}},
	},
	4: { // Matadi Lines - Bas-Congo
		{Name: "Matadi - Kinshasa Express", Origin: "Matadi", Destination: "Kinshasa", BasePrice: 15.0, Stops: []Stop{{Location: "Mbanza-Ngungu", Price: 12.0}, {Location: "Kisantu", Price: 8.0}}},
		{Name: "Matadi - Boma", Origin: "Matadi", Destination: "Boma", BasePrice: 10.0, Stops: []Stop{{Location: "Muanda", Price: 5.0}}},
		{Name: "Matadi - Tshela", Origin: "Matadi", Destination: "Tshela", BasePrice: 12.0, Stops: []Stop{{Location: "Lukula", Price: 6.0}}},
		{Name: "Matadi - Lukula", Origin: "Matadi", Destination: "Lukula", BasePrice: 8.0, Stops: []Stop{}},
		{Name: "Boma - Muanda", Origin: "Boma", Destination: "Muanda", BasePrice: 8.0, Stops: []Stop{}},
		{Name: "Matadi - Kimpese", Origin: "Matadi", Destination: "Kimpese", BasePrice: 10.0, Stops: []Stop{}},
		{Name: "Matadi - Songololo", Origin: "Matadi", Destination: "Songololo", BasePrice: 12.0, Stops: []Stop{{Location: "Kimpese", Price: 8.0}}},
		{Name: "Boma - Seke-Banza", Origin: "Boma", Destination: "Seke-Banza", BasePrice: 10.0, Stops: []Stop{}},
		{Name: "Matadi - Mbanza-Ngungu", Origin: "Matadi", Destination: "Mbanza-Ngungu", BasePrice: 10.0, Stops: []Stop{}},
		{Name: "Matadi - Moanda", Origin: "Matadi", Destination: "Moanda", BasePrice: 8.0, Stops: []Stop{}},
	},
	5: { // Congo Connect - Kinshasa inter-provincial
		{Name: "Kinshasa - Kananga", Origin: "Kinshasa", Destination: "Kananga", BasePrice: 40.0, Stops: []Stop{{Location: "Kikwit", Price: 15.0}, {Location: "Tshikapa", Price: 28.0}}},
		{Name: "Kinshasa - Mbandaka", Origin: "Kinshasa", Destination: "Mbandaka", BasePrice: 35.0, Stops: []Stop{{Location: "Mongala", Price: 18.0}}},
		{Name: "Kinshasa - Lisala", Origin: "Kinshasa", Destination: "Lisala", BasePrice: 45.0, Stops: []Stop{{Location: "Bumba", Price: 25.0}}},
		{Name: "Kinshasa - Gemena", Origin: "Kinshasa", Destination: "Gemena", BasePrice: 50.0, Stops: []Stop{{Location: "Mbandaka", Price: 35.0}}},
		{Name: "Kinshasa - Bumba", Origin: "Kinshasa", Destination: "Bumba", BasePrice: 40.0, Stops: []Stop{{Location: "Lisala", Price: 22.0}}},
		{Name: "Kinshasa - Isiro", Origin: "Kinshasa", Destination: "Isiro", BasePrice: 80.0, Stops: []Stop{{Location: "Kisangani", Price: 50.0}}},
		{Name: "Kinshasa - Watsa", Origin: "Kinshasa", Destination: "Watsa", BasePrice: 75.0, Stops: []Stop{{Location: "Isiro", Price: 50.0}}},
		{Name: "Kinshasa - Aru", Origin: "Kinshasa", Destination: "Aru", BasePrice: 85.0, Stops: []Stop{{Location: "Bunia", Price: 55.0}}},
		{Name: "Kinshasa - Bunia", Origin: "Kinshasa", Destination: "Bunia", BasePrice: 70.0, Stops: []Stop{{Location: "Kisangani", Price: 45.0}}},
		{Name: "Kinshasa - Kindu", Origin: "Kinshasa", Destination: "Kindu", BasePrice: 50.0, Stops: []Stop{{Location: "Kisangani", Price: 30.0}}},
	},
	6: { // Equateur Transit - Equateur province
		{Name: "Mbandaka - Gemena", Origin: "Mbandaka", Destination: "Gemena", BasePrice: 20.0, Stops: []Stop{{Location: "Bikoro", Price: 10.0}}},
		{Name: "Mbandaka - Lisala", Origin: "Mbandaka", Destination: "Lisala", BasePrice: 25.0, Stops: []Stop{{Location: "Bumba", Price: 15.0}}},
		{Name: "Mbandaka - Bumba", Origin: "Mbandaka", Destination: "Bumba", BasePrice: 30.0, Stops: []Stop{{Location: "Bokungu", Price: 15.0}}},
		{Name: "Mbandaka - Bokungu", Origin: "Mbandaka", Destination: "Bokungu", BasePrice: 15.0, Stops: []Stop{}},
		{Name: "Gemena - Zongo", Origin: "Gemena", Destination: "Zongo", BasePrice: 15.0, Stops: []Stop{}},
		{Name: "Mbandaka - Basankusu", Origin: "Mbandaka", Destination: "Basankusu", BasePrice: 35.0, Stops: []Stop{{Location: "Bikoro", Price: 12.0}}},
		{Name: "Lisala - Bumba", Origin: "Lisala", Destination: "Bumba", BasePrice: 15.0, Stops: []Stop{}},
		{Name: "Mbandaka - Ikela", Origin: "Mbandaka", Destination: "Ikela", BasePrice: 25.0, Stops: []Stop{{Location: "Bokungu", Price: 15.0}}},
		{Name: "Gemena - Libenge", Origin: "Gemena", Destination: "Libenge", BasePrice: 12.0, Stops: []Stop{}},
		{Name: "Mbandaka - Bolomba", Origin: "Mbandaka", Destination: "Bolomba", BasePrice: 20.0, Stops: []Stop{{Location: "Bikoro", Price: 10.0}}},
	},
	7: { // Kasaï Express - Kasaï provinces
		{Name: "Kananga - Tshikapa", Origin: "Kananga", Destination: "Tshikapa", BasePrice: 18.0, Stops: []Stop{{Location: "Mweka", Price: 10.0}}},
		{Name: "Kananga - Mbuji-Mayi", Origin: "Kananga", Destination: "Mbuji-Mayi", BasePrice: 25.0, Stops: []Stop{{Location: "Luiza", Price: 15.0}}},
		{Name: "Kananga - Luebo", Origin: "Kananga", Destination: "Luebo", BasePrice: 15.0, Stops: []Stop{}},
		{Name: "Tshikapa - Mweka", Origin: "Tshikapa", Destination: "Mweka", BasePrice: 10.0, Stops: []Stop{}},
		{Name: "Mbuji-Mayi - Miabi", Origin: "Mbuji-Mayi", Destination: "Miabi", BasePrice: 8.0, Stops: []Stop{}},
		{Name: "Kananga - Dibaya", Origin: "Kananga", Destination: "Dibaya", BasePrice: 12.0, Stops: []Stop{}},
		{Name: "Tshikapa - Kamonia", Origin: "Tshikapa", Destination: "Kamonia", BasePrice: 15.0, Stops: []Stop{}},
		{Name: "Mbuji-Mayi - Kabinda", Origin: "Mbuji-Mayi", Destination: "Kabinda", BasePrice: 20.0, Stops: []Stop{{Location: "Ngandajika", Price: 12.0}}},
		{Name: "Kananga - Luiza", Origin: "Kananga", Destination: "Luiza", BasePrice: 15.0, Stops: []Stop{}},
		{Name: "Mbuji-Mayi - Lodja", Origin: "Mbuji-Mayi", Destination: "Lodja", BasePrice: 30.0, Stops: []Stop{{Location: "Ngandajika", Price: 15.0}}},
	},
	8: { // Orientale Bus - Haut-Uele / Bas-Uele / Tshopo
		{Name: "Kisangani - Isiro", Origin: "Kisangani", Destination: "Isiro", BasePrice: 35.0, Stops: []Stop{{Location: "Niangara", Price: 20.0}}},
		{Name: "Kisangani - Bunia", Origin: "Kisangani", Destination: "Bunia", BasePrice: 40.0, Stops: []Stop{{Location: "Nia-Nia", Price: 22.0}}},
		{Name: "Kisangani - Buta", Origin: "Kisangani", Destination: "Buta", BasePrice: 30.0, Stops: []Stop{{Location: "Banalia", Price: 15.0}}},
		{Name: "Isiro - Watsa", Origin: "Isiro", Destination: "Watsa", BasePrice: 20.0, Stops: []Stop{{Location: "Wamba", Price: 10.0}}},
		{Name: "Bunia - Aru", Origin: "Bunia", Destination: "Aru", BasePrice: 15.0, Stops: []Stop{}},
		{Name: "Kisangani - Bafwasende", Origin: "Kisangani", Destination: "Bafwasende", BasePrice: 25.0, Stops: []Stop{{Location: "Ubundu", Price: 12.0}}},
		{Name: "Isiro - Dungu", Origin: "Isiro", Destination: "Dungu", BasePrice: 25.0, Stops: []Stop{{Location: "Niangara", Price: 12.0}}},
		{Name: "Bunia - Mahagi", Origin: "Bunia", Destination: "Mahagi", BasePrice: 18.0, Stops: []Stop{}},
		{Name: "Kisangani - Opala", Origin: "Kisangani", Destination: "Opala", BasePrice: 28.0, Stops: []Stop{{Location: "Ubundu", Price: 14.0}}},
		{Name: "Buta - Bondo", Origin: "Buta", Destination: "Bondo", BasePrice: 22.0, Stops: []Stop{{Location: "Bima", Price: 12.0}}},
	},
	9: { // Grand Nord Transport - Nord Kivu / Ituri
		{Name: "Beni - Butembo", Origin: "Beni", Destination: "Butembo", BasePrice: 12.0, Stops: []Stop{}},
		{Name: "Beni - Bunia", Origin: "Beni", Destination: "Bunia", BasePrice: 25.0, Stops: []Stop{{Location: "Komo", Price: 15.0}}},
		{Name: "Beni - Eringeti", Origin: "Beni", Destination: "Eringeti", BasePrice: 10.0, Stops: []Stop{}},
		{Name: "Butembo - Lubero", Origin: "Butembo", Destination: "Lubero", BasePrice: 10.0, Stops: []Stop{}},
		{Name: "Beni - Oicha", Origin: "Beni", Destination: "Oicha", BasePrice: 8.0, Stops: []Stop{}},
		{Name: "Beni - Kasindi", Origin: "Beni", Destination: "Kasindi", BasePrice: 15.0, Stops: []Stop{{Location: "Eringeti", Price: 10.0}}},
		{Name: "Butembo - Kirumba", Origin: "Butembo", Destination: "Kirumba", BasePrice: 12.0, Stops: []Stop{{Location: "Lubero", Price: 8.0}}},
		{Name: "Beni - Mutwanga", Origin: "Beni", Destination: "Mutwanga", BasePrice: 12.0, Stops: []Stop{{Location: "Lubero", Price: 8.0}}},
		{Name: "Butembo - Kanyabayonga", Origin: "Butembo", Destination: "Kanyabayonga", BasePrice: 18.0, Stops: []Stop{{Location: "Lubero", Price: 10.0}}},
		{Name: "Beni - Mambasa", Origin: "Beni", Destination: "Mambasa", BasePrice: 30.0, Stops: []Stop{{Location: "Komo", Price: 15.0}, {Location: "Nia-Nia", Price: 22.0}}},
	},
}

func main() {
	// Connect to database
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		DBHost, DBPort, DBUser, DBPassword, DBName)

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("Connected to database successfully")

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		log.Fatalf("Failed to start transaction: %v", err)
	}

	// Seed agencies, users, buses, routes, stops, journeys
	agencyIDs := make([]string, len(agencies))
	busIDs := make([][]string, len(agencies))
	routeIDs := make([][]string, len(agencies))

	for i, agency := range agencies {
		log.Printf("Processing agency %d/%d: %s", i+1, len(agencies), agency.Name)

		// Hash password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(agency.Password), bcrypt.DefaultCost)
		if err != nil {
			log.Fatalf("Failed to hash password: %v", err)
		}

		// Generate UUID for organization
		orgID := generateUUID()

		// Insert organization
		_, err = tx.Exec(`
			INSERT INTO organizations (id, name, email, phone, is_active, reference_code, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
			ON CONFLICT (reference_code) DO UPDATE SET
				name = EXCLUDED.name,
				email = EXCLUDED.email,
				phone = EXCLUDED.phone,
				is_active = EXCLUDED.is_active,
				updated_at = NOW()
		`, orgID, agency.Name, agency.Email, agency.Phone, true, agency.ReferenceCode)
		if err != nil {
			log.Fatalf("Failed to insert organization %s: %v", agency.Name, err)
		}
		agencyIDs[i] = orgID
		log.Printf("  Created organization: %s (ID: %s, Code: %s)", agency.Name, orgID, agency.ReferenceCode)

		// Insert owner user
		userID := generateUUID()
		_, err = tx.Exec(`
			INSERT INTO users (id, email, password_hash, full_name, phone, role, organization_id, is_verified, approval_status, is_active, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
			ON CONFLICT (email) DO UPDATE SET
				password_hash = EXCLUDED.password_hash,
				full_name = EXCLUDED.full_name,
				phone = EXCLUDED.phone,
				role = EXCLUDED.role,
				organization_id = EXCLUDED.organization_id,
				is_verified = EXCLUDED.is_verified,
				approval_status = EXCLUDED.approval_status,
				is_active = EXCLUDED.is_active,
				updated_at = NOW()
		`, userID, agency.Email, string(hashedPassword), agency.ManagerName, agency.Phone, "owner", orgID, true, "approved", true)
		if err != nil {
			log.Fatalf("Failed to insert owner user for %s: %v", agency.Name, err)
		}
		log.Printf("  Created owner user: %s", agency.ManagerName)

		// Create 5 buses per agency
		busIDs[i] = make([]string, 5)
		busRegs := []string{
			fmt.Sprintf("%s-%04d-A", agency.ReferenceCode[:3], 1000+i*5+1),
			fmt.Sprintf("%s-%04d-B", agency.ReferenceCode[:3], 1000+i*5+2),
			fmt.Sprintf("%s-%04d-C", agency.ReferenceCode[:3], 1000+i*5+3),
			fmt.Sprintf("%s-%04d-D", agency.ReferenceCode[:3], 1000+i*5+4),
			fmt.Sprintf("%s-%04d-E", agency.ReferenceCode[:3], 1000+i*5+5),
		}
		capacities := []int{50, 40, 55, 45, 50}
		driverNames := []string{
			"Chauffeur Principal", "Chauffeur Secondaire", "Chauffeur Expérimenté",
			"Chauffeur Senior", "Chauffeur Junior",
		}

		for j := 0; j < 5; j++ {
			busID := generateUUID()
			busIDs[i][j] = busID
			seatConfig := fmt.Sprintf(`{"rows":%d,"columns":4,"layout":"2+2"}`, capacities[j]/4)

			_, err = tx.Exec(`
				INSERT INTO buses (id, organization_id, registration_number, capacity, seat_configuration, is_active, driver_name, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7, NOW(), NOW())
				ON CONFLICT (organization_id, registration_number) DO UPDATE SET
					capacity = EXCLUDED.capacity,
					seat_configuration = EXCLUDED.seat_configuration,
					is_active = EXCLUDED.is_active,
					driver_name = EXCLUDED.driver_name,
					updated_at = NOW()
			`, busID, orgID, busRegs[j], capacities[j], seatConfig, true, driverNames[j])
			if err != nil {
				log.Fatalf("Failed to insert bus for %s: %v", agency.Name, err)
			}
		}
		log.Printf("  Created 5 buses")

		// Create routes and stops
		routes := agencyRoutes[i]
		routeIDs[i] = make([]string, len(routes))

		for k, route := range routes {
			routeID := generateUUID()
			routeIDs[i][k] = routeID

			_, err = tx.Exec(`
				INSERT INTO routes (id, organization_id, name, origin, destination, base_price, is_active, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
				ON CONFLICT DO NOTHING
			`, routeID, orgID, route.Name, route.Origin, route.Destination, route.BasePrice, true)
			if err != nil {
				log.Fatalf("Failed to insert route %s: %v", route.Name, err)
			}

			// Insert stops
			for stopIdx, stop := range route.Stops {
				stopID := generateUUID()
				_, err = tx.Exec(`
					INSERT INTO route_stops (id, route_id, stop_order, location, arrival_offset_minutes, departure_offset_minutes, stop_price, created_at)
					VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
					ON CONFLICT DO NOTHING
				`, stopID, routeID, stopIdx+1, stop.Location, (stopIdx+1)*60, (stopIdx+1)*60+15, stop.Price)
				if err != nil {
					log.Fatalf("Failed to insert stop for route %s: %v", route.Name, err)
				}
			}
		}
		log.Printf("  Created %d routes with stops", len(routes))
	}

	// Create journeys for next 30 days
	log.Println("Creating journeys for next 30 days...")
	journeyCount := 0
	startDate := time.Now().AddDate(0, 0, 1) // Start from tomorrow

	for day := 0; day < 30; day++ {
		currentDate := startDate.AddDate(0, 0, day)
		dateStr := currentDate.Format("2006-01-02")

		for i := range agencies {
			routes := agencyRoutes[i]
			busList := busIDs[i]
			routeList := routeIDs[i]

			for k := range routes {
				// Create 2-3 journeys per route per day
				numJourneys := 2 + (day + k + i) % 2

				for j := 0; j < numJourneys; j++ {
					journeyID := generateUUID()
					busID := busList[(k+j)%len(busList)]
					routeID := routeList[k]
					orgID := agencyIDs[i]

					// Calculate departure and arrival times
					baseHour := 6 + (k*2 + j*3) % 12
					baseMinute := (k*15 + j*20) % 60
					depTime := fmt.Sprintf("%02d:%02d:00", baseHour, baseMinute)

					// Travel time based on price (roughly 1 hour per $10)
					travelHours := int(math.Max(1, routes[k].BasePrice/10))
					arrHour := (baseHour + travelHours) % 24
					arrMinute := baseMinute
					if arrHour < baseHour {
						// Next day
					}
					arrTime := fmt.Sprintf("%02d:%02d:00", arrHour, arrMinute)
					arrDate := dateStr
					if arrHour < baseHour || (arrHour == baseHour && arrMinute < baseMinute) {
						arrDate = currentDate.AddDate(0, 0, 1).Format("2006-01-02")
					}

					_, err = tx.Exec(`
						INSERT INTO journeys (id, organization_id, route_id, bus_id, departure_date, departure_time, arrival_date, arrival_time, status, created_at, updated_at)
						VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'scheduled', NOW(), NOW())
						ON CONFLICT DO NOTHING
					`, journeyID, orgID, routeID, busID, dateStr, depTime, arrDate, arrTime)
					if err != nil {
						log.Fatalf("Failed to insert journey: %v", err)
					}
					journeyCount++
				}
			}
		}
	}

	log.Printf("Created %d journeys over 30 days", journeyCount)

	// Commit transaction
	if err = tx.Commit(); err != nil {
		log.Fatalf("Failed to commit transaction: %v", err)
	}

	log.Println("✅ DRC mock data seeded successfully!")
	log.Println("")
	log.Println("Summary:")
	log.Printf("  - 10 Agencies with unique 6-digit reference codes")
	log.Printf("  - 10 Owner users (one per agency)")
	log.Printf("  - 50 Buses (5 per agency)")
	log.Printf("  - 100 Routes (10 per agency) with stops")
	log.Printf("  - %d Journeys (over 30 days)", journeyCount)
	log.Println("")
	log.Println("Test Accounts:")
	for _, agency := range agencies {
		log.Printf("  Agency: %s | Email: %s | Password: %s | Code: %s",
			agency.Name, agency.Email, agency.Password, agency.ReferenceCode)
	}
}

func generateUUID() string {
	// Generate a UUID v4
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	// Set version (4) and variant bits
	b[6] = (b[6] & 0x0f) | 0x40 // Version 4
	b[8] = (b[8] & 0x3f) | 0x80 // Variant 10
	return hex.EncodeToString(b[:4]) + "-" + hex.EncodeToString(b[4:6]) + "-" +
		hex.EncodeToString(b[6:8]) + "-" + hex.EncodeToString(b[8:10]) + "-" +
		hex.EncodeToString(b[10:])
}

func randomInt(max int) int {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(max)))
	return int(n.Int64())
}