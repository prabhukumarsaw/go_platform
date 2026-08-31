-- 001_tenants.up.sql
-- Tenants represent state editions (Maharashtra, West Bengal, etc.)
-- plus one special "National" tenant for shared cross-state content.

CREATE TABLE tenants (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(100) NOT NULL,
    slug        VARCHAR(50) UNIQUE NOT NULL,
    is_national BOOLEAN DEFAULT FALSE,
    logo_url    TEXT,
    config      JSONB DEFAULT '{}',
    is_active   BOOLEAN DEFAULT TRUE,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON TABLE tenants IS 'State editions + 1 National tenant. Every tenant-scoped row carries a tenant_id FK.';
COMMENT ON COLUMN tenants.is_national IS 'Only one row should have TRUE. National content is visible across all state tenants.';
COMMENT ON COLUMN tenants.config IS 'Tenant-specific overrides: theme colors, feature flags, layout density, etc.';

CREATE TABLE districts (
    id          SERIAL PRIMARY KEY,
    tenant_id   INT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name        VARCHAR(100) NOT NULL,
    slug        VARCHAR(50) NOT NULL,
    is_active   BOOLEAN DEFAULT TRUE,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, slug)
);

COMMENT ON TABLE districts IS 'Districts within a state tenant. One level deep — no ward granularity unless hyperlocal features need it.';

-- Supported languages (enumerated for validation, not for per-row storage)
CREATE TABLE languages (
    code    VARCHAR(10) PRIMARY KEY,
    name    VARCHAR(50) NOT NULL,
    script  VARCHAR(30) NOT NULL,
    is_active BOOLEAN DEFAULT TRUE
);

INSERT INTO languages (code, name, script) VALUES
    ('en', 'English',  'Latin'),
    ('hi', 'Hindi',    'Devanagari'),
    ('bn', 'Bengali',  'Bengali'),
    ('mr', 'Marathi',  'Devanagari'),
    ('ta', 'Tamil',    'Tamil');
