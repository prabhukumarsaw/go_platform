package poll

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

type PollOption struct {
	ID         int    `json:"id"`
	Text       string `json:"text"`
	Votes      int64  `json:"votes"`
	Percentage int    `json:"percentage"`
}

type Poll struct {
	ID         uuid.UUID    `json:"id"`
	TenantID   int          `json:"tenant_id"`
	Question   string       `json:"question"`
	Language   string       `json:"language"`
	Options    []PollOption `json:"options"`
	TotalVotes int64        `json:"total_votes"`
	IsActive   bool         `json:"is_active"`
	ExpiresAt  *time.Time   `json:"expires_at,omitempty"`
	CreatedAt  time.Time    `json:"created_at"`
}

type Service struct {
	pool   *pgxpool.Pool
	logger zerolog.Logger
}

func NewService(pool *pgxpool.Pool, logger zerolog.Logger) *Service {
	return &Service{
		pool:   pool,
		logger: logger.With().Str("module", "poll").Logger(),
	}
}

func (s *Service) GetActivePoll(ctx context.Context, tx pgx.Tx, tenantID int, language string) (*Poll, error) {
	if language == "" {
		language = "hi"
	}

	query := `
		SELECT id, tenant_id, question, language, options, total_votes, is_active, expires_at, created_at
		FROM polls
		WHERE (tenant_id = $1 OR tenant_id = 1)
		  AND language = $2
		  AND is_active = TRUE
		  AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY created_at DESC
		LIMIT 1
	`

	var p Poll
	var rawOptions json.RawMessage
	err := tx.QueryRow(ctx, query, tenantID, language).Scan(
		&p.ID, &p.TenantID, &p.Question, &p.Language,
		&rawOptions, &p.TotalVotes, &p.IsActive, &p.ExpiresAt, &p.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	_ = json.Unmarshal(rawOptions, &p.Options)

	// Calculate percentages
	if p.TotalVotes > 0 {
		for i := range p.Options {
			p.Options[i].Percentage = int((float64(p.Options[i].Votes) / float64(p.TotalVotes)) * 100)
		}
	}

	return &p, nil
}

func (s *Service) Vote(ctx context.Context, tx pgx.Tx, pollID uuid.UUID, optionID int, userID *int64, ipAddress string) (*Poll, error) {
	// Deduplication check: check if already voted
	var alreadyVoted bool
	if userID != nil {
		_ = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM poll_votes WHERE poll_id = $1 AND user_id = $2)", pollID, *userID).Scan(&alreadyVoted)
	} else if ipAddress != "" {
		_ = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM poll_votes WHERE poll_id = $1 AND host(ip_address) = $2)", pollID, ipAddress).Scan(&alreadyVoted)
	}

	if alreadyVoted {
		return nil, fmt.Errorf("you have already voted in this poll")
	}

	// Record vote
	_, err := tx.Exec(ctx, `
		INSERT INTO poll_votes (poll_id, user_id, option_id, ip_address)
		VALUES ($1, $2, $3, $4::inet)
	`, pollID, userID, optionID, ipAddress)
	if err != nil {
		return nil, fmt.Errorf("record vote: %w", err)
	}

	// Fetch current options and increment tally
	var rawOptions json.RawMessage
	var totalVotes int64
	err = tx.QueryRow(ctx, "SELECT options, total_votes FROM polls WHERE id = $1 FOR UPDATE", pollID).Scan(&rawOptions, &totalVotes)
	if err != nil {
		return nil, err
	}

	var options []PollOption
	_ = json.Unmarshal(rawOptions, &options)

	for i := range options {
		if options[i].ID == optionID {
			options[i].Votes++
			break
		}
	}
	totalVotes++

	updatedOptionsJSON, _ := json.Marshal(options)
	_, err = tx.Exec(ctx, "UPDATE polls SET options = $1, total_votes = $2, updated_at = NOW() WHERE id = $3", updatedOptionsJSON, totalVotes, pollID)
	if err != nil {
		return nil, err
	}

	// Return updated poll
	var updatedPoll Poll
	_ = tx.QueryRow(ctx, `SELECT id, tenant_id, question, language, options, total_votes, is_active, expires_at, created_at FROM polls WHERE id = $1`, pollID).
		Scan(&updatedPoll.ID, &updatedPoll.TenantID, &updatedPoll.Question, &updatedPoll.Language, &rawOptions, &updatedPoll.TotalVotes, &updatedPoll.IsActive, &updatedPoll.ExpiresAt, &updatedPoll.CreatedAt)
	_ = json.Unmarshal(rawOptions, &updatedPoll.Options)

	if updatedPoll.TotalVotes > 0 {
		for i := range updatedPoll.Options {
			updatedPoll.Options[i].Percentage = int((float64(updatedPoll.Options[i].Votes) / float64(updatedPoll.TotalVotes)) * 100)
		}
	}

	return &updatedPoll, nil
}
