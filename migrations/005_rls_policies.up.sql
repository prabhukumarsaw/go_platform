-- 005_rls_policies.up.sql
-- Row-Level Security policies for tenant isolation.
-- The Go middleware sets app.tenant_id and app.is_super_admin via SET LOCAL
-- at the start of every transaction. RLS policies use these to filter rows.

-- ──────────────────────────────────────────────
-- ARTICLES: tenant isolation + national content exemption
-- ──────────────────────────────────────────────
ALTER TABLE articles ENABLE ROW LEVEL SECURITY;
ALTER TABLE articles FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON articles
    USING (
        tenant_id = current_setting('app.tenant_id', true)::int
        OR is_national = TRUE
        OR current_setting('app.is_super_admin', true)::boolean = true
    );

-- ──────────────────────────────────────────────
-- STORIES
-- ──────────────────────────────────────────────
ALTER TABLE stories ENABLE ROW LEVEL SECURITY;
ALTER TABLE stories FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON stories
    USING (
        tenant_id = current_setting('app.tenant_id', true)::int
        OR current_setting('app.is_super_admin', true)::boolean = true
    );

-- ──────────────────────────────────────────────
-- CATEGORIES (Global & Tenant Accessible)
-- ──────────────────────────────────────────────
ALTER TABLE categories ENABLE ROW LEVEL SECURITY;
ALTER TABLE categories FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON categories
    USING (
        tenant_id = 1
        OR tenant_id = current_setting('app.tenant_id', true)::int
        OR current_setting('app.is_super_admin', true)::boolean = true
    );

-- ──────────────────────────────────────────────
-- MEDIA
-- ──────────────────────────────────────────────
ALTER TABLE media ENABLE ROW LEVEL SECURITY;
ALTER TABLE media FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON media
    USING (
        tenant_id = current_setting('app.tenant_id', true)::int
        OR current_setting('app.is_super_admin', true)::boolean = true
    );

-- ──────────────────────────────────────────────
-- ROLES
-- ──────────────────────────────────────────────
ALTER TABLE roles ENABLE ROW LEVEL SECURITY;
ALTER TABLE roles FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON roles
    USING (
        tenant_id = current_setting('app.tenant_id', true)::int
        OR current_setting('app.is_super_admin', true)::boolean = true
    );

-- ──────────────────────────────────────────────
-- ROLE_MENU_ACTIONS
-- ──────────────────────────────────────────────
ALTER TABLE role_menu_actions ENABLE ROW LEVEL SECURITY;
ALTER TABLE role_menu_actions FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON role_menu_actions
    USING (
        tenant_id = current_setting('app.tenant_id', true)::int
        OR current_setting('app.is_super_admin', true)::boolean = true
    );

-- ──────────────────────────────────────────────
-- COMMENTS (scoped via article's tenant, but no direct tenant_id column)
-- Comments don't have their own tenant_id; they are scoped through articles.
-- RLS is not enabled on comments since they are always accessed via
-- article_id JOINs which are already tenant-isolated.
-- ──────────────────────────────────────────────

-- ──────────────────────────────────────────────
-- AD SLOTS (will be created in migration 006)
-- ──────────────────────────────────────────────

-- ──────────────────────────────────────────────
-- BYPASS ROLE for migrations and super_admin queries
-- ──────────────────────────────────────────────
-- The application's migration user and connection pool should use a role
-- that BYPASSRLS to avoid RLS during migrations and seed data.
-- Example: ALTER ROLE newsadmin BYPASSRLS;
-- This is set during database setup, not in migrations.
