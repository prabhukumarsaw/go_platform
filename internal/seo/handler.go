package seo

import (
	"github.com/gofiber/fiber/v2"
	"newsplatform/api/pkg/response"
)

// Handler exposes HTTP endpoints for SEO, Sitemaps, and RSS Feeds.
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
	router.Get("/sitemaps/:categorySlug.xml", h.GetCategorySitemap)
	router.Get("/feed/rss.xml", h.GetGlobalRSSFeed)
	router.Get("/feed/:categorySlug/rss.xml", h.GetCategoryRSSFeed)
}

// GetGoogleNewsSitemap returns the Google News XML sitemap for the last 48 hours.
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

// GetCategorySitemap returns a sitemap XML for a specific category or regional desk.
func (h *Handler) GetCategorySitemap(c *fiber.Ctx) error {
	categorySlug := c.Params("categorySlug")
	if categorySlug == "" {
		return response.BadRequest(c, "Category slug is required")
	}

	xmlData, err := h.service.GenerateCategorySitemap(c.Context(), categorySlug)
	if err != nil {
		return response.InternalError(c, "Failed to generate category sitemap")
	}

	c.Set("Content-Type", "application/xml; charset=utf-8")
	return c.Send(xmlData)
}

// GetGlobalRSSFeed produces standard RSS 2.0 XML across all published stories.
func (h *Handler) GetGlobalRSSFeed(c *fiber.Ctx) error {
	xmlData, err := h.service.GenerateRSSFeed(c.Context(), "")
	if err != nil {
		return response.InternalError(c, "Failed to generate RSS feed")
	}

	c.Set("Content-Type", "application/rss+xml; charset=utf-8")
	return c.Send(xmlData)
}

// GetCategoryRSSFeed produces RSS 2.0 XML for a specific category or region.
func (h *Handler) GetCategoryRSSFeed(c *fiber.Ctx) error {
	categorySlug := c.Params("categorySlug")
	xmlData, err := h.service.GenerateRSSFeed(c.Context(), categorySlug)
	if err != nil {
		return response.InternalError(c, "Failed to generate category RSS feed")
	}

	c.Set("Content-Type", "application/rss+xml; charset=utf-8")
	return c.Send(xmlData)
}
