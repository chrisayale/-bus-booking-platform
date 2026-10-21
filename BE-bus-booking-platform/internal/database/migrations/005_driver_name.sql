-- Add driver_name to buses for assigning a driver to each bus
ALTER TABLE buses
ADD COLUMN IF NOT EXISTS driver_name VARCHAR(255);
