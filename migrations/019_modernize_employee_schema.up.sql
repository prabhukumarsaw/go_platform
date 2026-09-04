-- 019_modernize_employee_schema.up.sql
-- Modernize employees table to unify with national platform architecture

-- 1. Drop obsolete multi-tenant and district constraints
ALTER TABLE IF EXISTS employees DROP CONSTRAINT IF EXISTS employees_user_id_tenant_id_key;
ALTER TABLE IF EXISTS employees DROP CONSTRAINT IF EXISTS employees_tenant_id_fkey;
ALTER TABLE IF EXISTS employees DROP CONSTRAINT IF EXISTS employees_district_id_fkey;

-- 2. Drop obsolete district_id and tenant_id columns
ALTER TABLE IF EXISTS employees DROP COLUMN IF EXISTS district_id;
ALTER TABLE IF EXISTS employees DROP COLUMN IF EXISTS tenant_id;

-- 3. Enforce 1-to-1 unique employee record per user
CREATE UNIQUE INDEX IF NOT EXISTS idx_employees_user_id ON employees(user_id);

-- 4. Seed / ensure an active employee record exists for user ID 1 (Platform Chief Editor)
INSERT INTO employees (user_id, employee_code, department, designation, joined_at, is_active)
SELECT 1, 'EMP-001', 'Editorial', 'Platform Chief Editor', NOW(), TRUE
WHERE EXISTS (SELECT 1 FROM users WHERE id = 1)
  AND NOT EXISTS (SELECT 1 FROM employees WHERE user_id = 1);
