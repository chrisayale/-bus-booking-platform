-- Seed script: Add more buses and 4-6 daily journeys per route for a week
-- Organization: CTM Maroc (ce69d662-3794-4570-a54e-e8057e41ee19)

BEGIN;

-- ===========================
-- 1. Add 5 more buses
-- ===========================
INSERT INTO buses (id, organization_id, registration_number, capacity, seat_configuration, is_active, driver_name)
VALUES
  (gen_random_uuid(), 'ce69d662-3794-4570-a54e-e8057e41ee19', 'CTM-1111-F', 50, '{"rows":10,"layout":"2+2","columns":4}'::jsonb, true, 'Ahmed El Mansouri'),
  (gen_random_uuid(), 'ce69d662-3794-4570-a54e-e8057e41ee19', 'CTM-2222-G', 40, '{"rows":10,"layout":"2+2","columns":4}'::jsonb, true, 'Youssef Benali'),
  (gen_random_uuid(), 'ce69d662-3794-4570-a54e-e8057e41ee19', 'CTM-3333-H', 55, '{"rows":11,"layout":"2+2","columns":5}'::jsonb, true, 'Mohammed Tazi'),
  (gen_random_uuid(), 'ce69d662-3794-4570-a54e-e8057e41ee19', 'CTM-4444-I', 45, '{"rows":9,"layout":"2+2","columns":5}'::jsonb, true, 'Hassan Ouazzani'),
  (gen_random_uuid(), 'ce69d662-3794-4570-a54e-e8057e41ee19', 'CTM-5555-J', 50, '{"rows":10,"layout":"2+2","columns":5}'::jsonb, true, 'Karim Idrissi');

-- ===========================
-- 2. Delete old journeys
-- ===========================
DELETE FROM journeys;

-- ===========================
-- 3. Create 4-6 journeys per route per day
--    Route IDs:
--      Casablanca→Tangier: 0b5f2d3a-4abf-4fb3-bb02-63fffe0edcba (5h travel)
--      Marrakech→Rabat:    2f5c99d5-3810-4bf4-9b73-c5353a5418c0 (4.5h travel)
--      Fes→Casablanca:     1dd130ae-ee66-4115-8c22-c7165648dea4 (5h travel)
--      Agadir→Casablanca:  eab1eb8a-8113-484b-b316-4bd6a3a25641 (8h travel, sometimes overnight)
-- ===========================

DO $$
DECLARE
  route_rec RECORD;
  bus_id_val UUID;
  bus_ids UUID[] := ARRAY(SELECT id FROM buses ORDER BY registration_number);
  bus_idx INT;
  start_date DATE := '2026-06-08';
  end_date DATE := '2026-06-14';
  curr_date DATE;
  dep_time TEXT;
  arr_time TEXT;
  dep_hour INT;
  -- Route-specific schedules (departure hours)
  -- Casablanca→Tangier: 06, 08, 10, 13, 15, 18
  ct_hours INT[] := ARRAY[6, 8, 10, 13, 15, 18];
  -- Marrakech→Rabat: 06, 08:30→8h, 11, 13:30→13h, 16, 18:30→18h
  mr_hours INT[] := ARRAY[6, 8, 11, 13, 16, 18];
  -- Fes→Casablanca: 05, 07:30→7h, 10, 13, 15:30→15h, 18
  fc_hours INT[] := ARRAY[5, 7, 10, 13, 15, 18];
  -- Agadir→Casablanca: 06, 09, 14, 22 (overnight)
  ac_hours INT[] := ARRAY[6, 9, 14, 22];
  route_hours INT[];
  travel_minutes INT;
  dep_minutes INT;
  arr_minutes INT;
  next_bus_date DATE;
BEGIN
  bus_idx := 1;
  
  FOR curr_date IN SELECT generate_series(start_date, end_date, '1 day') LOOP
    -- Process each route
    FOR route_rec IN SELECT * FROM routes LOOP
      -- Assign schedule based on route
      IF route_rec.id = '0b5f2d3a-4abf-4fb3-bb02-63fffe0edcba' THEN
        route_hours := ct_hours;
        travel_minutes := 300; -- 5h
      ELSIF route_rec.id = '2f5c99d5-3810-4bf4-9b73-c5353a5418c0' THEN
        route_hours := mr_hours;
        travel_minutes := 270; -- 4.5h
      ELSIF route_rec.id = '1dd130ae-ee66-4115-8c22-c7165648dea4' THEN
        route_hours := fc_hours;
        travel_minutes := 300; -- 5h
      ELSIF route_rec.id = 'eab1eb8a-8113-484b-b316-4bd6a3a25641' THEN
        route_hours := ac_hours;
        travel_minutes := 480; -- 8h
      ELSE
        CONTINUE;
      END IF;

      -- Create journeys for each departure time
      FOREACH dep_hour IN ARRAY route_hours LOOP
        -- Pick bus (round-robin)
        bus_id_val := bus_ids[bus_idx];
        bus_idx := (bus_idx % array_length(bus_ids, 1)) + 1;
        
        dep_minutes := dep_hour * 60;
        arr_minutes := dep_minutes + travel_minutes;
        
        dep_time := lpad(dep_hour::text, 2, '0') || ':00:00';
        
        -- Calculate arrival - may be next day
        IF arr_minutes >= 1440 THEN
          arr_minutes := arr_minutes - 1440;
          next_bus_date := curr_date + 1;
        ELSE
          next_bus_date := curr_date;
        END IF;
        
        arr_time := lpad((arr_minutes / 60)::text, 2, '0') || ':' || lpad((arr_minutes % 60)::text, 2, '0') || ':00';
        
        INSERT INTO journeys (id, organization_id, route_id, bus_id, departure_date, departure_time, arrival_date, arrival_time, status)
        VALUES (
          gen_random_uuid(),
          'ce69d662-3794-4570-a54e-e8057e41ee19',
          route_rec.id,
          bus_id_val,
          curr_date,
          dep_time::time without time zone,
          next_bus_date,
          arr_time::time without time zone,
          'scheduled'
        );
      END LOOP;
    END LOOP;
  END LOOP;
END $$;

COMMIT;
