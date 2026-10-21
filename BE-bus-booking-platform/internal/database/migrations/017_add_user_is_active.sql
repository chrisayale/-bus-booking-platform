-- Bus Booking Platform - Add is_active column to users table
-- Add is_active column to allow admins to activate/deactivate user accounts

ALTER TABLE users ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT TRUE;

-- Create index for better query performance on active status
CREATE INDEX IF NOT EXISTS idx_users_is_active ON users(is_active);
