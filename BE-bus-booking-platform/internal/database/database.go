package database

import (
	"embed"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

//go:embed seed/init.sql
var seedSQL string

func Connect(databaseURL string) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	return db, nil
}

func RunMigrations(db *sqlx.DB) error {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	for _, file := range files {
		content, err := migrationsFS.ReadFile("migrations/" + file)
		if err != nil {
			log.Printf("Warning: failed to read migration %s: %v", file, err)
			continue
		}

		log.Printf("Running migration: %s", file)
		if _, err := db.Exec(string(content)); err != nil {
			// Only skip "already exists" errors — fail on everything else
			if pgErr, ok := err.(*pq.Error); ok && (pgErr.Code == "42P07" || pgErr.Code == "42710" || pgErr.Code == "42701") {
				log.Printf("Skipping migration %s (already applied): %v", file, err)
				continue
			}
			return fmt.Errorf("migration %s failed: %w", file, err)
		}
	}

	return nil
}

func RunSeed(db *sqlx.DB) error {
	// Check if seed data already exists
	var orgCount int
	if err := db.Get(&orgCount, `SELECT COUNT(*) FROM organizations`); err != nil {
		// Tables might not exist yet — skip seed
		log.Printf("Seed: skipping (tables not ready): %v", err)
		return nil
	}
	if orgCount > 0 {
		log.Println("Seed: data already exists, skipping")
		return nil
	}

	log.Println("Seed: populating database with demo data...")
	if _, err := db.Exec(seedSQL); err != nil {
		return fmt.Errorf("seed failed: %w", err)
	}
	log.Println("Seed: done")
	return nil
}

// EnsureSuperAdmin ensures the super admin account exists on every startup.
// Uses UPSERT (ON CONFLICT) to be idempotent — safe to run on every boot.
// Uses "platform_admin" role which has full admin privileges in the application.
func EnsureSuperAdmin(db *sqlx.DB) error {
	const superAdminEmail = "superadmin@ayaletrans.com"
	const superAdminPasswordHash = "$2a$10$2xGTAcz5UTTDVbKjL1UDhOahYCvle7Y3.ghA5lGGTRUvAgeJd86Rm" // bcrypt cost 10 for "Chrisayale@2020"
	const superAdminFullName = "Super Admin"
	const superAdminPhone = "+212 6XX ZZZ ZZZ"
	const superAdminRole = "platform_admin"

	query := `
		INSERT INTO users (email, password_hash, full_name, phone, role, is_verified)
		VALUES ($1, $2, $3, $4, $5, true)
		ON CONFLICT (email) DO UPDATE SET
			password_hash = EXCLUDED.password_hash,
			full_name = EXCLUDED.full_name,
			phone = EXCLUDED.phone,
			role = EXCLUDED.role,
			is_verified = EXCLUDED.is_verified,
			updated_at = NOW()
	`

	log.Println("Ensuring super admin account exists...")
	if _, err := db.Exec(query, superAdminEmail, superAdminPasswordHash, superAdminFullName, superAdminPhone, superAdminRole); err != nil {
		return fmt.Errorf("failed to ensure super admin: %w", err)
	}
	log.Println("Super admin account ensured")
	return nil
}
