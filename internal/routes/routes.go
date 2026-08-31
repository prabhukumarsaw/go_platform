package routes

import (
	// Standard Library
	"strconv"
	"strings"

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

	// Internal Domain Handlers - Revenue, Discovery, Multi-Tenancy & Analytics
	"newsplatform/api/internal/ads"
	"newsplatform/api/internal/analytics"
	"newsplatform/api/internal/seo"
	"newsplatform/api/internal/tenant"

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

	// Monetization, Discovery & Tenant Infrastructure
	Ads           *ads.Handler
	SEO           *seo.Handler
	Analytics     *analytics.Handler
	Tenant        *tenant.Handler
	TenantService *tenant.Service
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

	// Indian ISO-3166-2 Region to State Edition Slug Mapping
	indianRegionMap := map[string]string{
		"MH": "maharashtra", "JH": "jharkhand", "DL": "delhi", "UP": "uttar-pradesh",
		"WB": "west-bengal", "TN": "tamil-nadu", "KA": "karnataka", "GJ": "gujarat",
		"RJ": "rajasthan", "PB": "punjab", "BR": "bihar", "MP": "madhya-pradesh",
		"KL": "kerala", "AP": "andhra-pradesh", "TS": "telangana", "TG": "telangana",
		"OR": "odisha", "OD": "odisha", "AS": "assam", "GA": "goa",
		"HP": "himachal-pradesh", "UK": "uttarakhand", "UT": "uttarakhand",
		"HR": "haryana", "JK": "jammu-kashmir", "CH": "chandigarh", "CT": "chhattisgarh",
		"CG": "chhattisgarh", "TR": "tripura", "ML": "meghalaya", "MN": "manipur",
		"NL": "nagaland", "MZ": "mizoram", "SK": "sikkim", "AR": "arunachal-pradesh",
		"PY": "puducherry", "AN": "andaman-nicobar", "LA": "ladakh",
	}

	// Public Tenant Resolver (?state=maharashtra, Cloudflare CF-Region, header, or default national)
	publicTenantResolver := func(c *fiber.Ctx) (int64, error) {
		stateSlug := c.Query("state")
		if stateSlug == "" {
			stateSlug = c.Query("tenant")
		}
		if stateSlug == "" {
			stateSlug = c.Query("edition")
		}

		// 1. Explicit query param (with regional aliases)
		if stateSlug != "" {
			normalizedSlug := strings.ToLower(strings.TrimSpace(stateSlug))
			aliases := map[string]string{
				"kolkata":      "west-bengal",
				"calcutta":     "west-bengal",
				"bengal":       "west-bengal",
				"up":           "uttar-pradesh",
				"orissa":       "odisha",
				"odisa":        "odisha",
				"ncr":          "delhi",
				"new-delhi":    "delhi",
				"cg":           "chhattisgarh",
				"chhatisgarh":  "chhattisgarh",
				"chhatishgar":  "chhattisgarh",
				"jh":           "jharkhand",
				"br":           "bihar",
			}
			if target, exists := aliases[normalizedSlug]; exists {
				normalizedSlug = target
			}

			t, err := h.TenantService.GetTenantBySlug(c.Context(), normalizedSlug)
			if err == nil && t != nil {
				return int64(t.ID), nil
			}
		}

		// 2. Custom header
		if tid := c.Get("X-Tenant-ID"); tid != "" {
			if id, err := strconv.ParseInt(tid, 10, 64); err == nil {
				return id, nil
			}
		}

		// 3. Hyperlocal Edge Geolocation (Cloudflare CF-Region-Code or CF-Region)
		cfRegion := strings.ToUpper(strings.TrimSpace(c.Get("CF-Region-Code")))
		if cfRegion == "" {
			cfRegion = strings.ToUpper(strings.TrimSpace(c.Get("CF-Region")))
		}
		if cfRegion == "" {
			cfRegion = strings.ToUpper(strings.TrimSpace(c.Get("X-Country-Region")))
		}

		if mappedSlug, ok := indianRegionMap[cfRegion]; ok {
			t, err := h.TenantService.GetTenantBySlug(c.Context(), mappedSlug)
			if err == nil && t != nil {
				return int64(t.ID), nil
			}
		}

		return 1, nil // Default National
	}

	api := app.Group("/api/v1")

	// ─── 1. Public Authentication Routes ────────────
	h.Auth.RegisterRoutes(api, cfg.JWT)

	// ─── 2. Public Reader & Audience Routes ─────────
	// Edge Caching: s-maxage=60s on Cloudflare/CDN edge, stale-while-revalidate=300s
	publicRoutes := api.Group("",
		middleware.TenantContextPublic(pool, publicTenantResolver),
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

	// Public Tenant / State editions list
	api.Get("/tenants", func(c *fiber.Ctx) error {
		tenants, err := h.TenantService.ListTenants(c.Context())
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to list tenants"})
		}
		return c.JSON(fiber.Map{"success": true, "data": tenants})
	})

	// ─── 3. Staff Studio Routes (Newsroom CMS) ──────
	authRequired := api.Group("", middleware.RequireAuth(cfg.JWT))
	staffRoutes := authRequired.Group("", middleware.RequireStaff(), middleware.TenantContext(pool))

	h.Content.RegisterStudioRoutes(staffRoutes)
	h.Media.RegisterRoutes(staffRoutes)
	h.WebStory.RegisterStudioRoutes(staffRoutes)
	h.EPaper.RegisterStudioRoutes(staffRoutes)
	h.AI.RegisterStudioRoutes(staffRoutes)

	// ─── 4. Admin & Governance Routes ───────────────
	adminRoutes := staffRoutes.Group("/admin")
	h.IAM.RegisterRoutes(adminRoutes)
	h.Tenant.RegisterRoutes(adminRoutes)
	h.Ads.RegisterRoutes(adminRoutes)
	h.Moderation.RegisterAdminRoutes(adminRoutes)
	h.Employee.RegisterAdminRoutes(adminRoutes)
	h.Analytics.RegisterAdminRoutes(adminRoutes)
	h.Notify.RegisterAdminRoutes(adminRoutes)

	// ─── 5. SEO Sitemaps & RSS Feeds ───────────────
	h.SEO.RegisterPublicRoutes(app)
	h.SEO.RegisterPublicRoutes(api)
}
