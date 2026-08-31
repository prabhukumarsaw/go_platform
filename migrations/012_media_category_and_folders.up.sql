-- 012_media_category_and_folders.up.sql
-- Media category classifications (news, ads, custom, avatars) and folder management

ALTER TABLE media 
    ADD COLUMN IF NOT EXISTS category VARCHAR(50) NOT NULL DEFAULT 'news',
    ADD COLUMN IF NOT EXISTS folder VARCHAR(100) DEFAULT 'general';

CREATE INDEX IF NOT EXISTS idx_media_category 
    ON media(tenant_id, category, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_media_mime 
    ON media(tenant_id, mime_type, created_at DESC);
