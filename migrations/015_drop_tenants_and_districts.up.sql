-- 015_drop_tenants_and_districts.up.sql
-- Completely drop legacy tenants, districts, user_tenant_mappings and all associated foreign keys

-- 1. Drop foreign key constraints referencing districts
ALTER TABLE IF EXISTS articles DROP CONSTRAINT IF EXISTS articles_district_id_fkey;
ALTER TABLE IF EXISTS employees DROP CONSTRAINT IF EXISTS employees_district_id_fkey;
ALTER TABLE IF EXISTS epapers DROP CONSTRAINT IF EXISTS epapers_district_id_fkey;
ALTER TABLE IF EXISTS user_district_scopes DROP CONSTRAINT IF EXISTS user_district_scopes_district_id_fkey;

-- 2. Drop foreign key constraints referencing tenants
ALTER TABLE IF EXISTS stories DROP CONSTRAINT IF EXISTS stories_tenant_id_fkey;
ALTER TABLE IF EXISTS articles DROP CONSTRAINT IF EXISTS articles_tenant_id_fkey;
ALTER TABLE IF EXISTS categories DROP CONSTRAINT IF EXISTS categories_tenant_id_fkey;
ALTER TABLE IF EXISTS roles DROP CONSTRAINT IF EXISTS roles_tenant_id_fkey;
ALTER TABLE IF EXISTS role_menu_actions DROP CONSTRAINT IF EXISTS role_menu_actions_tenant_id_fkey;
ALTER TABLE IF EXISTS user_tenant_mappings DROP CONSTRAINT IF EXISTS user_tenant_mappings_tenant_id_fkey;
ALTER TABLE IF EXISTS user_district_scopes DROP CONSTRAINT IF EXISTS user_district_scopes_tenant_id_fkey;
ALTER TABLE IF EXISTS user_permission_overrides DROP CONSTRAINT IF EXISTS user_permission_overrides_tenant_id_fkey;
ALTER TABLE IF EXISTS abac_policies DROP CONSTRAINT IF EXISTS abac_policies_tenant_id_fkey;
ALTER TABLE IF EXISTS media DROP CONSTRAINT IF EXISTS media_tenant_id_fkey;
ALTER TABLE IF EXISTS comments DROP CONSTRAINT IF EXISTS comments_tenant_id_fkey;
ALTER TABLE IF EXISTS ads DROP CONSTRAINT IF EXISTS ads_tenant_id_fkey;
ALTER TABLE IF EXISTS ad_slots DROP CONSTRAINT IF EXISTS ad_slots_tenant_id_fkey;
ALTER TABLE IF EXISTS web_stories DROP CONSTRAINT IF EXISTS web_stories_tenant_id_fkey;
ALTER TABLE IF EXISTS polls DROP CONSTRAINT IF EXISTS polls_tenant_id_fkey;
ALTER TABLE IF EXISTS epapers DROP CONSTRAINT IF EXISTS epapers_tenant_id_fkey;
ALTER TABLE IF EXISTS employees DROP CONSTRAINT IF EXISTS employees_tenant_id_fkey;
ALTER TABLE IF EXISTS feedbacks DROP CONSTRAINT IF EXISTS feedbacks_tenant_id_fkey;
ALTER TABLE IF EXISTS tags DROP CONSTRAINT IF EXISTS tags_tenant_id_fkey;

-- 3. Drop user_district_scopes table (not needed with Hierarchical Taxonomy)
DROP TABLE IF EXISTS user_district_scopes CASCADE;

-- 4. Create unified user_roles table & migrate mappings
CREATE TABLE IF NOT EXISTS user_roles (
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id     INT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    is_active   BOOLEAN DEFAULT TRUE,
    assigned_by BIGINT REFERENCES users(id),
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (user_id, role_id)
);

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'user_tenant_mappings') THEN
        INSERT INTO user_roles (user_id, role_id, is_active, assigned_by)
        SELECT user_id, role_id, is_active, assigned_by FROM user_tenant_mappings
        ON CONFLICT (user_id, role_id) DO NOTHING;
    END IF;
END $$;

-- 5. Drop user_tenant_mappings, districts, and tenants tables completely
DROP TABLE IF EXISTS user_tenant_mappings CASCADE;
DROP TABLE IF EXISTS districts CASCADE;
DROP TABLE IF EXISTS tenants CASCADE;

-- 6. Ensure categories table has unique index on slug for clean hierarchical lookups
CREATE UNIQUE INDEX IF NOT EXISTS idx_categories_slug_unique ON categories(slug);
