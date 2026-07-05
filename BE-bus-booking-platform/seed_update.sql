-- Seed Update: Delete old journeys & create new ones for the week of July 6-12, 2026
-- Organization: CTM Maroc (ce69d662-3794-4570-a54e-e8057e41ee19)

BEGIN;

-- ===========================
-- 1. Delete old tickets and bookings (they reference journeys)
-- ===========================
DELETE FROM tickets;
DELETE FROM bookings;

-- ===========================
-- 2. Delete old journeys (June 22-28, all past)
-- ===========================
DELETE FROM journeys;

-- ===========================
-- 3. Assign driver names to existing buses
-- ===========================
UPDATE buses SET driver_name = 'Ahmed El Mansouri' WHERE registration_number = 'CTM-1111-F';
UPDATE buses SET driver_name = 'Youssef Benali' WHERE registration_number = 'CTM-2222-G';
UPDATE buses SET driver_name = 'Mohammed Tazi' WHERE registration_number = 'CTM-3333-H';
UPDATE buses SET driver_name = 'Hassan Ouazzani' WHERE registration_number = 'CTM-4444-I';
UPDATE buses SET driver_name = 'Karim Idrissi' WHERE registration_number = 'CTM-5555-J';
UPDATE buses SET driver_name = 'Rachid El Fassi' WHERE driver_name IS NULL;

-- ===========================
-- 4. Delete the test route "Test City A - City B"
-- ===========================
DELETE FROM route_stops WHERE route_id = 'a4469271-7074-42b8-b693-0c10ba1a3bde';
DELETE FROM routes WHERE id = 'a4469271-7074-42b8-b693-0c10ba1a3bde';

-- ===========================
-- 5. Create 5-6 journeys per route per day for July 6-12, 2026
-- ===========================
DO $$
DECLARE
  route_rec RECORD;
  bus_id_val UUID;
  bus_ids UUID[] := ARRAY(
    SELECT id FROM buses ORDER BY registration_number
  );
  bus_idx INT := 1;
  start_date DATE := '2026-07-06';
  end_date DATE := '2026-07-12';
  curr_date DATE;
  dep_time TEXT;
  arr_time TEXT;
  dep_hour INT;
  dep_minute INT;
  -- 5-6 departure times per route
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
      -- Assign schedule based on route
      IF route_rec.id = '0b5f2d3a-4abf-4fb3-bb02-63fffe0edcba' THEN
        route_hours := ct_hours;   -- Casablanca→Tangier (5h)
        route_minutes := ct_minutes;
        travel_minutes := 300;
      ELSIF route_rec.id = '2f5c99d5-3810-4bf4-9b73-c5353a5418c0' THEN
        route_hours := mr_hours;   -- Marrakech→Rabat (4.5h)
        route_minutes := mr_minutes;
        travel_minutes := 270;
      ELSIF route_rec.id = '1dd130ae-ee66-4115-8c22-c7165648dea4' THEN
        route_hours := fc_hours;   -- Fes→Casablanca (5h)
        route_minutes := fc_minutes;
        travel_minutes := 300;
      ELSIF route_rec.id = 'eab1eb8a-8113-484b-b316-4bd6a3a25641' THEN
        route_hours := ac_hours;   -- Agadir→Casablanca (8h)
        route_minutes := ac_minutes;
        travel_minutes := 480;
      ELSE
        CONTINUE;
      END IF;

      -- Create 5 journeys for this route on this day
      FOR i IN 1..5 LOOP
        dep_hour := route_hours[i];
        dep_minute := route_minutes[i];

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
          dep_time_formatted::time without time zone,
          next_bus_date,
          arr_time_formatted::time without time zone,
          'scheduled'
        );
      END LOOP;
    END LOOP;
  END LOOP;
END $$;

COMMIT;
