package stream

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/valyala/fasthttp"
)

type Service struct {
	redis  *redis.Client
	logger zerolog.Logger
}

func NewService(redis *redis.Client, logger zerolog.Logger) *Service {
	return &Service{
		redis:  redis,
		logger: logger.With().Str("module", "stream").Logger(),
	}
}

// PublishBreakingNews publishes a breaking news alert to Redis.
func (s *Service) PublishBreakingNews(ctx context.Context, data interface{}) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return s.redis.Publish(ctx, "stream:breaking_news", payload).Err()
}

// PublishLiveBlogEntry publishes an entry to the live blog channel.
func (s *Service) PublishLiveBlogEntry(ctx context.Context, articleID string, entry interface{}) error {
	payload, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return s.redis.Publish(ctx, "stream:live_blog:"+articleID, payload).Err()
}

// Handler handles SSE stream endpoints.
type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterPublicRoutes(router fiber.Router) {
	router.Get("/stream/breaking", h.StreamBreakingNews)
	router.Get("/stream/live-blogs/:articleId", h.StreamLiveBlog)
}

// StreamBreakingNews streams breaking news alerts to clients via Server-Sent Events.
func (h *Handler) StreamBreakingNews(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	pubsub := h.service.redis.Subscribe(c.Context(), "stream:breaking_news")

	c.Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		defer pubsub.Close()
		ch := pubsub.Channel()

		// Initial keepalive ping
		fmt.Fprintf(w, ": connected\n\n")
		_ = w.Flush()

		pingTicker := time.NewTicker(25 * time.Second)
		defer pingTicker.Stop()

		for {
			select {
			case msg, ok := <-ch:
				if !ok {
					return
				}
				fmt.Fprintf(w, "event: breaking_news\ndata: %s\n\n", msg.Payload)
				if err := w.Flush(); err != nil {
					return
				}
			case <-pingTicker.C:
				fmt.Fprintf(w, ": ping\n\n")
				if err := w.Flush(); err != nil {
					return
				}
			}
		}
	}))

	return nil
}

// StreamLiveBlog streams real-time updates for an ongoing live blog.
func (h *Handler) StreamLiveBlog(c *fiber.Ctx) error {
	articleID := c.Params("articleId")
	if articleID == "" {
		return c.Status(400).SendString("Article ID required")
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	pubsub := h.service.redis.Subscribe(c.Context(), "stream:live_blog:"+articleID)

	c.Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		defer pubsub.Close()
		ch := pubsub.Channel()

		fmt.Fprintf(w, ": connected to live blog %s\n\n", articleID)
		_ = w.Flush()

		pingTicker := time.NewTicker(25 * time.Second)
		defer pingTicker.Stop()

		for {
			select {
			case msg, ok := <-ch:
				if !ok {
					return
				}
				fmt.Fprintf(w, "event: live_update\ndata: %s\n\n", msg.Payload)
				if err := w.Flush(); err != nil {
					return
				}
			case <-pingTicker.C:
				fmt.Fprintf(w, ": ping\n\n")
				if err := w.Flush(); err != nil {
					return
				}
			}
		}
	}))

	return nil
}
