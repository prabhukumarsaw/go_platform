-- 002_users_auth.up.sql
-- Unified users table: viewers (OAuth/OTP) + staff (email+password+TOTP).
-- Mirrors the tbl_otp_login pattern from the ULB schema but unified.

CREATE TABLE users (
    id              BIGSERIAL PRIMARY KEY,
    email           VARCHAR(255) UNIQUE,
    phone           VARCHAR(20) UNIQUE,
    password_hash   TEXT,
    display_name    VARCHAR(100),
    avatar_url      TEXT,
    provider        VARCHAR(20) DEFAULT 'email',
    provider_id     VARCHAR(255),
    is_staff        BOOLEAN DEFAULT FALSE,
    is_super_admin  BOOLEAN DEFAULT FALSE,
    totp_secret     TEXT,
    totp_enabled    BOOLEAN DEFAULT FALSE,
    is_active       BOOLEAN DEFAULT TRUE,
    last_login_at   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON COLUMN users.is_super_admin IS 'Bypass flag — checked FIRST before any RBAC/ABAC evaluation. Cannot be set via admin UI.';
COMMENT ON COLUMN users.totp_secret IS 'TOTP secret for MFA. Mandatory for is_staff=true accounts.';
COMMENT ON COLUMN users.provider IS 'Authentication provider: email, google, phone';

CREATE INDEX idx_users_email ON users(email) WHERE email IS NOT NULL;
CREATE INDEX idx_users_phone ON users(phone) WHERE phone IS NOT NULL;
CREATE INDEX idx_users_provider ON users(provider, provider_id);

-- Refresh tokens: one per device, supports rotation and revocation.
CREATE TABLE refresh_tokens (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  TEXT NOT NULL UNIQUE,
    device_info JSONB DEFAULT '{}',
    ip_address  INET,
    is_revoked  BOOLEAN DEFAULT FALSE,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_refresh_tokens_user ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_expires ON refresh_tokens(expires_at) WHERE is_revoked = FALSE;

-- OTP requests: rate-limited, short-lived.
CREATE TABLE otp_requests (
    id          BIGSERIAL PRIMARY KEY,
    phone       VARCHAR(20) NOT NULL,
    otp_hash    TEXT NOT NULL,
    purpose     VARCHAR(20) DEFAULT 'login',
    attempts    INT DEFAULT 0,
    is_used     BOOLEAN DEFAULT FALSE,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_otp_phone ON otp_requests(phone, purpose) WHERE is_used = FALSE;
