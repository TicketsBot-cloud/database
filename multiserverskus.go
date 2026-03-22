package database

import (
	"context"
	_ "embed"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type MultiServerSkus struct {
	*pgxpool.Pool
}

var (
	//go:embed sql/multi_server_skus/schema.sql
	multiServerSkusSchema string

	//go:embed sql/multi_server_skus/get_permitted_server_count.sql
	multiServerSkusGetPermittedServerCount string

	//go:embed sql/multi_server_skus/insert.sql
	multiServerSkusInsert string

	//go:embed sql/multi_server_skus/delete.sql
	multiServerSkusDelete string
)

func newMultiServerSkusTable(db *pgxpool.Pool) *MultiServerSkus {
	return &MultiServerSkus{
		db,
	}
}

func (MultiServerSkus) Schema() string {
	return multiServerSkusSchema
}

// Upsert inserts or updates a multi-server SKU entry within a transaction.
func (m *MultiServerSkus) Upsert(ctx context.Context, tx pgx.Tx, skuId uuid.UUID, serversPermitted int) error {
	_, err := tx.Exec(ctx, multiServerSkusInsert, skuId, serversPermitted)
	return err
}

// DeleteBySku removes a multi-server SKU entry by its SKU ID within a transaction.
func (m *MultiServerSkus) DeleteBySku(ctx context.Context, tx pgx.Tx, skuId uuid.UUID) error {
	_, err := tx.Exec(ctx, multiServerSkusDelete, skuId)
	return err
}

func (m *MultiServerSkus) GetPermittedServerCount(ctx context.Context, tx pgx.Tx, skuId uuid.UUID) (int, bool, error) {
	var count int
	if err := tx.QueryRow(ctx, multiServerSkusGetPermittedServerCount, skuId).Scan(&count); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, false, nil
		}

		return 0, false, err
	}

	return count, true, nil
}
