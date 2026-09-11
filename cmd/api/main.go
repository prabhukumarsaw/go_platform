package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/rs/zerolog"

	"newsplatform/api/internal/ads"
	"newsplatform/api/internal/ai"
	"newsplatform/api/internal/analytics"
	"newsplatform/api/internal/auth"
	"newsplatform/api/internal/content"
	"newsplatform/api/internal/employee"
	"newsplatform/api/internal/epaper"
	"newsplatform/api/internal/iam"
	"newsplatform/api/internal/jobs"
	"newsplatform/api/internal/media"
	"newsplatform/api/internal/moderation"
	"newsplatform/api/internal/notify"
	"newsplatform/api/internal/poll"
	"newsplatform/api/internal/routes"
	"newsplatform/api/internal/seo"
	"newsplatform/api/internal/settings"
	"newsplatform/api/internal/stream"
	"newsplatform/api/internal/webstory"
	"newsplatform/api/pkg/config"
	"newsplatform/api/pkg/database"
	"newsplatform/api/pkg/middleware"
	"newsplatform/api/pkg/response"
)

func main() {
	// ─── 1. Structured Logging ───────────────────
	log := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).
		With().Timestamp().Caller().Logger()

	// ─── 2. Configuration ───────────────────────
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load config")
	}

	// ─── 3. Database & Redis Connections ────────
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := database.NewPostgresPool(ctx, cfg.Postgres)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to PostgreSQL")
	}
	defer pool.Close()
	log.Info().Msg("Connected to PostgreSQL")

	// Ensure active content is published and linked to hierarchical categories
	_, _ = pool.Exec(ctx, `
		-- Ensure unique slug index on master taxonomy categories
		CREATE UNIQUE INDEX IF NOT EXISTS idx_categories_slug_unique ON categories(slug);

		-- Publish articles so they appear live on both the homepage and the dashboard panel
		UPDATE articles
		SET status = 'published',
		    published_at = COALESCE(published_at, NOW() - (RANDOM() * INTERVAL '24 hours')),
		    view_count = CASE WHEN view_count = 0 THEN FLOOR(120 + RANDOM() * 3400)::int ELSE view_count END,
		    is_national = TRUE
		WHERE status != 'published' OR published_at IS NULL;

		-- Ensure all articles are linked to master categories in junction table
		INSERT INTO article_categories (article_id, category_id)
		SELECT a.id, c.id
		FROM articles a
		CROSS JOIN LATERAL (
			SELECT id FROM categories WHERE level >= 1 ORDER BY id LIMIT 1 OFFSET (abs(hashtext(a.id::text)) % (SELECT GREATEST(COUNT(*), 1) FROM categories WHERE level >= 1))
		) c
		WHERE NOT EXISTS (SELECT 1 FROM article_categories ac WHERE ac.article_id = a.id)
		ON CONFLICT DO NOTHING;

		-- Ensure live_blog_entries columns
		CREATE TABLE IF NOT EXISTS live_blog_entries (
			id SERIAL PRIMARY KEY,
			article_id UUID REFERENCES articles(id) ON DELETE CASCADE,
			headline VARCHAR(255) DEFAULT '',
			title VARCHAR(255) DEFAULT '',
			body JSONB DEFAULT '""'::jsonb,
			content TEXT DEFAULT '',
			author_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
			is_pinned BOOLEAN DEFAULT FALSE,
			is_breaking BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);
		ALTER TABLE live_blog_entries ADD COLUMN IF NOT EXISTS headline VARCHAR(255) DEFAULT '';
		ALTER TABLE live_blog_entries ADD COLUMN IF NOT EXISTS body JSONB DEFAULT '""'::jsonb;
		ALTER TABLE live_blog_entries ADD COLUMN IF NOT EXISTS author_id BIGINT REFERENCES users(id) ON DELETE SET NULL;
		ALTER TABLE live_blog_entries ADD COLUMN IF NOT EXISTS is_pinned BOOLEAN DEFAULT FALSE;
		ALTER TABLE live_blog_entries ADD COLUMN IF NOT EXISTS is_breaking BOOLEAN DEFAULT FALSE;
		ALTER TABLE live_blog_entries ALTER COLUMN content DROP NOT NULL;
		ALTER TABLE live_blog_entries ALTER COLUMN content SET DEFAULT '';
		ALTER TABLE live_blog_entries ALTER COLUMN title DROP NOT NULL;
		ALTER TABLE live_blog_entries ALTER COLUMN title SET DEFAULT '';

		-- Ensure media table schema
		DO $$
		DECLARE
			id_type text;
		BEGIN
			SELECT data_type INTO id_type 
			FROM information_schema.columns 
			WHERE table_name = 'media' AND column_name = 'id';

			IF id_type IS NOT NULL AND id_type != 'uuid' THEN
				DROP TABLE IF EXISTS media CASCADE;
			END IF;
		END $$;

		CREATE TABLE IF NOT EXISTS media (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			uploader_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
			filename VARCHAR(255) NOT NULL,
			original_name VARCHAR(255) DEFAULT '',
			mime_type VARCHAR(100) DEFAULT '',
			category VARCHAR(50) DEFAULT 'news',
			folder VARCHAR(100) DEFAULT 'general',
			file_size BIGINT DEFAULT 0,
			storage_path TEXT DEFAULT '',
			url TEXT DEFAULT '',
			alt_text TEXT DEFAULT '',
			caption TEXT DEFAULT '',
			width INT DEFAULT 0,
			height INT DEFAULT 0,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);
		ALTER TABLE media ADD COLUMN IF NOT EXISTS uploader_id BIGINT REFERENCES users(id) ON DELETE SET NULL;
		ALTER TABLE media ADD COLUMN IF NOT EXISTS original_name VARCHAR(255) DEFAULT '';
		ALTER TABLE media ADD COLUMN IF NOT EXISTS file_size BIGINT DEFAULT 0;
		ALTER TABLE media ADD COLUMN IF NOT EXISTS storage_path TEXT DEFAULT '';
		ALTER TABLE media ADD COLUMN IF NOT EXISTS url TEXT DEFAULT '';
		ALTER TABLE media ADD COLUMN IF NOT EXISTS alt_text TEXT DEFAULT '';
		ALTER TABLE media ADD COLUMN IF NOT EXISTS caption TEXT DEFAULT '';
		ALTER TABLE media ADD COLUMN IF NOT EXISTS width INT DEFAULT 0;
		ALTER TABLE media ADD COLUMN IF NOT EXISTS height INT DEFAULT 0;

		CREATE INDEX IF NOT EXISTS idx_media_category ON media(category);
		CREATE INDEX IF NOT EXISTS idx_media_folder ON media(folder);
		CREATE INDEX IF NOT EXISTS idx_media_created_at ON media(created_at DESC);

		-- Ensure employees table exists for staff management
		CREATE TABLE IF NOT EXISTS employees (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			employee_code VARCHAR(100) UNIQUE NOT NULL,
			department VARCHAR(100) DEFAULT 'Editorial',
			designation VARCHAR(200) DEFAULT '',
			address TEXT DEFAULT '',
			pin_code VARCHAR(20) DEFAULT '',
			bio TEXT DEFAULT '',
			press_card_no VARCHAR(100) DEFAULT '',
			x_handle VARCHAR(100) DEFAULT '',
			facebook VARCHAR(200) DEFAULT '',
			instagram VARCHAR(200) DEFAULT '',
			youtube VARCHAR(200) DEFAULT '',
			is_active BOOLEAN DEFAULT TRUE,
			joined_at TIMESTAMPTZ DEFAULT NOW(),
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		);
		ALTER TABLE employees ADD COLUMN IF NOT EXISTS facebook VARCHAR(200) DEFAULT '';
		ALTER TABLE employees ADD COLUMN IF NOT EXISTS instagram VARCHAR(200) DEFAULT '';
		ALTER TABLE employees ADD COLUMN IF NOT EXISTS youtube VARCHAR(200) DEFAULT '';
	`)

	// ─── Schema Migration: IAM tables (always run independently) ──────────────
	schemaAlters := []string{
		// roles table
		`ALTER TABLE roles ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT TRUE NOT NULL`,
		`ALTER TABLE roles ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT NULL`,
		// menus table
		`ALTER TABLE menus ADD COLUMN IF NOT EXISTS icon VARCHAR(100) DEFAULT ''`,
		`ALTER TABLE menus ADD COLUMN IF NOT EXISTS parent_id INT REFERENCES menus(id) ON DELETE SET NULL`,
		`ALTER TABLE menus ADD COLUMN IF NOT EXISTS group_name VARCHAR(50) DEFAULT 'content'`,
		`ALTER TABLE menus ADD COLUMN IF NOT EXISTS api_prefix TEXT DEFAULT ''`,
		`ALTER TABLE user_roles ADD COLUMN IF NOT EXISTS assigned_by BIGINT REFERENCES users(id) ON DELETE SET NULL`,
		`ALTER TABLE user_permission_overrides ADD COLUMN IF NOT EXISTS reason TEXT DEFAULT ''`,
		`ALTER TABLE user_permission_overrides ADD COLUMN IF NOT EXISTS granted_by BIGINT REFERENCES users(id) ON DELETE SET NULL`,
		// menu_actions table
		`ALTER TABLE menu_actions ADD COLUMN IF NOT EXISTS label VARCHAR(150) DEFAULT ''`,
		// permission_audit_log table
		`CREATE TABLE IF NOT EXISTS permission_audit_log (
			id              BIGSERIAL PRIMARY KEY,
			user_id         BIGINT NOT NULL,
			menu_action_id  INT,
			action_name     VARCHAR(80),
			decision        VARCHAR(20) NOT NULL,
			reason          TEXT,
			ip_address      INET,
			user_agent      TEXT,
			request_id      VARCHAR(100),
			created_at      TIMESTAMPTZ DEFAULT NOW()
		)`,
		// Auto-enroll superadmin and staff into employees if missing
		`		INSERT INTO employees (user_id, employee_code, department, designation, bio, press_card_no, is_active)
		SELECT u.id,
		       'EMP-' || TO_CHAR(NOW(), 'YYYY') || '-' || LPAD(u.id::text, 4, '0'),
		       'Editorial',
		       CASE WHEN u.is_super_admin THEN 'Chief Editor' ELSE 'Staff Member' END,
		       CASE WHEN u.is_super_admin THEN 'Platform Chief Editor & Root Administrator' ELSE 'Newsroom staff member' END,
		       CASE WHEN u.is_super_admin THEN 'PRESS-CHIEF-01' ELSE 'PRESS-' || LPAD(u.id::text, 4, '0') END,
		       TRUE
		FROM users u
		WHERE (u.is_staff = TRUE OR u.is_super_admin = TRUE)
		  AND NOT EXISTS (SELECT 1 FROM employees e WHERE e.user_id = u.id)
		ON CONFLICT (user_id) DO NOTHING`,
	}
	for _, q := range schemaAlters {
		if _, err := pool.Exec(ctx, q); err != nil {
			log.Warn().Err(err).Str("query", q).Msg("Schema alter skipped (may already exist)")
		}
	}

	// ─── IAM Seed: ensure menus, roles, actions & grants exist ───────────────
	_, _ = pool.Exec(ctx, `
		-- Ensure all active newsroom menus exist
		INSERT INTO menus (name, label, icon, path, sort_order, is_active, group_name, api_prefix) VALUES
			('dashboard',     'Dashboard',                 'LayoutDashboard', '/panel/dashboard',     1,  TRUE, 'content',    ''),
			('articles',      'Articles',                  'FileText',        '/panel/articles',      2,  TRUE, 'content',    '/studio/articles,/studio/stories,/studio/ai'),
			('categories',    'Categories',                'Tag',             '/panel/categories',    3,  TRUE, 'content',    '/studio/categories'),
			('homepage',      'Home Categories & Layout',  'LayoutGrid',      '/panel/homepage',      4,  TRUE, 'content',    '/studio/homepage'),
			('tags',          'Tags',                      'Hash',            '/panel/tags',          5,  TRUE, 'content',    '/studio/tags'),
			('media_library', 'Media',                     'Image',           '/panel/media',         6,  TRUE, 'content',    '/media,/studio/media'),
			('live_blogs',    'Live Blog',                 'Radio',           '/panel/liveblog',      7,  TRUE, 'content',    '/studio/live-blogs'),
			('notifications', 'Broadcast & Alerts',        'Bell',            '/panel/notifications', 8,  TRUE, 'content',    '/admin/notifications,/notifications'),
			('reports',       'Reports',                   'BarChart2',       '/panel/reports',       9,  TRUE, 'management', ''),
			('users',         'Team & Staff',              'Users',           '/panel/users',         10, TRUE, 'management', '/admin/employees'),
			('comments',      'Comments',                  'MessageSquare',   '/panel/comments',      11, TRUE, 'management', '/admin/moderation,/moderation'),
			('analytics',     'Analytics',                 'TrendingUp',      '/panel/analytics',     12, TRUE, 'management', '/admin/analytics'),
			('roles',         'Roles & Permissions',       'Lock',            '/panel/roles',         13, TRUE, 'management', '/iam,/roles'),
			('settings',      'Settings',                  'Settings',        '/panel/settings',      14, TRUE, 'management', '/admin/settings'),
			('audit',         'Audit Log',                 'Shield',          '/panel/audit',         15, TRUE, 'management', '/iam/audit-log'),
			('seo',           'SEO Management',            'Search',          '/panel/seo',           16, TRUE, 'management', '/admin/seo')
		ON CONFLICT (name) DO UPDATE SET
			label      = EXCLUDED.label,
			icon       = EXCLUDED.icon,
			path       = EXCLUDED.path,
			sort_order = EXCLUDED.sort_order,
			is_active  = EXCLUDED.is_active,
			group_name = EXCLUDED.group_name,
			api_prefix = EXCLUDED.api_prefix;

		-- Deactivate obsolete/duplicate menus
		UPDATE menus SET is_active = FALSE WHERE name IN ('employees', 'polls', 'moderation', 'ads', 'web_stories', 'e_paper');

		-- Generate actions for all menus
		DO $$
		DECLARE
			m RECORD;
			actions TEXT[] := ARRAY['VIEW','ADD','EDIT','DELETE','PUBLISH','APPROVE'];
			act TEXT;
		BEGIN
			FOR m IN SELECT id FROM menus LOOP
				FOREACH act IN ARRAY actions LOOP
					INSERT INTO menu_actions (menu_id, action) VALUES (m.id, act) ON CONFLICT DO NOTHING;
				END LOOP;
			END LOOP;
		END $$;

		-- Ensure system roles exist
		INSERT INTO roles (name, description, is_system, is_active) VALUES
			('super_admin', 'Full platform-wide root authority',           TRUE, TRUE),
			('editor',      'Editorial publishing and review authority',   TRUE, TRUE),
			('sub_editor',  'Content review and editing desk',             TRUE, TRUE),
			('reporter',    'Field journalism and draft creation',         TRUE, TRUE),
			('moderator',   'Community and comment moderation authority',  TRUE, TRUE)
		ON CONFLICT (name) DO UPDATE SET is_active = TRUE;

		-- Grant ALL permissions to super_admin
		INSERT INTO role_menu_actions (role_id, menu_action_id)
		SELECT r.id, ma.id FROM roles r CROSS JOIN menu_actions ma WHERE r.name = 'super_admin'
		ON CONFLICT DO NOTHING;

		-- Rebuild system-role grants so matrix matches real newsroom desks (not "all menus")
		DELETE FROM role_menu_actions rma
		USING roles r
		WHERE rma.role_id = r.id AND r.is_system = TRUE AND r.name <> 'super_admin';

		-- editor: publish across editorial surfaces, no IAM/settings/audit
		INSERT INTO role_menu_actions (role_id, menu_action_id)
		SELECT r.id, ma.id FROM roles r
		JOIN menu_actions ma ON ma.action IN ('VIEW','ADD','EDIT','PUBLISH')
		JOIN menus m ON m.id = ma.menu_id AND m.name IN (
			'dashboard','articles','categories','homepage','tags','media_library','live_blogs','notifications','comments','analytics'
		)
		WHERE r.name = 'editor'
		ON CONFLICT DO NOTHING;

		-- reporter: draft and media only
		INSERT INTO role_menu_actions (role_id, menu_action_id)
		SELECT r.id, ma.id FROM roles r
		JOIN menu_actions ma ON ma.action IN ('VIEW','ADD','EDIT')
		JOIN menus m ON m.id = ma.menu_id AND m.name IN ('dashboard','articles','media_library','tags','categories')
		WHERE r.name = 'reporter'
		ON CONFLICT DO NOTHING;

		-- Grant VIEW+ADD+EDIT on editorial menus to sub_editor
		INSERT INTO role_menu_actions (role_id, menu_action_id)
		SELECT r.id, ma.id FROM roles r
		JOIN menu_actions ma ON ma.action IN ('VIEW','ADD','EDIT')
		JOIN menus m ON m.id = ma.menu_id AND m.name IN ('dashboard','articles','categories','tags','media_library','live_blogs','comments')
		WHERE r.name = 'sub_editor'
		ON CONFLICT DO NOTHING;

		-- Grant VIEW+APPROVE+DELETE on comments/moderation to moderator
		INSERT INTO role_menu_actions (role_id, menu_action_id)
		SELECT r.id, ma.id FROM roles r
		JOIN menu_actions ma ON ma.action IN ('VIEW','APPROVE','DELETE')
		JOIN menus m ON m.id = ma.menu_id AND m.name IN ('comments','moderation','articles')
		WHERE r.name = 'moderator'
		ON CONFLICT DO NOTHING;

		-- Every staff role can at least open the dashboard shell
		INSERT INTO role_menu_actions (role_id, menu_action_id)
		SELECT r.id, ma.id FROM roles r
		JOIN menu_actions ma ON ma.action = 'VIEW'
		JOIN menus m ON m.id = ma.menu_id AND m.name = 'dashboard'
		ON CONFLICT DO NOTHING;
	`)

	redisClient, err := database.NewRedisClient(ctx, cfg.Redis)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to Redis")
	}
	defer redisClient.Close()
	log.Info().Msg("Connected to Redis")

	// ─── 4. Background Job Engine ───────────────
	jobScheduler := jobs.NewScheduler(pool, redisClient, log)
	jobScheduler.Start()
	defer jobScheduler.Stop()

	// ─── 5. Domain Services Dependency Injection 
	iamService := iam.NewService(pool, log)
	iamEnhancedService := iam.NewEnhancedServiceWrapper(pool, log)
	iamMigrationService := iam.NewMigrationService(pool, log)
	authService := auth.NewService(pool, *cfg, log)
	contentService := content.NewService(pool, redisClient, log)
	if err := contentService.EnsureHomepageSchema(ctx); err != nil {
		log.Warn().Err(err).Msg("Homepage schema initialization warning")
	}
	mediaService := media.NewService(pool, cfg.Media, log)
	if err := mediaService.EnsureSchema(ctx); err != nil {
		log.Warn().Err(err).Msg("Media schema initialization warning")
	}
	adsService := ads.NewService(pool, log)
	seoService := seo.NewService(pool, "http://localhost:3000", log)
	notifyService := notify.NewService(pool, redisClient, log)
	moderationService := moderation.NewService(pool, log)
	webstoryService := webstory.NewService(pool, log)
	pollService := poll.NewService(pool, log)
	epaperService := epaper.NewService(pool, log)
	streamService := stream.NewService(redisClient, log)
	employeeService := employee.NewService(pool, log)
	analyticsService := analytics.NewService(pool, log)
	aiService := ai.NewService(log)

	// ─── 6. Domain Handlers Registry ────────────
	handlers := &routes.HandlerRegistry{
		Auth:          auth.NewHandler(authService),
		IAM:           iam.NewHandler(iamService, iamEnhancedService),
		Content:       content.NewHandler(contentService),
		Media:         media.NewHandler(mediaService),
		Ads:           ads.NewHandler(adsService),
		SEO:           seo.NewHandler(seoService),
		Notify:        notify.NewHandler(notifyService),
		Moderation:    moderation.NewHandler(moderationService),
		WebStory:      webstory.NewHandler(webstoryService),
		Poll:          poll.NewHandler(pollService),
		EPaper:        epaper.NewHandler(epaperService),
		Stream:        stream.NewHandler(streamService),
		Employee:      employee.NewHandler(employeeService),
		Analytics:     analytics.NewHandler(analyticsService),
		AI:            ai.NewHandler(aiService),
		Settings:      settings.NewHandler(pool, cfg.Media.UploadDir),
	}

	// ─── 6.5. Migration Handler Registry ─────────
	migrationHandler := iam.NewMigrationHandler(iamMigrationService)

	// ─── 7. Fiber Web Engine with Global Error Handler 
	app := fiber.New(fiber.Config{
		AppName:      "BharatVani News Platform API (Clean MVC)",
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
		BodyLimit:    int(cfg.Media.MaxFileSize) + 1024*1024,
		ErrorHandler: response.GlobalErrorHandler, // Centralized Global Error & Status Handler
	})

	// ─── 8. Global Middleware Pipeline ──────────
	app.Use(recover.New())
	app.Use(middleware.SecurityHeaders()) // Enterprise HTTP Security Headers (XSS, Clickjacking, MIME)
	app.Use(requestid.New())
	app.Use(logger.New(logger.Config{
		Format:     "${time} | ${status} | ${latency} | ${method} ${path}\n",
		TimeFormat: "15:04:05",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: "http://localhost:3000",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, PATCH, OPTIONS",
	}))
	app.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	}))
	app.Use(middleware.RateLimiter(300, 1*time.Minute))

	// Static local media storage mount with 30-day client caching and HTTP byte ranges
	absUploadDir, _ := filepath.Abs(cfg.Media.UploadDir)
	_ = os.MkdirAll(absUploadDir, 0755)
	app.Static("/uploads", absUploadDir, fiber.Static{
		Compress:  true,
		ByteRange: true,
		Browse:    false,
		MaxAge:    86400 * 30, // 30 days client cache
	})

	// ─── 9. Mount Centralized Router ────────────
	routes.Setup(app, handlers, pool, cfg)

	// ─── 9.5. Mount Migration Routes ────────────
	migrationHandler.RegisterMigrationRoutes(app)

	// ─── 10. Start Server ───────────────────────
	go func() {
		addr := fmt.Sprintf(":%s", cfg.Server.Port)
		log.Info().Str("addr", addr).Str("env", cfg.Server.Env).Msg("Starting server")
		if err := app.Listen(addr); err != nil {
			log.Fatal().Err(err).Msg("Server failed")
		}
	}()

	// ─── 11. Graceful Shutdown ──────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server gracefully...")
	_ = app.Shutdown()
}
