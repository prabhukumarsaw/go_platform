-- 010_auth_security_feedback.up.sql
-- Password Resets, User Activity Audit, and Reader Feedback

CREATE TABLE IF NOT EXISTS password_reset_tokens (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  TEXT NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    used_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_activity_logs (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event       VARCHAR(50) NOT NULL, -- login, logout, password_change, otp_request, totp_enabled
    ip_address  VARCHAR(50),
    user_agent  TEXT,
    metadata    JSONB DEFAULT '{}',
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_activity_user ON user_activity_logs(user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS feedbacks (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT REFERENCES users(id) ON DELETE SET NULL,
    tenant_id   INT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name        VARCHAR(100) NOT NULL,
    email       VARCHAR(255) NOT NULL,
    category    VARCHAR(50) NOT NULL DEFAULT 'general', -- general, fact_check_correction, technical, editorial
    article_id  UUID REFERENCES articles(id) ON DELETE SET NULL,
    message     TEXT NOT NULL,
    status      VARCHAR(20) NOT NULL DEFAULT 'open', -- open, reviewed, resolved
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_feedback_tenant ON feedbacks(tenant_id, status);
