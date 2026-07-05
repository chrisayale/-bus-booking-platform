-- Full setup: Organization + Routes + Buses + Users + Journeys
BEGIN;

-- ===========================
-- 1. Organization: CTM Maroc
-- ===========================
INSERT INTO organizations (id, name, email, phone)
VALUES (
  'ce69d662-3794-4570-a54e-e8057e41ee19',
  'CTM Maroc',
  'contact@ctm.ma',
  '+212 5XX XX XX XX'
)
ON CONFLICT (id) DO NOTHING;

-- ===========================
-- 2. Routes (with fixed IDs)
-- ===========================
INSERT INTO routes (id, organization_id, name, origin, destination, base_price, is_active)
VALUES
  ('0b5f2d3a-4abf-4fb3-bb02-63fffe0edcba', 'ce69d662-3794-4570-a54e-e8057e41ee19', 'Casa-Tangier Express', 'Casablanca', 'Tangier', 300, true),
  ('2f5c99d5-3810-4bf4-9b73-c5353a5418c0', 'ce69d662-3794-4570-a54e-e8057e41ee19', 'Marrakech-Rabat', 'Marrakech', 'Rabat', 250, true),
  ('1dd130ae-ee66-4115-8c22-c7165648dea4', 'ce69d662-3794-4570-a54e-e8057e41ee19', 'Fes-Casa Direct', 'Fes', 'Casablanca', 280, true),
  ('eab1eb8a-8113-484b-b316-4bd6a3a25641', 'ce69d662-3794-4570-a54e-e8057e41ee19', 'Agadir-Casa', 'Agadir', 'Casablanca', 400, true)
ON CONFLICT (id) DO NOTHING;

-- ===========================
-- 3. Buses (5 buses)
-- ===========================
INSERT INTO buses (id, organization_id, registration_number, capacity, seat_configuration, is_active, driver_name)
VALUES
  (gen_random_uuid(), 'ce69d662-3794-4570-a54e-e8057e41ee19', 'CTM-1111-F', 50, '{"rows":10,"layout":"2+2","columns":4}'::jsonb, true, 'Ahmed El Mansouri'),
  (gen_random_uuid(), 'ce69d662-3794-4570-a54e-e8057e41ee19', 'CTM-2222-G', 40, '{"rows":10,"layout":"2+2","columns":4}'::jsonb, true, 'Youssef Benali'),
  (gen_random_uuid(), 'ce69d662-3794-4570-a54e-e8057e41ee19', 'CTM-3333-H', 55, '{"rows":11,"layout":"2+2","columns":5}'::jsonb, true, 'Mohammed Tazi'),
  (gen_random_uuid(), 'ce69d662-3794-4570-a54e-e8057e41ee19', 'CTM-4444-I', 45, '{"rows":9,"layout":"2+2","columns":5}'::jsonb, true, 'Hassan Ouazzani'),
  (gen_random_uuid(), 'ce69d662-3794-4570-a54e-e8057e41ee19', 'CTM-5555-J', 50, '{"rows":10,"layout":"2+2","columns":5}'::jsonb, true, 'Karim Idrissi');

-- ===========================
-- 4. Test Users
-- ===========================
-- Owner: admin@busbook.com / admin123
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
)
ON CONFLICT (email) DO NOTHING;

-- Passenger: passenger@test.com / admin123
INSERT INTO users (id, email, password_hash, full_name, phone, role, is_verified)
VALUES (
  gen_random_uuid(),
  'passenger@test.com',
  '$2a$10$7jDOE/Znqd7zyIPMqDK97.3Vht7iTCV6wWE6x.AM/awkpJKwbypwm',
  'Test Passenger',
  '+212 6XX YYY YYY',
  'passenger',
  true
)
ON CONFLICT (email) DO NOTHING;

COMMIT;
