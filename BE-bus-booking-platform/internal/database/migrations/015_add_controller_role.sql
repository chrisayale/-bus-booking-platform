-- Migration to add controller role to users table constraint
-- Includes all valid application roles: owner, passenger, platform_admin, agent, controller
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('owner', 'passenger', 'platform_admin', 'agent', 'controller'));
