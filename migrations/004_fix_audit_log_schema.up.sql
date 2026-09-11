-- 004_fix_audit_log_schema.up.sql
-- Fix audit log schema to handle both legacy menu action IDs and new RAS permission strings
-- This resolves the type mismatch causing audit log failures

-- ──────────────────────────────────────────────
-- 1. Add new columns to handle RAS permissions
-- ──────────────────────────────────────────────
ALTER TABLE permission_audit_log ADD COLUMN IF NOT EXISTS permission_string VARCHAR(100);
ALTER TABLE permission_audit_log ADD COLUMN IF NOT EXISTS resource VARCHAR(50);
ALTER TABLE permission_audit_log ADD COLUMN IF NOT EXISTS action VARCHAR(50);
ALTER TABLE permission_audit_log ADD COLUMN IF NOT EXISTS scope VARCHAR(50);

-- Add indexes for the new columns
CREATE INDEX IF NOT EXISTS idx_audit_log_permission_string ON permission_audit_log(permission_string);
CREATE INDEX IF NOT EXISTS idx_audit_log_resource ON permission_audit_log(resource);
CREATE INDEX IF NOT EXISTS idx_audit_log_action ON permission_audit_log(action);

-- ──────────────────────────────────────────────
-- 2. Make menu_action_id nullable to support both systems
-- ──────────────────────────────────────────────
ALTER TABLE permission_audit_log ALTER COLUMN menu_action_id DROP NOT NULL;

-- ──────────────────────────────────────────────
-- 3. Add a trigger to normalize permission strings
-- ──────────────────────────────────────────────
CREATE OR REPLACE FUNCTION normalize_audit_permission()
RETURNS TRIGGER AS $$
BEGIN
    -- If menu_action_id is provided but permission_string is not, try to look it up
    IF NEW.menu_action_id IS NOT NULL AND NEW.permission_string IS NULL THEN
        SELECT ma.action || '.' || m.name 
        INTO NEW.permission_string
        FROM menu_actions ma
        JOIN menus m ON m.id = ma.menu_id
        WHERE ma.id = NEW.menu_action_id;
    END IF;
    
    -- If permission_string is provided but resource/action/scope are not, parse it
    IF NEW.permission_string IS NOT NULL THEN
        -- Parse format: "resource:action:scope" or "resource.action"
        IF NEW.permission_string LIKE '%:%' THEN
            -- RAS format: "resource:action:scope"
            NEW.resource := split_part(NEW.permission_string, ':', 1);
            NEW.action := split_part(NEW.permission_string, ':', 2);
            NEW.scope := split_part(NEW.permission_string, ':', 3);
        ELSIF NEW.permission_string LIKE '%.%' THEN
            -- Legacy format: "resource.action"
            NEW.resource := split_part(NEW.permission_string, '.', 1);
            NEW.action := split_part(NEW.permission_string, '.', 2);
            NEW.scope := 'all';
        END IF;
    END IF;
    
    -- If action_name is not set, use the parsed action
    IF NEW.action_name IS NULL OR NEW.action_name = '' THEN
        NEW.action_name := COALESCE(NEW.action, NEW.permission_string, 'UNKNOWN');
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_normalize_audit_permission
    BEFORE INSERT OR UPDATE ON permission_audit_log
    FOR EACH ROW
    EXECUTE FUNCTION normalize_audit_permission();

-- ──────────────────────────────────────────────
-- 4. Update existing audit log entries
-- ──────────────────────────────────────────────
-- Populate permission_string for existing entries that have menu_action_id
UPDATE permission_audit_log pal
SET permission_string = 
    COALESCE(
        ma.action || '.' || m.name,
        pal.action_name
    )
FROM menu_actions ma
JOIN menus m ON m.id = ma.menu_id
WHERE pal.menu_action_id IS NOT NULL 
  AND pal.permission_string IS NULL
  AND ma.id = pal.menu_action_id;

-- ──────────────────────────────────────────────
-- 5. Add a function to safely log permissions
-- ──────────────────────────────────────────────
CREATE OR REPLACE FUNCTION log_permission_check(
    p_user_id BIGINT,
    p_decision VARCHAR(20)
)
RETURNS VOID AS $$
BEGIN
    INSERT INTO permission_audit_log (
        user_id, 
        decision
    ) VALUES (
        p_user_id,
        p_decision
    );
    
EXCEPTION WHEN OTHERS THEN
    -- Log the error but don't fail the transaction
    RAISE WARNING 'Failed to log permission check: %', SQLERRM;
END;
$$ LANGUAGE plpgsql;

-- ──────────────────────────────────────────────
-- 6. Add enhanced function with all parameters
-- ──────────────────────────────────────────────
CREATE OR REPLACE FUNCTION log_permission_check_full(
    p_user_id BIGINT,
    p_decision VARCHAR(20),
    p_menu_action_id INT,
    p_permission_string VARCHAR(100),
    p_action_name VARCHAR(80),
    p_reason TEXT,
    p_ip_address INET,
    p_user_agent TEXT,
    p_request_id VARCHAR(100)
)
RETURNS VOID AS $$
BEGIN
    INSERT INTO permission_audit_log (
        user_id, 
        menu_action_id, 
        permission_string, 
        action_name, 
        decision, 
        reason, 
        ip_address, 
        user_agent, 
        request_id
    ) VALUES (
        p_user_id,
        p_menu_action_id,
        p_permission_string,
        p_action_name,
        p_decision,
        p_reason,
        p_ip_address,
        p_user_agent,
        p_request_id
    );
    
EXCEPTION WHEN OTHERS THEN
    -- Log the error but don't fail the transaction
    RAISE WARNING 'Failed to log permission check: %', SQLERRM;
END;
$$ LANGUAGE plpgsql;

-- ──────────────────────────────────────────────
-- 7. Add view for unified audit log access
-- ──────────────────────────────────────────────
CREATE OR REPLACE VIEW v_unified_audit_log AS
SELECT 
    pal.id,
    pal.user_id,
    pal.menu_action_id,
    pal.permission_string,
    COALESCE(pal.resource, split_part(COALESCE(pal.permission_string, ''), ':', 1)) as resource,
    COALESCE(pal.action, split_part(COALESCE(pal.permission_string, ''), ':', 2)) as action,
    COALESCE(pal.scope, split_part(COALESCE(pal.permission_string, ''), ':', 3), 'all') as scope,
    pal.action_name,
    pal.decision,
    pal.reason,
    pal.ip_address,
    pal.user_agent,
    pal.request_id,
    pal.created_at
FROM permission_audit_log pal;

-- ──────────────────────────────────────────────
-- 7. Update existing IAM service to use new logging function
-- ──────────────────────────────────────────────
-- This will be handled in the Go code updates