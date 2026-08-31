package ads

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// Service handles ad slot management and sponsored content.
type Service struct {
	pool   *pgxpool.Pool
	logger zerolog.Logger
}

// NewService creates a new ads service.
func NewService(pool *pgxpool.Pool, logger zerolog.Logger) *Service {
	return &Service{
		pool:   pool,
		logger: logger.With().Str("module", "ads").Logger(),
	}
}

// ─── Models ─────────────────────────────────────

// AdSlot represents an ad placement configuration.
type AdSlot struct {
	ID        int       `json:"id"`
	TenantID  int       `json:"tenant_id"`
	Name      string    `json:"name"`
	SlotType  string    `json:"slot_type"`
	AdUnitID  string    `json:"ad_unit_id"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// SponsoredArticle tracks sponsored/native content.
type SponsoredArticle struct {
	ArticleID  string `json:"article_id"`
	Sponsor    string `json:"sponsor"`
	CampaignID string `json:"campaign_id"`
	StartDate  string `json:"start_date"`
	EndDate    string `json:"end_date"`
}

// CreateAdSlotInput is the input for creating an ad slot.
type CreateAdSlotInput struct {
	Name     string `json:"name"`
	SlotType string `json:"slot_type"`
	AdUnitID string `json:"ad_unit_id"`
}

// ─── Ad Slot CRUD ───────────────────────────────

// ListAdSlots returns all ad slots for the current tenant.
func (s *Service) ListAdSlots(ctx context.Context, tx pgx.Tx) ([]AdSlot, error) {
	query := `SELECT id, tenant_id, name, slot_type, COALESCE(ad_unit_id, ''), is_active, created_at
			  FROM ad_slots ORDER BY name`

	rows, err := tx.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list ad slots: %w", err)
	}
	defer rows.Close()

	var slots []AdSlot
	for rows.Next() {
		var slot AdSlot
		if err := rows.Scan(&slot.ID, &slot.TenantID, &slot.Name, &slot.SlotType,
			&slot.AdUnitID, &slot.IsActive, &slot.CreatedAt); err != nil {
			return nil, err
		}
		slots = append(slots, slot)
	}
	return slots, rows.Err()
}

// CreateAdSlot creates a new ad slot for the current tenant.
func (s *Service) CreateAdSlot(ctx context.Context, tx pgx.Tx, tenantID int, input CreateAdSlotInput) (*AdSlot, error) {
	query := `INSERT INTO ad_slots (tenant_id, name, slot_type, ad_unit_id)
			  VALUES ($1, $2, $3, $4)
			  RETURNING id, tenant_id, name, slot_type, COALESCE(ad_unit_id, ''), is_active, created_at`

	var slot AdSlot
	err := tx.QueryRow(ctx, query, tenantID, input.Name, input.SlotType, input.AdUnitID).
		Scan(&slot.ID, &slot.TenantID, &slot.Name, &slot.SlotType, &slot.AdUnitID, &slot.IsActive, &slot.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create ad slot: %w", err)
	}
	return &slot, nil
}

// UpdateAdSlot updates an ad slot.
func (s *Service) UpdateAdSlot(ctx context.Context, tx pgx.Tx, id int, input CreateAdSlotInput) error {
	_, err := tx.Exec(ctx,
		`UPDATE ad_slots SET name = $1, slot_type = $2, ad_unit_id = $3, updated_at = NOW() WHERE id = $4`,
		input.Name, input.SlotType, input.AdUnitID, id,
	)
	return err
}

// ToggleAdSlot enables or disables an ad slot.
func (s *Service) ToggleAdSlot(ctx context.Context, tx pgx.Tx, id int, isActive bool) error {
	_, err := tx.Exec(ctx, `UPDATE ad_slots SET is_active = $1, updated_at = NOW() WHERE id = $2`, isActive, id)
	return err
}

// DeleteAdSlot removes an ad slot.
func (s *Service) DeleteAdSlot(ctx context.Context, tx pgx.Tx, id int) error {
	_, err := tx.Exec(ctx, `DELETE FROM ad_slots WHERE id = $1`, id)
	return err
}
