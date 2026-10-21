-- Bus Booking Platform - Add unique constraint on user phone for agents/controllers
-- Ensures agents and controllers have unique phone numbers

-- Create a partial unique index that only applies to agents and controllers
-- This allows passengers and owners to share phone numbers if needed
CREATE UNIQUE INDEX IF NOT EXISTS unique_agent_controller_phone
ON users (phone)
WHERE role IN ('agent', 'controller') AND phone IS NOT NULL AND phone != '';