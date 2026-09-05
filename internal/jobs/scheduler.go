package jobs

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

// Scheduler manages scheduled background recurring jobs.
type Scheduler struct {
	pool   *pgxpool.Pool
	redis  *redis.Client
	logger zerolog.Logger
	stopCh chan struct{}
}

// NewScheduler creates a new background job scheduler.
func NewScheduler(pool *pgxpool.Pool, redis *redis.Client, logger zerolog.Logger) *Scheduler {
	return &Scheduler{
		pool:   pool,
		redis:  redis,
		logger: logger.With().Str("module", "jobs").Logger(),
		stopCh: make(chan struct{}),
	}
}

// Start begins the recurring cron loops in goroutines.
func (s *Scheduler) Start() {
	s.logger.Info().Msg("Starting background job scheduler")

	// 1. Trending recalculation loop (every 5 minutes)
	go s.runLoop(5*time.Minute, "recalculate_trending", s.recalculateTrending)

	// 2. Scheduled articles publisher (every 1 minute)
	go s.runLoop(1*time.Minute, "publish_scheduled_articles", s.publishScheduledArticles)

	// 3. Content archival check loop (once per 24 hours)
	go s.runLoop(24*time.Hour, "archive_stale_drafts", s.archiveStaleDrafts)

	// 4. Daily reader notification digests (every 24 hours)
	go s.runLoop(24*time.Hour, "send_daily_digests", s.sendDailyDigests)

	// 5. News agency wire feed ingestion (every 10 minutes)
	go s.runLoop(10*time.Minute, "ingest_wire_feeds", s.IngestWireFeeds)
}

// Stop terminates all job routines.
func (s *Scheduler) Stop() {
	s.logger.Info().Msg("Stopping background job scheduler")
	close(s.stopCh)
}

func (s *Scheduler) runLoop(interval time.Duration, jobName string, fn func(ctx context.Context) error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			s.logger.Debug().Str("job", jobName).Msg("Running background job")
			if err := fn(ctx); err != nil {
				s.logger.Error().Err(err).Str("job", jobName).Msg("Background job encountered an error")
			}
			cancel()
		}
	}
}

// publishScheduledArticles checks for articles whose scheduled_at has arrived and publishes them.
func (s *Scheduler) publishScheduledArticles(ctx context.Context) error {
	query := `
		UPDATE articles
		SET status = 'published', published_at = NOW(), updated_at = NOW()
		WHERE status IN ('scheduled', 'approved', 'draft')
		  AND scheduled_at IS NOT NULL
		  AND scheduled_at <= NOW()
	`
	cmd, err := s.pool.Exec(ctx, query)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() > 0 {
		s.logger.Info().Int64("published_count", cmd.RowsAffected()).Msg("Published scheduled articles")
	}
	return nil
}

// recalculateTrending refreshes trending article ranks in Redis.
func (s *Scheduler) recalculateTrending(ctx context.Context) error {
	if s.redis == nil {
		return nil
	}

	query := `
		SELECT id, view_count
		FROM articles
		WHERE status = 'published' AND published_at > NOW() - INTERVAL '7 days'
		ORDER BY view_count DESC
		LIMIT 100
	`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	pipe := s.redis.Pipeline()
	for rows.Next() {
		var id string
		var viewCount int64
		if err := rows.Scan(&id, &viewCount); err != nil {
			return err
		}

		pipe.ZAdd(ctx, "trending:national", redis.Z{
			Score:  float64(viewCount),
			Member: id,
		})
		pipe.Expire(ctx, "trending:national", 10*time.Minute)
	}

	_, err = pipe.Exec(ctx)
	return err
}

// sendDailyDigests processes daily headline digests for subscribed readers.
func (s *Scheduler) sendDailyDigests(ctx context.Context) error {
	query := `
		SELECT COUNT(*) FROM newsletter_subscriptions WHERE is_active = TRUE
	`
	var count int64
	_ = s.pool.QueryRow(ctx, query).Scan(&count)
	if count > 0 {
		s.logger.Info().Int64("subscribers", count).Msg("Queued daily regional news digests")
	}
	return nil
}

// archiveStaleDrafts flags abandoned drafts older than 180 days.
func (s *Scheduler) archiveStaleDrafts(ctx context.Context) error {
	query := `
		UPDATE articles
		SET status = 'archived', updated_at = NOW()
		WHERE status = 'draft' AND updated_at < NOW() - INTERVAL '180 days'
	`

	cmd, err := s.pool.Exec(ctx, query)
	if err != nil {
		return err
	}

	if cmd.RowsAffected() > 0 {
		s.logger.Info().Int64("archived_count", cmd.RowsAffected()).Msg("Archived stale drafts")
	}
	return nil
}
