-- Bus Booking Platform - Initial Schema
-- PostgreSQL 16+

-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================
-- ORGANIZATIONS (single bus company owner)
-- ============================================================
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    phone VARCHAR(50),
    logo_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- USERS (owner + passengers)
-- ============================================================
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    phone VARCHAR(50),
    role VARCHAR(20) NOT NULL CHECK (role IN ('owner', 'passenger')),
    organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    is_verified BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- FLEET: BUSES
-- ============================================================
CREATE TABLE buses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    registration_number VARCHAR(100) NOT NULL,
    capacity INT NOT NULL CHECK (capacity > 0),
    seat_configuration JSONB NOT NULL DEFAULT '{"rows": 10, "columns": 4, "layout": "2+2"}',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    image_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(organization_id, registration_number)
);

-- ============================================================
-- ROUTES (origin → destination with dynamic stops)
-- ============================================================
CREATE TABLE routes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    origin VARCHAR(255) NOT NULL,
    destination VARCHAR(255) NOT NULL,
    base_price DECIMAL(10, 2) NOT NULL CHECK (base_price >= 0),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Dynamic intermediary stops along a route
CREATE TABLE route_stops (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    route_id UUID NOT NULL REFERENCES routes(id) ON DELETE CASCADE,
    stop_order INT NOT NULL CHECK (stop_order >= 0),
    location VARCHAR(255) NOT NULL,
    arrival_offset_minutes INT,  -- minutes after journey departure
    departure_offset_minutes INT, -- minutes after journey departure
    stop_price DECIMAL(10, 2) NOT NULL DEFAULT 0 CHECK (stop_price >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(route_id, stop_order)
);

-- ============================================================
-- JOURNEYS (bus + route + date)
-- ============================================================
CREATE TABLE journeys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    route_id UUID NOT NULL REFERENCES routes(id) ON DELETE CASCADE,
    bus_id UUID NOT NULL REFERENCES buses(id) ON DELETE CASCADE,
    departure_date DATE NOT NULL,
    departure_time TIME NOT NULL,
    arrival_date DATE NOT NULL,
    arrival_time TIME NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'scheduled'
        CHECK (status IN ('scheduled', 'boarding', 'departed', 'completed', 'cancelled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- BOOKINGS
-- ============================================================
CREATE TABLE bookings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    booking_reference VARCHAR(20) UNIQUE NOT NULL,
    guest_email VARCHAR(255),
    guest_name VARCHAR(255),
    total_amount DECIMAL(10, 2) NOT NULL CHECK (total_amount >= 0),
    status VARCHAR(20) NOT NULL DEFAULT 'confirmed'
        CHECK (status IN ('pending', 'confirmed', 'cancelled', 'refunded')),
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- TICKETS (individual seat per passenger per journey)
-- ============================================================
CREATE TABLE tickets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id UUID NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
    journey_id UUID NOT NULL REFERENCES journeys(id) ON DELETE CASCADE,
    seat_number VARCHAR(10) NOT NULL,
    passenger_name VARCHAR(255) NOT NULL,
    passenger_phone VARCHAR(50),
    boarding_stop_id UUID REFERENCES route_stops(id) ON DELETE SET NULL,
    dropoff_stop_id UUID REFERENCES route_stops(id) ON DELETE SET NULL,
    price DECIMAL(10, 2) NOT NULL CHECK (price >= 0),
    qr_token VARCHAR(255) UNIQUE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'checked_in', 'missed', 'cancelled', 'refunded')),
    checked_in_at TIMESTAMPTZ,
    download_token VARCHAR(255) UNIQUE,
    download_token_expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(journey_id, seat_number)
);

-- ============================================================
-- INDEXES
-- ============================================================
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_buses_org ON buses(organization_id);
CREATE INDEX idx_routes_org ON routes(organization_id);
CREATE INDEX idx_route_stops_route ON route_stops(route_id);
CREATE INDEX idx_journeys_org ON journeys(organization_id);
CREATE INDEX idx_journeys_route ON journeys(route_id);
CREATE INDEX idx_journeys_date ON journeys(departure_date);
CREATE INDEX idx_journeys_status ON journeys(status);
CREATE INDEX idx_bookings_user ON bookings(user_id);
CREATE INDEX idx_bookings_reference ON bookings(booking_reference);
CREATE INDEX idx_tickets_journey ON tickets(journey_id);
CREATE INDEX idx_tickets_booking ON tickets(booking_id);
CREATE INDEX idx_tickets_qr ON tickets(qr_token);
CREATE INDEX idx_tickets_download ON tickets(download_token);
CREATE INDEX idx_tickets_status ON tickets(status);
