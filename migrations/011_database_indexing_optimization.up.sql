-- 011_database_indexing_optimization.up.sql
-- Enterprise High-Performance Indexes for Foreign Keys, Filter Matrix, and Composite Queries

-- ─── 1. Content & Articles Indexing ───────────────────────────
CREATE INDEX IF NOT EXISTS idx_articles_slug_lang_status 
    ON articles(slug, language, status);

CREATE INDEX IF NOT EXISTS idx_articles_tenant_status_date 
    ON articles(tenant_id, status, published_at DESC NULLS LAST);

CREATE INDEX IF NOT EXISTS idx_articles_breaking 
    ON articles(tenant_id, is_breaking, status, published_at DESC NULLS LAST);

CREATE INDEX IF NOT EXISTS idx_articles_featured 
    ON articles(tenant_id, is_featured, status, published_at DESC NULLS LAST);

CREATE INDEX IF NOT EXISTS idx_articles_trending 
    ON articles(tenant_id, status, view_count DESC, published_at DESC NULLS LAST);

CREATE INDEX IF NOT EXISTS idx_articles_author 
    ON articles(author_id, status);

CREATE INDEX IF NOT EXISTS idx_articles_district 
    ON articles(district_id, status, published_at DESC NULLS LAST);

CREATE INDEX IF NOT EXISTS idx_articles_national 
    ON articles(is_national, status, published_at DESC NULLS LAST);

-- ─── 2. Taxonomy & Junction Tables ────────────────────────────
CREATE INDEX IF NOT EXISTS idx_article_categories_category 
    ON article_categories(category_id, article_id);

CREATE INDEX IF NOT EXISTS idx_article_tags_tag 
    ON article_tags(tag_id, article_id);

-- ─── 3. IAM & Dynamic Permission Evaluator ────────────────────
CREATE INDEX IF NOT EXISTS idx_utm_user_tenant_active 
    ON user_tenant_mappings(user_id, tenant_id, is_active);

CREATE INDEX IF NOT EXISTS idx_rma_role_tenant_action 
    ON role_menu_actions(role_id, tenant_id, menu_action_id);

CREATE INDEX IF NOT EXISTS idx_upo_user_eval 
    ON user_permission_overrides(user_id, tenant_id, menu_action_id, effect);

CREATE INDEX IF NOT EXISTS idx_audit_log_tenant_time 
    ON permission_audit_log(tenant_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_abac_user_tenant 
    ON abac_policies(user_id, tenant_id, is_active);

-- ─── 4. Media, Web Stories, Polls & E-Papers ──────────────────
CREATE INDEX IF NOT EXISTS idx_web_stories_tenant_status_date 
    ON web_stories(tenant_id, status, published_at DESC NULLS LAST);

CREATE INDEX IF NOT EXISTS idx_polls_tenant_active 
    ON polls(tenant_id, is_active);

CREATE INDEX IF NOT EXISTS idx_poll_votes_dedup_ip 
    ON poll_votes(poll_id, ip_address);

CREATE INDEX IF NOT EXISTS idx_poll_votes_dedup_user 
    ON poll_votes(poll_id, user_id);

CREATE INDEX IF NOT EXISTS idx_epapers_tenant_date 
    ON epapers(tenant_id, edition_date DESC);

CREATE INDEX IF NOT EXISTS idx_media_tenant_date 
    ON media(tenant_id, created_at DESC);

-- ─── 5. Comments, Feedback, Auth & Security ───────────────────
CREATE INDEX IF NOT EXISTS idx_comments_article_status_date 
    ON comments(article_id, status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_feedback_tenant_status_date 
    ON feedbacks(tenant_id, status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_user_activity_user_date 
    ON user_activity_logs(user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_expiry 
    ON refresh_tokens(user_id, expires_at);

CREATE INDEX IF NOT EXISTS idx_reset_tokens_hash_expiry 
    ON password_reset_tokens(token_hash, expires_at);

CREATE INDEX IF NOT EXISTS idx_newsletter_tenant_active 
    ON newsletter_subscriptions(tenant_id, is_active);
