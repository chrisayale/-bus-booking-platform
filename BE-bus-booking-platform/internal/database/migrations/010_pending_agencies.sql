-- Bus Booking Platform - Pending Agencies Table
-- Creates table to store agency creation requests awaiting admin approval

-- Enable UUID generation if not already enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================
-- PENDING AGENCIES (agency creation requests awaiting approval)
-- ============================================================
CREATE TABLE IF NOT EXISTS pending_agencies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    agency_name VARCHAR(255) NOT NULL,
    manager_name VARCHAR(255) NOT NULL,
    address VARCHAR(255) NOT NULL,
    contact_phone VARCHAR(50) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    default_password VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- INDEXES
-- ============================================================
CREATE INDEX IF NOT EXISTS idx_pending_agencies_email ON pending_agencies(email);
CREATE INDEX IF NOT EXISTS idx_pending_agencies_status ON pending_agencies(status);
CREATE INDEX IF NOT EXISTS idx_pending_agencies_created_at ON pending_agencies(created_at);