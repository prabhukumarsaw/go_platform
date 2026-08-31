-- 009_employee_schema.up.sql
-- Newsroom Staff & Employee Governance System

CREATE TABLE IF NOT EXISTS employees (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id       INT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    employee_code   VARCHAR(50) NOT NULL UNIQUE,
    department      VARCHAR(50) NOT NULL DEFAULT 'Editorial',
    designation     VARCHAR(100) NOT NULL,
    district_id     INT REFERENCES districts(id) ON DELETE SET NULL,
    joined_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, tenant_id)
);

CREATE INDEX IF NOT EXISTS idx_employees_tenant ON employees(tenant_id, department);
CREATE INDEX IF NOT EXISTS idx_employees_user ON employees(user_id);

-- Seed initial employees for the seeded staff accounts
DO $$
DECLARE
    t_national INT;
    t_mh INT;
    t_wb INT;
    d_pune INT;
BEGIN
    SELECT id INTO t_national FROM tenants WHERE slug='national';
    SELECT id INTO t_mh FROM tenants WHERE slug='maharashtra';
    SELECT id INTO t_wb FROM tenants WHERE slug='west-bengal';
    SELECT id INTO d_pune FROM districts WHERE slug='pune';

    -- Aditi Sharma (National Desk Head)
    IF t_national IS NOT NULL THEN
        INSERT INTO employees (user_id, tenant_id, employee_code, department, designation)
        VALUES (2, t_national, 'EMP-NAT-001', 'Desk', 'National Bureau Chief')
        ON CONFLICT (user_id, tenant_id) DO NOTHING;
    END IF;

    -- Rajesh Patil (Maharashtra Editor)
    IF t_mh IS NOT NULL THEN
        INSERT INTO employees (user_id, tenant_id, employee_code, department, designation)
        VALUES (3, t_mh, 'EMP-MH-001', 'Editorial', 'Senior State Editor')
        ON CONFLICT (user_id, tenant_id) DO NOTHING;
    END IF;

    -- Sneha Deshmukh (Pune District Reporter)
    IF t_mh IS NOT NULL AND d_pune IS NOT NULL THEN
        INSERT INTO employees (user_id, tenant_id, employee_code, department, designation, district_id)
        VALUES (4, t_mh, 'EMP-MH-002', 'Bureau', 'Special District Correspondent', d_pune)
        ON CONFLICT (user_id, tenant_id) DO NOTHING;
    END IF;

    -- Sourav Ganguly (Bengal Editor)
    IF t_wb IS NOT NULL THEN
        INSERT INTO employees (user_id, tenant_id, employee_code, department, designation)
        VALUES (5, t_wb, 'EMP-WB-001', 'Editorial', 'Bengal Bureau Chief')
        ON CONFLICT (user_id, tenant_id) DO NOTHING;
    END IF;

    -- Kavita Iyer (Community Moderator)
    IF t_national IS NOT NULL THEN
        INSERT INTO employees (user_id, tenant_id, employee_code, department, designation)
        VALUES (6, t_national, 'EMP-NAT-002', 'SocialMedia', 'UGC & Moderation Lead')
        ON CONFLICT (user_id, tenant_id) DO NOTHING;
    END IF;
END $$;
