-- Bus Booking Platform — Automatic Seed Data
-- Runs once on first PostgreSQL container start (via docker-entrypoint-initdb.d)
-- Creates organization, routes, buses, test users, and 7 days of journeys from tomorrow

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
-- 2. Routes
-- ===========================
INSERT INTO routes (id, organization_id, name, origin, destination, base_price, is_active)
VALUES
  ('0b5f2d3a-4abf-4fb3-bb02-63fffe0edcba', 'ce69d662-3794-4570-a54e-e8057e41ee19', 'Casablanca-Tangier Express', 'Casablanca', 'Tangier', 300, true),
  ('2f5c99d5-3810-4bf4-9b73-c5353a5418c0', 'ce69d662-3794-4570-a54e-e8057e41ee19', 'Marrakech-Rabat', 'Marrakech', 'Rabat', 250, true),
  ('1dd130ae-ee66-4115-8c22-c7165648dea4', 'ce69d662-3794-4570-a54e-e8057e41ee19', 'Fes-Casablanca Direct', 'Fes', 'Casablanca', 280, true),
  ('eab1eb8a-8113-484b-b316-4bd6a3a25641', 'ce69d662-3794-4570-a54e-e8057e41ee19', 'Agadir-Casablanca', 'Agadir', 'Casablanca', 400, true)
ON CONFLICT (id) DO NOTHING;

-- ===========================
-- 3. Buses
-- ===========================
INSERT INTO buses (organization_id, registration_number, capacity, seat_configuration, is_active, driver_name)
VALUES
  ('ce69d662-3794-4570-a54e-e8057e41ee19', 'CTM-1111-F', 50, '{"rows":10,"layout":"2+2","columns":4}'::jsonb, true, 'Ahmed El Mansouri'),
  ('ce69d662-3794-4570-a54e-e8057e41ee19', 'CTM-2222-G', 40, '{"rows":10,"layout":"2+2","columns":4}'::jsonb, true, 'Youssef Benali'),
  ('ce69d662-3794-4570-a54e-e8057e41ee19', 'CTM-3333-H', 55, '{"rows":11,"layout":"2+2","columns":5}'::jsonb, true, 'Mohammed Tazi'),
  ('ce69d662-3794-4570-a54e-e8057e41ee19', 'CTM-4444-I', 45, '{"rows":9,"layout":"2+2","columns":5}'::jsonb, true, 'Hassan Ouazzani'),
  ('ce69d662-3794-4570-a54e-e8057e41ee19', 'CTM-5555-J', 50, '{"rows":10,"layout":"2+2","columns":5}'::jsonb, true, 'Karim Idrissi');

-- ===========================
-- 4. Test Users
-- ===========================
-- Owner admin — password: admin123
INSERT INTO users (email, password_hash, full_name, phone, role, organization_id, is_verified)
VALUES (
  'admin@busbook.com',
  '$2a$10$7jDOE/Znqd7zyIPMqDK97.3Vht7iTCV6wWE6x.AM/awkpJKwbypwm',
  'Admin CTM',
  '+212 6XX XXX XXX',
  'owner',
  'ce69d662-3794-4570-a54e-e8057e41ee19',
  true
)
ON CONFLICT (email) DO NOTHING;

-- Passenger — password: admin123
INSERT INTO users (email, password_hash, full_name, phone, role, is_verified)
VALUES (
  'passenger@test.com',
  '$2a$10$7jDOE/Znqd7zyIPMqDK97.3Vht7iTCV6wWE6x.AM/awkpJKwbypwm',
  'Test Passenger',
  '+212 6XX YYY YYY',
  'passenger',
  true
)
ON CONFLICT (email) DO NOTHING;

-- ===========================
-- 5. Journeys — 7 days starting tomorrow
-- ===========================
DO $$
DECLARE
  route_rec RECORD;
  bus_id_val UUID;
  bus_ids UUID[] := ARRAY(SELECT id FROM buses ORDER BY registration_number);
  bus_idx INT := 1;
  start_date DATE := CURRENT_DATE + 1;
  end_date DATE := CURRENT_DATE + 7;
  curr_date DATE;
  dep_hour INT;
  dep_minute INT;
  ct_hours INT[] := ARRAY[6, 9, 12, 15, 18];
  ct_minutes INT[] := ARRAY[0, 0, 0, 0, 0];
  mr_hours INT[] := ARRAY[6, 9, 12, 15, 18];
  mr_minutes INT[] := ARRAY[0, 30, 0, 0, 30];
  fc_hours INT[] := ARRAY[5, 8, 11, 14, 17];
  fc_minutes INT[] := ARRAY[0, 0, 0, 0, 0];
  ac_hours INT[] := ARRAY[6, 9, 12, 15, 18];
  ac_minutes INT[] := ARRAY[0, 0, 0, 0, 0];
  route_hours INT[];
  route_minutes INT[];
  travel_minutes INT;
  dep_total_min INT;
  arr_total_min INT;
  next_bus_date DATE;
  dep_time_formatted TEXT;
  arr_time_formatted TEXT;
BEGIN
  FOR curr_date IN SELECT generate_series(start_date, end_date, '1 day') LOOP
    FOR route_rec IN SELECT * FROM routes ORDER BY name LOOP
      IF route_rec.id = '0b5f2d3a-4abf-4fb3-bb02-63fffe0edcba' THEN
        route_hours := ct_hours; route_minutes := ct_minutes; travel_minutes := 300;
      ELSIF route_rec.id = '2f5c99d5-3810-4bf4-9b73-c5353a5418c0' THEN
        route_hours := mr_hours; route_minutes := mr_minutes; travel_minutes := 270;
      ELSIF route_rec.id = '1dd130ae-ee66-4115-8c22-c7165648dea4' THEN
        route_hours := fc_hours; route_minutes := fc_minutes; travel_minutes := 300;
      ELSIF route_rec.id = 'eab1eb8a-8113-484b-b316-4bd6a3a25641' THEN
        route_hours := ac_hours; route_minutes := ac_minutes; travel_minutes := 480;
      ELSE
        CONTINUE;
      END IF;

      FOR i IN 1..5 LOOP
        dep_hour := route_hours[i];
        dep_minute := route_minutes[i];

        -- Round-robin bus selection
        bus_id_val := bus_ids[bus_idx];
        bus_idx := (bus_idx % array_length(bus_ids, 1)) + 1;

        dep_total_min := dep_hour * 60 + dep_minute;
        arr_total_min := dep_total_min + travel_minutes;

        dep_time_formatted := lpad(dep_hour::text, 2, '0') || ':' || lpad(dep_minute::text, 2, '0') || ':00';

        IF arr_total_min >= 1440 THEN
          arr_total_min := arr_total_min - 1440;
          next_bus_date := curr_date + 1;
        ELSE
          next_bus_date := curr_date;
        END IF;

        arr_time_formatted := lpad((arr_total_min / 60)::text, 2, '0') || ':' || lpad((arr_total_min % 60)::text, 2, '0') || ':00';

        INSERT INTO journeys (id, organization_id, route_id, bus_id, departure_date, departure_time, arrival_date, arrival_time, status)
        VALUES (
          gen_random_uuid(),
          'ce69d662-3794-4570-a54e-e8057e41ee19',
          route_rec.id,
          bus_id_val,
          curr_date,
          dep_time_formatted::time,
          next_bus_date,
          arr_time_formatted::time,
          'scheduled'
        );
      END LOOP;
    END LOOP;
  END LOOP;
END $$;

COMMIT;
