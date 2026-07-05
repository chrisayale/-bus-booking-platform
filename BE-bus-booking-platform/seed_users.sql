-- Seed users for testing
-- Run this AFTER the migrations and seed_full.sql have been applied.
--
-- Default passwords (change these in production):
--   Owner:     admin@busbook.com / admin123
--   Passenger: passenger@test.com / admin123
--
-- The owner is linked to the existing "CTM Maroc" organization
-- (ce69d662-3794-4570-a54e-e8057e41ee19) so they can see all the seed
-- buses, routes, and journeys immediately.

BEGIN;

-- Owner account (linked to CTM Maroc)
INSERT INTO users (id, email, password_hash, full_name, phone, role, organization_id, is_verified)
VALUES (
  gen_random_uuid(),
  'admin@busbook.com',
  '$2a$10$7jDOE/Znqd7zyIPMqDK97.3Vht7iTCV6wWE6x.AM/awkpJKwbypwm',
  'Admin CTM',
  '+212 6XX XXX XXX',
  'owner',
  'ce69d662-3794-4570-a54e-e8057e41ee19',
  true
);

-- Passenger account (no organization)
INSERT INTO users (id, email, password_hash, full_name, phone, role, is_verified)
VALUES (
  gen_random_uuid(),
  'passenger@test.com',
  '$2a$10$7jDOE/Znqd7zyIPMqDK97.3Vht7iTCV6wWE6x.AM/awkpJKwbypwm',
  'Test Passenger',
  '+212 6XX YYY YYY',
  'passenger',
  true
);

COMMIT;
