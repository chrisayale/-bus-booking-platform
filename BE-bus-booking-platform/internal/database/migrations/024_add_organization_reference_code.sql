-- Bus Booking Platform - Add reference_code to organizations
-- Adds a 6-digit reference code for agency identification

ALTER TABLE organizations ADD COLUMN IF NOT EXISTS reference_code VARCHAR(6) UNIQUE;

-- Create index for faster lookups
CREATE INDEX IF NOT EXISTS idx_organizations_reference_code ON organizations(reference_code);