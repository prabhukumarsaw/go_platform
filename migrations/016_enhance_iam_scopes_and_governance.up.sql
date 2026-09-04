-- 016_enhance_iam_scopes_and_governance.up.sql
-- Enhanced IAM Scopes, Role Governance & Bureau Scoping

-- 1. Modernize roles table: drop tenant_id, ensure unique name, add is_system and timestamps
ALTER TABLE roles DROP CONSTRAINT IF EXISTS roles_tenant_id_name_key;
ALTER TABLE roles DROP COLUMN IF EXISTS tenant_id CASCADE;
ALTER TABLE roles DROP CONSTRAINT IF EXISTS roles_name_unique;
ALTER TABLE roles ADD CONSTRAINT roles_name_unique UNIQUE(name);
ALTER TABLE roles ADD COLUMN IF NOT EXISTS is_system BOOLEAN DEFAULT FALSE;
ALTER TABLE roles ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT NOW();

-- Mark default foundational roles as system roles
UPDATE roles SET is_system = TRUE WHERE id IN (1, 2, 3, 4) OR LOWER(name) IN ('super administrator', 'superadmin', 'senior editor', 'editor', 'reporter', 'bureau chief');

-- 2. Create user_category_scopes table for granular editorial bureau scoping
CREATE TABLE IF NOT EXISTS user_category_scopes (
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category_id INT NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    assigned_by BIGINT REFERENCES users(id),
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (user_id, category_id)
);

CREATE INDEX IF NOT EXISTS idx_user_category_scopes_user_id ON user_category_scopes(user_id);
CREATE INDEX IF NOT EXISTS idx_user_category_scopes_category_id ON user_category_scopes(category_id);

-- 3. Modernize role_menu_actions table: drop legacy tenant_id and composite PK, set clean (role_id, menu_action_id) PK
ALTER TABLE role_menu_actions DROP CONSTRAINT IF EXISTS role_menu_actions_pkey;
ALTER TABLE role_menu_actions DROP COLUMN IF EXISTS tenant_id CASCADE;
ALTER TABLE role_menu_actions ADD PRIMARY KEY (role_id, menu_action_id);

-- 4. Modernize abac_policies and user_permission_overrides
ALTER TABLE abac_policies DROP COLUMN IF EXISTS tenant_id CASCADE;
ALTER TABLE user_permission_overrides DROP COLUMN IF EXISTS tenant_id CASCADE;

-- 5. Ensure core menu actions exist for all system menus
INSERT INTO menu_actions (menu_id, action, label)
SELECT m.id, a.action, a.label
FROM menus m
CROSS JOIN (VALUES 
    ('view', 'View'),
    ('create', 'Create'),
    ('edit', 'Edit'),
    ('delete', 'Delete'),
    ('publish', 'Publish'),
    ('export', 'Export')
) a(action, label)
ON CONFLICT (menu_id, action) DO NOTHING;

-- 6. Super Administrator gets all menu actions by default
INSERT INTO role_menu_actions (role_id, menu_action_id)
SELECT 1, ma.id
FROM menu_actions ma
ON CONFLICT (role_id, menu_action_id) DO NOTHING;

-- 7. Seed Additional Editorial Roles if not exist
INSERT INTO roles (name, description, is_system)
VALUES 
    ('Fact Checker', 'Responsible for claim verification and truth rating', TRUE),
    ('Multimedia Producer', 'Manages video, web stories, and audio podcasts', TRUE)
ON CONFLICT (name) DO UPDATE SET is_system = TRUE, description = EXCLUDED.description;
