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

// CleanupTestData removes test agencies and associated accounts to allow fresh agency registrations.
// Keeps only the Super Admin account (superadmin@ayaletrans.com).
func CleanupTestData(db *sqlx.DB) error {
	log.Println("Cleaning up test agencies and associated accounts...")

	// First, find organizations to delete (exclude any that might be needed, but we clean all test orgs)
	// We'll delete all organizations except we keep none - super admin has no org
	// Delete associated data in correct order due to foreign keys

	// 1. Delete tickets for journeys of these orgs
	// 2. Delete bookings for users of these orgs
	// 3. Delete journeys, buses, routes, route_stops for these orgs
	// 4. Delete user_organizations entries
	// 5. Delete users with role 'owner', 'agent', 'controller' (keep super_admin, platform_admin, passenger)
	// 6. Delete organizations

	// Get organization IDs to delete (all of them for clean slate)
	var orgIDs []string
	err := db.Select(&orgIDs, `SELECT id FROM organizations`)
	if err != nil {
		return fmt.Errorf("failed to fetch organizations: %w", err)
	}

	if len(orgIDs) > 0 {
		// Build placeholders for IN clause
		placeholders := make([]string, len(orgIDs))
		args := make([]interface{}, len(orgIDs))
		for i, id := range orgIDs {
			placeholders[i] = fmt.Sprintf("$%d", i+1)
			args[i] = id
		}
		inClause := strings.Join(placeholders, ", ")

		// Delete in transaction for consistency
		tx, err := db.Beginx()
		if err != nil {
			return fmt.Errorf("failed to start transaction: %w", err)
		}
		defer tx.Rollback()

		// Delete tickets for journeys of these orgs
		_, err = tx.Exec(fmt.Sprintf(`
			DELETE FROM tickets WHERE journey_id IN (
				SELECT id FROM journeys WHERE organization_id IN (%s)
			)`, inClause), args...)
		if err != nil {
			return fmt.Errorf("failed to delete tickets: %w", err)
		}

		// Delete bookings for users of these orgs (owners)
		_, err = tx.Exec(fmt.Sprintf(`
			DELETE FROM bookings WHERE user_id IN (
				SELECT id FROM users WHERE organization_id IN (%s)
			)`, inClause), args...)
		if err != nil {
			return fmt.Errorf("failed to delete bookings: %w", err)
		}

		// Delete journeys
		_, err = tx.Exec(fmt.Sprintf(`DELETE FROM journeys WHERE organization_id IN (%s)`, inClause), args...)
		if err != nil {
			return fmt.Errorf("failed to delete journeys: %w", err)
		}

		// Delete buses
		_, err = tx.Exec(fmt.Sprintf(`DELETE FROM buses WHERE organization_id IN (%s)`, inClause), args...)
		if err != nil {
			return fmt.Errorf("failed to delete buses: %w", err)
		}

		// Delete route_stops for routes of these orgs
		_, err = tx.Exec(fmt.Sprintf(`
			DELETE FROM route_stops WHERE route_id IN (
				SELECT id FROM routes WHERE organization_id IN (%s)
			)`, inClause), args...)
		if err != nil {
			return fmt.Errorf("failed to delete route_stops: %w", err)
		}

		// Delete routes
		_, err = tx.Exec(fmt.Sprintf(`DELETE FROM routes WHERE organization_id IN (%s)`, inClause), args...)
		if err != nil {
			return fmt.Errorf("failed to delete routes: %w", err)
		}

		// Delete user_organizations entries for these orgs
		_, err = tx.Exec(fmt.Sprintf(`DELETE FROM user_organizations WHERE organization_id IN (%s)`, inClause), args...)
		if err != nil {
			return fmt.Errorf("failed to delete user_organizations: %w", err)
		}

		// Delete users with role in ('owner', 'agent', 'controller') for these orgs
		_, err = tx.Exec(fmt.Sprintf(`
			DELETE FROM users WHERE organization_id IN (%s) AND role IN ('owner', 'agent', 'controller')
		`, inClause), args...)
		if err != nil {
			return fmt.Errorf("failed to delete users: %w", err)
		}

		// Delete organizations
		_, err = tx.Exec(fmt.Sprintf(`DELETE FROM organizations WHERE id IN (%s)`, inClause), args...)
		if err != nil {
			return fmt.Errorf("failed to delete organizations: %w", err)
		}

		if err = tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit cleanup transaction: %w", err)
		}

		log.Printf("Cleaned up %d test organization(s) and associated data", len(orgIDs))
	} else {
		log.Println("No test organizations found to clean up")
	}

	// Also clean up any pending_agencies
	_, err = db.Exec(`DELETE FROM pending_agencies`)
	if err != nil {
		log.Printf("Warning: failed to clean pending_agencies: %v", err)
	} else {
		log.Println("Cleaned up pending_agencies table")
	}

	return nil
}

// EnsureSuperAdmin ensures the super admin account exists on every startup.
// Uses UPSERT (ON CONFLICT) to be idempotent — safe to run on every boot.
// Uses "super_admin" role which has full admin privileges in the application.
func EnsureSuperAdmin(db *sqlx.DB) error {
	const superAdminEmail = "superadmin@ayaletrans.com"
	const superAdminPasswordHash = "$2a$10$2xGTAcz5UTTDVbKjL1UDhOahYCvle7Y3.ghA5lGGTRUvAgeJd86Rm" // bcrypt cost 10 for "Chrisayale@2020"
	const superAdminFullName = "Super Admin"
	const superAdminPhone = "+212 6XX ZZZ ZZZ"
	const superAdminRole = "super_admin"

	query := `
		INSERT INTO users (email, password_hash, full_name, phone, role, is_verified, is_active)
		VALUES ($1, $2, $3, $4, $5, true, true)
		ON CONFLICT (email) DO UPDATE SET
			password_hash = EXCLUDED.password_hash,
			full_name = EXCLUDED.full_name,
			phone = EXCLUDED.phone,
			role = EXCLUDED.role,
			is_verified = EXCLUDED.is_verified,
			is_active = EXCLUDED.is_active,
			updated_at = NOW()
	`

	log.Println("Ensuring super admin account exists...")
	if _, err := db.Exec(query, superAdminEmail, superAdminPasswordHash, superAdminFullName, superAdminPhone, superAdminRole); err != nil {
		return fmt.Errorf("failed to ensure super admin: %w", err)
	}
	log.Println("Super admin account ensured")
	return nil
}
