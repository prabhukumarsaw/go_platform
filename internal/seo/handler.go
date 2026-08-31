package seo

import (
	"github.com/gofiber/fiber/v2"
	"newsplatform/api/pkg/response"
)

// Handler exposes HTTP endpoints for SEO and Sitemaps.
type Handler struct {
	service *Service
}

// NewHandler creates a new SEO handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterPublicRoutes registers public SEO routes (sitemaps and RSS feeds).
func (h *Handler) RegisterPublicRoutes(router fiber.Router) {
	router.Get("/sitemap.xml", h.GetSitemapIndex)
	router.Get("/sitemap-news.xml", h.GetGoogleNewsSitemap)
	router.Get("/sitemaps/:tenantSlug.xml", h.GetTenantSitemap)
	router.Get("/feed/rss.xml", h.GetGlobalRSSFeed)
	router.Get("/feed/:tenantSlug/rss.xml", h.GetTenantRSSFeed)
}

// GetGoogleNewsSitemap returns the Google News XML sitemap for last 48 hours.
func (h *Handler) GetGoogleNewsSitemap(c *fiber.Ctx) error {
	xmlData, err := h.service.GenerateGoogleNewsSitemap(c.Context())
	if err != nil {
		return response.InternalError(c, "Failed to generate Google News sitemap: "+err.Error())
	}

	c.Set("Content-Type", "application/xml; charset=utf-8")
	return c.Send(xmlData)
}

// GetSitemapIndex returns the main sitemap index XML.
func (h *Handler) GetSitemapIndex(c *fiber.Ctx) error {
	xmlData, err := h.service.GenerateSitemapIndex(c.Context())
	if err != nil {
		return response.InternalError(c, "Failed to generate sitemap index")
	}

	c.Set("Content-Type", "application/xml; charset=utf-8")
	return c.Send(xmlData)
}

// GetTenantSitemap returns a sitemap XML for a specific state edition/tenant.
func (h *Handler) GetTenantSitemap(c *fiber.Ctx) error {
	tenantSlug := c.Params("tenantSlug")
	if tenantSlug == "" {
		return response.BadRequest(c, "Tenant slug is required")
	}

	xmlData, err := h.service.GenerateTenantSitemap(c.Context(), tenantSlug)
	if err != nil {
		return response.InternalError(c, "Failed to generate tenant sitemap")
	}

	c.Set("Content-Type", "application/xml; charset=utf-8")
	return c.Send(xmlData)
}

// GetGlobalRSSFeed produces standard RSS 2.0 XML across all state editions.
func (h *Handler) GetGlobalRSSFeed(c *fiber.Ctx) error {
	xmlData, err := h.service.GenerateRSSFeed(c.Context(), "")
	if err != nil {
		return response.InternalError(c, "Failed to generate RSS feed")
	}

	c.Set("Content-Type", "application/rss+xml; charset=utf-8")
	return c.Send(xmlData)
}

// GetTenantRSSFeed produces RSS 2.0 XML for a specific state edition.
func (h *Handler) GetTenantRSSFeed(c *fiber.Ctx) error {
	tenantSlug := c.Params("tenantSlug")
	xmlData, err := h.service.GenerateRSSFeed(c.Context(), tenantSlug)
	if err != nil {
		return response.InternalError(c, "Failed to generate state RSS feed")
	}

	c.Set("Content-Type", "application/rss+xml; charset=utf-8")
	return c.Send(xmlData)
}
