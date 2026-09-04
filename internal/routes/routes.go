package routes

import (
	// Third-Party Core Frameworks
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	// Internal Domain Handlers - Identity & Access Management
	"newsplatform/api/internal/auth"
	"newsplatform/api/internal/employee"
	"newsplatform/api/internal/iam"

	// Internal Domain Handlers - Content & Editorial Newsroom Suite
	"newsplatform/api/internal/ai"
	"newsplatform/api/internal/content"
	"newsplatform/api/internal/epaper"
	"newsplatform/api/internal/media"
	"newsplatform/api/internal/webstory"

	// Internal Domain Handlers - Engagement, Real-Time & Interactivity
	"newsplatform/api/internal/moderation"
	"newsplatform/api/internal/notify"
	"newsplatform/api/internal/poll"
	"newsplatform/api/internal/stream"

	// Internal Domain Handlers - Revenue, Discovery, Settings & Analytics
	"newsplatform/api/internal/ads"
	"newsplatform/api/internal/analytics"
	"newsplatform/api/internal/seo"
	"newsplatform/api/internal/settings"

	// Shared Infrastructure & Middleware
	"newsplatform/api/pkg/config"
	"newsplatform/api/pkg/middleware"
)

// HandlerRegistry aggregates domain HTTP handlers for centralized route mounting.
type HandlerRegistry struct {
	// Identity, Access & Staff Governance
	Auth     *auth.Handler
	IAM      *iam.Handler
	Employee *employee.Handler

	// Editorial CMS & Multimedia Suite
	Content  *content.Handler
	Media    *media.Handler
	WebStory *webstory.Handler
	EPaper   *epaper.Handler
	AI       *ai.Handler

	// Audience Engagement & Moderation
	Poll       *poll.Handler
	Moderation *moderation.Handler
	Stream     *stream.Handler
	Notify     *notify.Handler

	// Monetization, Discovery, Settings & Analytics
	Ads       *ads.Handler
	SEO       *seo.Handler
	Analytics *analytics.Handler
	Settings  *settings.Handler
}

// Setup builds and attaches all API route groups to Fiber.
func Setup(app *fiber.App, h *HandlerRegistry, pool *pgxpool.Pool, cfg *config.Config) {
	// Root health check & deep readiness probe
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"version": "0.1.0",
		})
	})
	app.Get("/health/ready", func(c *fiber.Ctx) error {
		if err := pool.Ping(c.Context()); err != nil {
			return c.Status(503).JSON(fiber.Map{"status": "degraded", "database": "disconnected"})
		}
		return c.JSON(fiber.Map{"status": "ready", "database": "connected"})
	})

	api := app.Group("/api/v1")

	// ─── 1. Public Authentication Routes ────────────
	h.Auth.RegisterRoutes(api, cfg.JWT)

	// ─── 2. Public Reader & Audience Routes ─────────
	// Edge Caching: s-maxage=60s on Cloudflare/CDN edge, stale-while-revalidate=300s
	publicRoutes := api.Group("",
		middleware.PublicTransactionContext(pool),
		middleware.EdgeCache(60, 300),
		middleware.OptionalAuth(cfg.JWT),
	)

	h.Content.RegisterPublicRoutes(publicRoutes)
	h.Notify.RegisterPublicRoutes(publicRoutes)
	h.Moderation.RegisterPublicRoutes(publicRoutes)
	h.WebStory.RegisterPublicRoutes(publicRoutes)
	h.Poll.RegisterPublicRoutes(publicRoutes)
	h.EPaper.RegisterPublicRoutes(publicRoutes)
	h.Stream.RegisterPublicRoutes(publicRoutes)

	// Public Regional Bureaus list (uses master hierarchical taxonomy)
	api.Get("/regions", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"success": true,
			"data": []fiber.Map{
				{"id": 1, "name": "National Desk", "slug": "national", "is_national": true},
			},
		})
	})

	// Public Menus route for dynamic navigation
	api.Get("/menus", h.IAM.ListMenus)
	api.Get("/iam/menus", h.IAM.ListMenus)

	// Public Settings (Platform identity, social channels, maintenance status)
	if h.Settings != nil {
		h.Settings.RegisterPublicRoutes(api)
	}

	// ─── 3. Staff Studio Routes (Newsroom CMS) ──────
	authRequired := api.Group("", middleware.RequireAuth(cfg.JWT))
	staffRoutes := authRequired.Group("", middleware.RequireStaff(), middleware.TransactionContext(pool))

	staffRoutes.Get("/menus", h.IAM.ListMenus)
	staffRoutes.Get("/iam/menus", h.IAM.ListMenus)
	h.IAM.RegisterRoutes(staffRoutes)

	h.Content.RegisterStudioRoutes(staffRoutes)
	h.Media.RegisterRoutes(staffRoutes)
	h.WebStory.RegisterStudioRoutes(staffRoutes)
	h.EPaper.RegisterStudioRoutes(staffRoutes)
	h.AI.RegisterStudioRoutes(staffRoutes)

	// ─── 4. Admin & Governance Routes ───────────────
	adminRoutes := staffRoutes.Group("/admin")
	h.IAM.RegisterRoutes(adminRoutes)
	h.Ads.RegisterRoutes(adminRoutes)
	h.Moderation.RegisterAdminRoutes(adminRoutes)
	h.Employee.RegisterAdminRoutes(adminRoutes)
	h.Analytics.RegisterAdminRoutes(adminRoutes)
	h.Notify.RegisterAdminRoutes(adminRoutes)
	if h.Settings != nil {
		h.Settings.RegisterAdminRoutes(adminRoutes)
	}

	// ─── 5. SEO Sitemaps & RSS Feeds ───────────────
	h.SEO.RegisterPublicRoutes(app)
	h.SEO.RegisterPublicRoutes(api)
}
