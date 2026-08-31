-- 014_push_notifications.up.sql
-- Web Push Subscriptions for Regional / Hyperlocal Breaking Alerts (RFC 8292 VAPID)

CREATE TABLE IF NOT EXISTS push_subscriptions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       INT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    district_id     INT REFERENCES districts(id) ON DELETE SET NULL,
    user_id         BIGINT REFERENCES users(id) ON DELETE SET NULL,
    endpoint        TEXT NOT NULL UNIQUE,
    p256dh_key      TEXT NOT NULL,
    auth_key        TEXT NOT NULL,
    user_agent      TEXT,
    is_active       BOOLEAN DEFAULT TRUE,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_push_tenant_district ON push_subscriptions(tenant_id, district_id) WHERE is_active = TRUE;
