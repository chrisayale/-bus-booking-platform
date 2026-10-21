-- Create join table for users and organizations (many-to-many)
CREATE TABLE IF NOT EXISTS user_organizations (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, organization_id)
);

-- Index for fast lookups by organization
CREATE INDEX IF NOT EXISTS idx_user_organizations_org ON user_organizations(organization_id);
-- Index for fast lookups by user
CREATE INDEX IF NOT EXISTS idx_user_organizations_user ON user_organizations(user_id);

-- Populate existing users with their current organization_id (if not null)
INSERT INTO user_organizations (user_id, organization_id, created_at)
SELECT id, organization_id, created_at
FROM users
WHERE organization_id IS NOT NULL
ON CONFLICT DO NOTHING;
