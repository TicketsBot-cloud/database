package database

import (
	"context"
	_ "embed"
	"errors"

	"github.com/TicketsBot-cloud/common/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Skus struct {
	*pgxpool.Pool
}

// SkuWithDetails represents a SKU with its optional subscription and multi-server details.
type SkuWithDetails struct {
	model.Sku
	Tier             *string `json:"tier,omitempty"`
	Priority         *int32  `json:"priority,omitempty"`
	IsGlobal         *bool   `json:"is_global,omitempty"`
	ServersPermitted *int    `json:"servers_permitted,omitempty"`
}

var (
	//go:embed sql/skus/schema.sql
	skusSchema string

	//go:embed sql/skus/get.sql
	skusGet string

	//go:embed sql/skus/insert.sql
	skusInsert string

	//go:embed sql/skus/update.sql
	skusUpdate string

	//go:embed sql/skus/delete.sql
	skusDelete string

	//go:embed sql/skus/list_all.sql
	skusListAll string
)

func newSkusTable(db *pgxpool.Pool) *Skus {
	return &Skus{
		db,
	}
}

func (Skus) Schema() string {
	return skusSchema
}

// ListAll returns all SKUs with their subscription and multi-server details.
func (s *Skus) ListAll(ctx context.Context) ([]SkuWithDetails, error) {
	rows, err := s.Query(ctx, skusListAll)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var skus []SkuWithDetails
	for rows.Next() {
		var sku SkuWithDetails
		if err := rows.Scan(
			&sku.Id,
			&sku.Label,
			&sku.SkuType,
			&sku.Tier,
			&sku.Priority,
			&sku.IsGlobal,
			&sku.ServersPermitted,
		); err != nil {
			return nil, err
		}

		skus = append(skus, sku)
	}

	return skus, nil
}

// GetById returns a single SKU with its details, or nil if not found.
func (s *Skus) GetById(ctx context.Context, id uuid.UUID) (*SkuWithDetails, error) {
	var sku SkuWithDetails
	if err := s.QueryRow(ctx, skusGet, id).Scan(
		&sku.Id,
		&sku.Label,
		&sku.SkuType,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return &sku, nil
}

// Create inserts a new SKU and its optional subscription and multi-server details within a transaction.
func (s *Skus) Create(ctx context.Context, tx pgx.Tx, label string, skuType model.SkuType) (model.Sku, error) {
	var sku model.Sku
	if err := tx.QueryRow(ctx, skusInsert, label, skuType).Scan(
		&sku.Id,
		&sku.Label,
		&sku.SkuType,
	); err != nil {
		return model.Sku{}, err
	}

	return sku, nil
}

// Update updates the base SKU fields within a transaction.
func (s *Skus) Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, label string, skuType model.SkuType) error {
	_, err := tx.Exec(ctx, skusUpdate, id, label, skuType)
	return err
}

// Delete removes a SKU by its ID within a transaction.
func (s *Skus) Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	_, err := tx.Exec(ctx, skusDelete, id)
	return err
}
