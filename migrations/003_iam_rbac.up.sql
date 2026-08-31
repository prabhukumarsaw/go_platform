-- 003_iam_rbac.up.sql
-- Full RBAC + ABAC permission system.
-- Generalized from the ULB panel schema, evaluated in fixed 5-step order.

-- Roles are tenant-scoped: a "reporter" in Maharashtra is a different role row
-- than a "reporter" in West Bengal, even though they share the same name.
CREATE TABLE roles (
    id          SERIAL PRIMARY KEY,
    tenant_id   INT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name        VARCHAR(50) NOT NULL,
    description TEXT,
    is_system   BOOLEAN DEFAULT FALSE,
    is_active   BOOLEAN DEFAULT TRUE,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

COMMENT ON COLUMN roles.is_system IS 'System roles (reporter, editor, state_admin) cannot be deleted via the admin UI.';

-- Menu modules — the navigation/feature tree of the application.
CREATE TABLE menus (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(100) NOT NULL UNIQUE,
    label       VARCHAR(100) NOT NULL,
    parent_id   INT REFERENCES menus(id),
    sort_order  INT DEFAULT 0,
    icon        VARCHAR(50),
    is_active   BOOLEAN DEFAULT TRUE
);

-- Actions within each menu (the granular permission atoms).
CREATE TABLE menu_actions (
    id          SERIAL PRIMARY KEY,
    menu_id     INT NOT NULL REFERENCES menus(id) ON DELETE CASCADE,
    action      VARCHAR(30) NOT NULL,
    label       VARCHAR(100),
    UNIQUE(menu_id, action)
);

COMMENT ON TABLE menu_actions IS 'Permission atoms: each row = one thing a user can do (e.g., articles.PUBLISH, media.DELETE).';

-- RBAC grants: which role has which menu_action in which tenant.
CREATE TABLE role_menu_actions (
    role_id         INT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    menu_action_id  INT NOT NULL REFERENCES menu_actions(id) ON DELETE CASCADE,
    tenant_id       INT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (role_id, menu_action_id, tenant_id)
);

-- User ↔ Tenant ↔ Role mapping.
-- A user can have one role per tenant (use multiple rows for multiple tenants).
CREATE TABLE user_tenant_mappings (
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id   INT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    role_id     INT NOT NULL REFERENCES roles(id),
    is_active   BOOLEAN DEFAULT TRUE,
    assigned_by BIGINT REFERENCES users(id),
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (user_id, tenant_id)
);

-- District scoping: which districts a user can access within a tenant.
-- No rows = whole-tenant access (editorial assignment is tenant-wide).
CREATE TABLE user_district_scopes (
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id   INT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    district_id INT NOT NULL REFERENCES districts(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (user_id, tenant_id, district_id)
);

-- Permission overrides: grant or revoke specific menu_actions for specific users.
-- Evaluated BEFORE role grants (step 2-3 of the evaluation chain).
CREATE TABLE user_permission_overrides (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id       INT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    menu_action_id  INT NOT NULL REFERENCES menu_actions(id) ON DELETE CASCADE,
    effect          VARCHAR(10) NOT NULL CHECK (effect IN ('GRANT', 'REVOKE')),
    reason          TEXT NOT NULL,
    valid_from      TIMESTAMPTZ DEFAULT NOW(),
    valid_until     TIMESTAMPTZ,
    granted_by      BIGINT NOT NULL REFERENCES users(id),
    is_active       BOOLEAN DEFAULT TRUE,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_overrides_user_tenant ON user_permission_overrides(user_id, tenant_id)
    WHERE is_active = TRUE;

-- ABAC policies: additional attribute-based conditions checked AFTER RBAC passes.
CREATE TABLE abac_policies (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id   INT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    attribute   VARCHAR(30) NOT NULL,
    value       JSONB NOT NULL,
    is_active   BOOLEAN DEFAULT TRUE,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON COLUMN abac_policies.attribute IS 'TIME_RANGE, IP_WHITELIST, DISTRICT_RESTRICT, DEVICE_TYPE';
COMMENT ON COLUMN abac_policies.value IS 'JSON payload, e.g. {"from":"09:00","to":"20:00"} or {"ips":["10.0.0.0/8"]}';

-- Permission audit log: every decision is logged (including super_admin bypasses).
CREATE TABLE permission_audit_log (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL,
    tenant_id       INT,
    menu_action_id  INT,
    action_name     VARCHAR(80),
    decision        VARCHAR(20) NOT NULL CHECK (decision IN ('GRANTED', 'DENIED', 'OVERRIDE_GRANT', 'OVERRIDE_REVOKE')),
    reason          TEXT,
    ip_address      INET,
    user_agent      TEXT,
    request_id      VARCHAR(64),
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_audit_user ON permission_audit_log(user_id, created_at DESC);
CREATE INDEX idx_audit_tenant ON permission_audit_log(tenant_id, created_at DESC);

-- Session context: tracks which tenant the user is currently viewing/editing.
-- Swappable without re-login (multi-tenant workspace switching).
CREATE TABLE user_session_contexts (
    user_id             BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    active_tenant_id    INT REFERENCES tenants(id),
    active_district_id  INT REFERENCES districts(id),
    updated_at          TIMESTAMPTZ DEFAULT NOW()
);
