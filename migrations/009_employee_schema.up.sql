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
    -- Platform Chief Editor (User ID = 1)
    IF t_national IS NOT NULL AND EXISTS (SELECT 1 FROM users WHERE id = 1) THEN
        INSERT INTO employees (user_id, tenant_id, employee_code, department, designation)
        VALUES (1, t_national, 'EMP-NAT-001', 'Editorial', 'Chief Editor & Administrator')
        ON CONFLICT (user_id, tenant_id) DO NOTHING;
    END IF;
END $$;
