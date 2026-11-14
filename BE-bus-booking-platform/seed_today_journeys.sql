-- Seed script: Add journeys for TODAY for all 10 DRC agencies
-- This script adds 3 journeys per agency (Morning, Midday, Evening) on their main routes
-- Run date: 2026-11-14

BEGIN;

-- ===========================
-- 1. Delete existing journeys for today (to avoid duplicates)
-- ===========================
DELETE FROM journeys WHERE departure_date = CURRENT_DATE;

-- ===========================
-- 2. Add journeys for today for each agency's main route
-- ===========================
DO $$
DECLARE
    org_rec RECORD;
    route_rec RECORD;
    bus_rec RECORD;
    bus_ids UUID[];
    bus_count INT;
    bus_idx INT := 1;
    dep_time TIME;
    arr_time TIME;
    arr_date DATE;
    travel_minutes INT;
    journey_count INT := 0;

    -- Today's date
    today_date DATE := CURRENT_DATE;

    -- Schedule: Morning (6:30), Midday (12:00), Evening (18:30)
    -- With travel time of ~3 hours per route
    sched_hours INT[] := ARRAY[6, 12, 18];
    sched_minutes INT[] := ARRAY[30, 0, 30];
    sched_labels TEXT[] := ARRAY['MORNING', 'MIDDAY', 'EVENING'];
BEGIN
    -- Process each agency by reference_code order
    FOR org_rec IN
        SELECT id, name, reference_code
        FROM organizations
        WHERE reference_code IS NOT NULL
        AND reference_code LIKE '1000%'
        ORDER BY reference_code
    LOOP
        RAISE NOTICE 'Processing agency: % (ref: %)', org_rec.name, org_rec.reference_code;

        -- Get the main route for this agency (first route alphabetically as proxy for main route)
        -- In production, you'd identify the main route more precisely
        SELECT * INTO route_rec
        FROM routes
        WHERE organization_id = org_rec.id
        ORDER BY name
        LIMIT 1;

        IF NOT FOUND THEN
            RAISE NOTICE '  No routes found for %, skipping', org_rec.name;
            CONTINUE;
        END IF;

        RAISE NOTICE '  Main route: % (% → %)', route_rec.name, route_rec.origin, route_rec.destination;

        -- Calculate travel time based on base_price (roughly 1 hour per $10, min 2 hours)
        travel_minutes := GREATEST(120, FLOOR(route_rec.base_price * 6)::int);

        -- Get active buses for this agency
        SELECT ARRAY_AGG(id ORDER BY created_at) INTO bus_ids
        FROM buses
        WHERE organization_id = org_rec.id AND is_active = true;

        bus_count := array_length(bus_ids, 1);
        IF bus_count IS NULL OR bus_count = 0 THEN
            RAISE NOTICE '  No active buses for %, skipping', org_rec.name;
            CONTINUE;
        END IF;

        RAISE NOTICE '  Using % buses', bus_count;

        -- Create 3 journeys per agency (Morning, Midday, Evening)
        FOR i IN 1..3 LOOP
            -- Pick bus in round-robin
            IF bus_idx > bus_count THEN
                bus_idx := 1;
            END IF;

            -- Calculate departure time
            dep_time := MAKE_TIME(sched_hours[i], sched_minutes[i], 0);

            -- Calculate arrival time
            DECLARE
                dep_total_min INT := sched_hours[i] * 60 + sched_minutes[i];
                arr_total_min INT := dep_total_min + travel_minutes;
            BEGIN
                IF arr_total_min >= 1440 THEN
                    arr_total_min := arr_total_min - 1440;
                    arr_date := today_date + 1;
                ELSE
                    arr_date := today_date;
                END IF;

                arr_time := MAKE_TIME(arr_total_min / 60, arr_total_min % 60, 0);
            END;

            -- Determine status based on current time
            DECLARE
                current_hour INT := EXTRACT(HOUR FROM CURRENT_TIME)::int;
                current_minute INT := EXTRACT(MINUTE FROM CURRENT_TIME)::int;
                status_text TEXT := 'scheduled';
            BEGIN
                IF sched_hours[i] < current_hour OR
                   (sched_hours[i] = current_hour AND sched_minutes[i] <= current_minute) THEN
                    IF sched_hours[i] < current_hour OR
                       (sched_hours[i] = current_hour AND sched_minutes[i] + 30 <= current_minute) THEN
                        status_text := 'departed';
                    ELSE
                        status_text := 'boarding';
                    END IF;
                END IF;

                -- Insert the journey
                INSERT INTO journeys (id, organization_id, route_id, bus_id, departure_date, departure_time, arrival_date, arrival_time, status)
                VALUES (
                    gen_random_uuid(),
                    org_rec.id,
                    route_rec.id,
                    bus_ids[bus_idx],
                    today_date,
                    dep_time,
                    arr_date,
                    arr_time,
                    status_text
                );

                journey_count := journey_count + 1;
                bus_idx := bus_idx + 1;

                RAISE NOTICE '  ✓ % → % | % | % | % | Bus: %',
                    route_rec.origin, route_rec.destination,
                    to_char(dep_time, 'HH24:MI'), status_text, sched_labels[i],
                    bus_ids[bus_idx-1];
            END;
        END LOOP;
    END LOOP;

    RAISE NOTICE 'Total journeys created: %', journey_count;
END $$;

COMMIT;

-- ===========================
-- 3. Verification query
-- ===========================
SELECT
    o.name as agency,
    o.reference_code,
    r.name as route,
    r.origin || ' → ' || r.destination as route_path,
    j.departure_time,
    j.arrival_time,
    j.status,
    b.registration_number as bus
FROM journeys j
JOIN organizations o ON j.organization_id = o.id
JOIN routes r ON j.route_id = r.id
JOIN buses b ON j.bus_id = b.id
WHERE j.departure_date = CURRENT_DATE
ORDER BY o.reference_code, j.departure_time;