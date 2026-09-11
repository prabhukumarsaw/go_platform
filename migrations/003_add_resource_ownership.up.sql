-- 003_add_resource_ownership.up.sql
-- Add resource ownership tracking for proper "own" scope support in IAM
-- This enables proper scoping where users can only access their own resources

-- ──────────────────────────────────────────────
-- 1. Add ownership tracking to articles
-- ──────────────────────────────────────────────
ALTER TABLE articles ADD COLUMN IF NOT EXISTS owner_id BIGINT REFERENCES users(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_articles_owner_id ON articles(owner_id);

-- Update existing articles to set owner_id from author_id
UPDATE articles SET owner_id = author_id WHERE owner_id IS NULL AND author_id IS NOT NULL;

-- ──────────────────────────────────────────────
-- 2. Add ownership tracking to media
-- ──────────────────────────────────────────────
ALTER TABLE media ADD COLUMN IF NOT EXISTS owner_id BIGINT REFERENCES users(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_media_owner_id ON media(owner_id);

-- Update existing media to set owner_id from uploader_id
UPDATE media SET owner_id = uploader_id WHERE owner_id IS NULL AND uploader_id IS NOT NULL;

-- ──────────────────────────────────────────────
-- 3. Add ownership tracking to live blogs
-- ──────────────────────────────────────────────
ALTER TABLE live_blog_entries ADD COLUMN IF NOT EXISTS owner_id BIGINT REFERENCES users(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_live_blog_owner_id ON live_blog_entries(owner_id);

-- Update existing live blog entries to set owner_id from author_id
UPDATE live_blog_entries SET owner_id = author_id WHERE owner_id IS NULL AND author_id IS NOT NULL;

-- ──────────────────────────────────────────────
-- 4. Add ownership tracking to web stories
-- ──────────────────────────────────────────────
ALTER TABLE web_stories ADD COLUMN IF NOT EXISTS owner_id BIGINT REFERENCES users(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_web_stories_owner_id ON web_stories(owner_id);

-- Update existing web stories to set owner_id from author_id
UPDATE web_stories SET owner_id = author_id WHERE owner_id IS NULL AND author_id IS NOT NULL;

-- ──────────────────────────────────────────────
-- 5. Enhanced permission checking functions with ownership support
-- ──────────────────────────────────────────────

-- Enhanced function to check permission with ownership support
CREATE OR REPLACE FUNCTION check_user_permission_with_ownership(
    p_user_id BIGINT,
    p_resource VARCHAR,
    p_action VARCHAR,
    p_scope VARCHAR DEFAULT 'all',
    p_resource_id VARCHAR DEFAULT NULL,
    p_resource_owner_id BIGINT DEFAULT NULL
)
RETURNS BOOLEAN AS $$
DECLARE
    has_perm BOOLEAN;
    is_owner BOOLEAN;
BEGIN
    -- Check if user is the resource owner for "own" scope
    is_owner := (p_resource_owner_id IS NOT NULL AND p_resource_owner_id = p_user_id);
    
    -- If requesting "own" scope and user is not owner, deny
    IF p_scope = 'own' AND NOT is_owner THEN
        RETURN FALSE;
    END IF;
    
    -- If requesting "own" scope and user is owner, check if they have "own" or "all" permission
    IF p_scope = 'own' AND is_owner THEN
        SELECT EXISTS (
            SELECT 1 FROM user_effective_permissions_cache
            WHERE user_id = p_user_id
            AND resource = p_resource
            AND action = p_action
            AND (scope = 'own' OR scope = 'all')
            AND (expires_at IS NULL OR expires_at > NOW())
        ) INTO has_perm;
        
        RETURN has_perm;
    END IF;
    
    -- For "all" scope, check for "all" permission
    IF p_scope = 'all' THEN
        SELECT EXISTS (
            SELECT 1 FROM user_effective_permissions_cache
            WHERE user_id = p_user_id
            AND resource = p_resource
            AND action = p_action
            AND scope = 'all'
            AND (expires_at IS NULL OR expires_at > NOW())
        ) INTO has_perm;
        
        RETURN has_perm;
    END IF;
    
    -- For "department" scope, check department access
    IF p_scope = 'department' THEN
        -- Check if user has department permission or all permission
        SELECT EXISTS (
            SELECT 1 FROM user_effective_permissions_cache
            WHERE user_id = p_user_id
            AND resource = p_resource
            AND action = p_action
            AND (scope = 'department' OR scope = 'all')
            AND (expires_at IS NULL OR expires_at > NOW())
        ) INTO has_perm;
        
        RETURN has_perm;
    END IF;
    
    -- Default: check exact scope match
    SELECT EXISTS (
        SELECT 1 FROM user_effective_permissions_cache
        WHERE user_id = p_user_id
        AND resource = p_resource
        AND action = p_action
        AND scope = p_scope
        AND (expires_at IS NULL OR expires_at > NOW())
    ) INTO has_perm;
    
    RETURN has_perm;
END;
$$ LANGUAGE plpgsql;

-- Function to check resource access with ownership
CREATE OR REPLACE FUNCTION check_resource_access_with_ownership(
    p_user_id BIGINT,
    p_resource_type VARCHAR,
    p_resource_id VARCHAR,
    p_action VARCHAR,
    p_resource_owner_id BIGINT DEFAULT NULL
)
RETURNS BOOLEAN AS $$
DECLARE
    has_access BOOLEAN;
    has_own_perm BOOLEAN;
    has_all_perm BOOLEAN;
BEGIN
    -- Check resource-specific grant first
    SELECT EXISTS (
        SELECT 1 FROM resource_grants rg
        JOIN permissions p ON p.id = rg.permission_id
        WHERE rg.user_id = p_user_id
        AND rg.resource_type = p_resource_type
        AND rg.resource_id = p_resource_id
        AND p.action = p_action
        AND (rg.expires_at IS NULL OR rg.expires_at > NOW())
    ) INTO has_access;
    
    IF has_access THEN
        RETURN TRUE;
    END IF;
    
    -- Check if user is the owner
    IF p_resource_owner_id IS NOT NULL AND p_resource_owner_id = p_user_id THEN
        -- Check if user has "own" permission
        SELECT EXISTS (
            SELECT 1 FROM user_effective_permissions_cache
            WHERE user_id = p_user_id
            AND resource = p_resource_type
            AND action = p_action
            AND (scope = 'own' OR scope = 'all')
            AND (expires_at IS NULL OR expires_at > NOW())
        ) INTO has_own_perm;
        
        IF has_own_perm THEN
            RETURN TRUE;
        END IF;
    END IF;
    
    -- Check if user has "all" permission
    SELECT EXISTS (
        SELECT 1 FROM user_effective_permissions_cache
        WHERE user_id = p_user_id
        AND resource = p_resource_type
        AND action = p_action
        AND scope = 'all'
        AND (expires_at IS NULL OR expires_at > NOW())
    ) INTO has_all_perm;
    
    RETURN has_all_perm;
END;
$$ LANGUAGE plpgsql;

-- ──────────────────────────────────────────────
-- 6. Add helper function to get resource owner
-- ──────────────────────────────────────────────
CREATE OR REPLACE FUNCTION get_resource_owner(
    p_resource_type VARCHAR,
    p_resource_id VARCHAR
)
RETURNS BIGINT AS $$
DECLARE
    owner_id BIGINT;
BEGIN
    CASE p_resource_type
        WHEN 'articles' THEN
            SELECT owner_id INTO owner_id FROM articles WHERE id::text = p_resource_id;
        WHEN 'media' THEN
            SELECT owner_id INTO owner_id FROM media WHERE id::text = p_resource_id;
        WHEN 'live_blog_entries' THEN
            SELECT owner_id INTO owner_id FROM live_blog_entries WHERE id::text = p_resource_id;
        WHEN 'web_stories' THEN
            SELECT owner_id INTO owner_id FROM web_stories WHERE id::text = p_resource_id;
        ELSE
            owner_id := NULL;
    END CASE;
    
    RETURN owner_id;
END;
$$ LANGUAGE plpgsql;

-- ──────────────────────────────────────────────
-- 7. Add triggers to maintain ownership on insert/update
-- ──────────────────────────────────────────────

-- Articles ownership trigger
CREATE OR REPLACE FUNCTION maintain_article_ownership()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.owner_id IS NULL AND NEW.author_id IS NOT NULL THEN
        NEW.owner_id := NEW.author_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_maintain_article_ownership
    BEFORE INSERT OR UPDATE ON articles
    FOR EACH ROW
    EXECUTE FUNCTION maintain_article_ownership();

-- Media ownership trigger
CREATE OR REPLACE FUNCTION maintain_media_ownership()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.owner_id IS NULL AND NEW.uploader_id IS NOT NULL THEN
        NEW.owner_id := NEW.uploader_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_maintain_media_ownership
    BEFORE INSERT OR UPDATE ON media
    FOR EACH ROW
    EXECUTE FUNCTION maintain_media_ownership();

-- Live blog ownership trigger
CREATE OR REPLACE FUNCTION maintain_live_blog_ownership()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.owner_id IS NULL AND NEW.author_id IS NOT NULL THEN
        NEW.owner_id := NEW.author_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_maintain_live_blog_ownership
    BEFORE INSERT OR UPDATE ON live_blog_entries
    FOR EACH ROW
    EXECUTE FUNCTION maintain_live_blog_ownership();

-- Web stories ownership trigger
CREATE OR REPLACE FUNCTION maintain_web_story_ownership()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.owner_id IS NULL AND NEW.author_id IS NOT NULL THEN
        NEW.owner_id := NEW.author_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_maintain_web_story_ownership
    BEFORE INSERT OR UPDATE ON web_stories
    FOR EACH ROW
    EXECUTE FUNCTION maintain_web_story_ownership();

-- ──────────────────────────────────────────────
-- 8. Refresh permission cache to apply changes
-- ──────────────────────────────────────────────
SELECT refresh_user_permissions_cache();