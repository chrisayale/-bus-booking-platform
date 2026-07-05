-- Seed Next Week: Delete old journeys & create 5 journeys per route per day
-- for the week of June 22-28, 2026
-- Organization: CTM Maroc (ce69d662-3794-4570-a54e-e8057e41ee19)

BEGIN;

-- ===========================
-- 1. Delete old journeys
-- ===========================
DELETE FROM tickets;
DELETE FROM bookings;
DELETE FROM journeys;

-- ===========================
-- 2. Create 5 journeys per route per day for next week
-- ===========================
DO $$
DECLARE
  route_rec RECORD;
  bus_id_val UUID;
  bus_ids UUID[] := ARRAY(
    SELECT id FROM buses ORDER BY registration_number
  );
  bus_idx INT := 1;
  start_date DATE := '2026-06-22';
  end_date DATE := '2026-06-28';
  curr_date DATE;
  dep_time TEXT;
  arr_time TEXT;
  dep_hour INT;
  dep_minute INT;
  -- 5 departure times per route
  -- Casablanca→Tangier (5h travel)
  ct_hours INT[] := ARRAY[6, 9, 12, 15, 18];
  ct_minutes INT[] := ARRAY[0, 0, 0, 0, 0];
  -- Marrakech→Rabat (4.5h = 270min travel)
  mr_hours INT[] := ARRAY[6, 9, 12, 15, 18];
  mr_minutes INT[] := ARRAY[0, 30, 0, 0, 30];
  -- Fes→Casablanca (5h travel)
  fc_hours INT[] := ARRAY[5, 8, 11, 14, 17];
  fc_minutes INT[] := ARRAY[0, 0, 0, 0, 0];
  -- Agadir→Casablanca (8h travel)
  ac_hours INT[] := ARRAY[6, 9, 14, 18, 22];
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
    -- Process each route
    FOR route_rec IN SELECT * FROM routes ORDER BY name LOOP
      -- Assign schedule based on route
      IF route_rec.id = '0b5f2d3a-4abf-4fb3-bb02-63fffe0edcba' THEN
        route_hours := ct_hours;
        route_minutes := ct_minutes;
        travel_minutes := 300; -- 5h
      ELSIF route_rec.id = '2f5c99d5-3810-4bf4-9b73-c5353a5418c0' THEN
        route_hours := mr_hours;
        route_minutes := mr_minutes;
        travel_minutes := 270; -- 4.5h
      ELSIF route_rec.id = '1dd130ae-ee66-4115-8c22-c7165648dea4' THEN
        route_hours := fc_hours;
        route_minutes := fc_minutes;
        travel_minutes := 300; -- 5h
      ELSIF route_rec.id = 'eab1eb8a-8113-484b-b316-4bd6a3a25641' THEN
        route_hours := ac_hours;
        route_minutes := ac_minutes;
        travel_minutes := 480; -- 8h
      ELSE
        CONTINUE;
      END IF;

      -- Create 5 journeys for this route on this day
      FOR i IN 1..5 LOOP
        dep_hour := route_hours[i];
        dep_minute := route_minutes[i];

        -- Pick bus (round-robin)
        bus_id_val := bus_ids[bus_idx];
        bus_idx := (bus_idx % array_length(bus_ids, 1)) + 1;

        dep_total_min := dep_hour * 60 + dep_minute;
        arr_total_min := dep_total_min + travel_minutes;

        dep_time_formatted := lpad(dep_hour::text, 2, '0') || ':' || lpad(dep_minute::text, 2, '0') || ':00';

        -- Calculate arrival - may be next day
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
