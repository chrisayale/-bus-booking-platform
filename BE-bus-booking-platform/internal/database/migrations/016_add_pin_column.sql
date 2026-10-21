-- Bus Booking Platform - Add PIN column to users
ALTER TABLE users ADD COLUMN IF NOT EXISTS pin VARCHAR(10);