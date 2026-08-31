package epaper

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

type EPaper struct {
	ID           uuid.UUID `json:"id"`
	TenantID     int       `json:"tenant_id"`
	DistrictID   *int      `json:"district_id,omitempty"`
	DistrictName string    `json:"district_name,omitempty"`
	EditionDate  string    `json:"edition_date"`
	Title        string    `json:"title"`
	PDFURL       string    `json:"pdf_url"`
	ThumbnailURL string    `json:"thumbnail_url"`
	PageCount    int       `json:"page_count"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
}

type Service struct {
	pool   *pgxpool.Pool
	logger zerolog.Logger
}

func NewService(pool *pgxpool.Pool, logger zerolog.Logger) *Service {
	return &Service{
		pool:   pool,
		logger: logger.With().Str("module", "epaper").Logger(),
	}
}

func (s *Service) ListEPapers(ctx context.Context, tx pgx.Tx, tenantID int, districtID *int, dateStr string) ([]EPaper, error) {
	query := `
		SELECT e.id, e.tenant_id, e.district_id, COALESCE(d.name, '') as district_name,
		       to_char(e.edition_date, 'YYYY-MM-DD') as edition_date,
		       e.title, e.pdf_url, COALESCE(e.thumbnail_url, ''), e.page_count, e.is_active, e.created_at
		FROM epapers e
		LEFT JOIN districts d ON d.id = e.district_id
		WHERE (e.tenant_id = $1 OR $1 = 1) AND e.is_active = TRUE
	`
	args := []interface{}{tenantID}
	argIdx := 2

	if districtID != nil {
		query += fmt.Sprintf(" AND e.district_id = $%d", argIdx)
		args = append(args, *districtID)
		argIdx++
	}

	if dateStr != "" {
		query += fmt.Sprintf(" AND e.edition_date = $%d::date", argIdx)
		args = append(args, dateStr)
		argIdx++
	}

	query += " ORDER BY e.edition_date DESC, e.title ASC LIMIT 30"

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []EPaper
	for rows.Next() {
		var ep EPaper
		if err := rows.Scan(
			&ep.ID, &ep.TenantID, &ep.DistrictID, &ep.DistrictName,
			&ep.EditionDate, &ep.Title, &ep.PDFURL, &ep.ThumbnailURL,
			&ep.PageCount, &ep.IsActive, &ep.CreatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, ep)
	}
	return list, rows.Err()
}

func (s *Service) CreateEPaper(ctx context.Context, tx pgx.Tx, tenantID int, districtID *int, editionDate, title, pdfURL, thumbURL string, pageCount int) (*EPaper, error) {
	query := `
		INSERT INTO epapers (tenant_id, district_id, edition_date, title, pdf_url, thumbnail_url, page_count)
		VALUES ($1, $2, $3::date, $4, $5, $6, $7)
		RETURNING id, tenant_id, district_id, to_char(edition_date, 'YYYY-MM-DD'), title, pdf_url, COALESCE(thumbnail_url, ''), page_count, is_active, created_at
	`
	var ep EPaper
	err := tx.QueryRow(ctx, query, tenantID, districtID, editionDate, title, pdfURL, thumbURL, pageCount).
		Scan(&ep.ID, &ep.TenantID, &ep.DistrictID, &ep.EditionDate, &ep.Title, &ep.PDFURL, &ep.ThumbnailURL, &ep.PageCount, &ep.IsActive, &ep.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create epaper: %w", err)
	}
	return &ep, nil
}
