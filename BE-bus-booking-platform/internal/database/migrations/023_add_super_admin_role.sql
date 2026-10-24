-- Migration to add super_admin role to users table constraint
-- super_admin is an alias for platform_admin (highest privilege role)
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('owner', 'passenger', 'platform_admin', 'agent', 'controller', 'super_admin'));