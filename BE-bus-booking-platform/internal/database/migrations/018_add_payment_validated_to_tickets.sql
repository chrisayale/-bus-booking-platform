-- Add payment_validated column to tickets table
ALTER TABLE tickets ADD COLUMN IF NOT EXISTS payment_validated BOOLEAN NOT NULL DEFAULT FALSE;