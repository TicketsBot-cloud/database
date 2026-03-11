package database

import (
	"context"
	_ "embed"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type PremiumKeys struct {
	*pgxpool.Pool
}

var (
	//go:embed sql/premium_keys/list_all.sql
	premiumKeysListAll string

	//go:embed sql/premium_keys/count_all.sql
	premiumKeysCountAll string
)

type PremiumKeyEntry struct {
	Key         uuid.UUID      `json:"key"`
	Length      *time.Duration `json:"length"`
	SkuId       *uuid.UUID     `json:"sku_id"`
	GeneratedAt *time.Time     `json:"generated_at"`
	SkuLabel    *string        `json:"sku_label"`
	Tier        *string        `json:"tier"`
	GuildId     *uint64        `json:"guild_id"`
	ActivatedBy *uint64        `json:"activated_by"`
	IsUsed      bool           `json:"is_used"`
}

func newPremiumKeys(db *pgxpool.Pool) *PremiumKeys {
	return &PremiumKeys{
		db,
	}
}

func (k PremiumKeys) Schema() string {
	return `
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE TABLE IF NOT EXISTS premium_keys(
	"key" uuid NOT NULL UNIQUE,
	"length" interval NOT NULL,
	"sku_id" UUID NOT NULL,
	"generated_at" TIMESTAMPTZ NOT NULL,
	PRIMARY KEY("key"),
	FOREIGN KEY("sku_id") REFERENCES skus("id")
);`
}

func (k *PremiumKeys) Create(ctx context.Context, key uuid.UUID, length time.Duration, skuId uuid.UUID) (err error) {
	_, err = k.Exec(ctx, `INSERT INTO premium_keys("key", "length", "sku_id", "generated_at") VALUES($1, $2, $3, NOW());`, key, length, skuId)
	return
}

func (k *PremiumKeys) Delete(ctx context.Context, tx pgx.Tx, key uuid.UUID) (time.Duration, uuid.UUID, bool, error) {
	var length time.Duration
	var skuId uuid.UUID

	query := `DELETE from premium_keys WHERE "key" = $1 RETURNING "length", "sku_id";`
	if err := tx.QueryRow(ctx, query, key).Scan(&length, &skuId); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, uuid.Nil, false, nil
		}

		return 0, uuid.Nil, false, err
	}

	return length, skuId, true, nil
}

func (k *PremiumKeys) ListAll(ctx context.Context, limit, offset int) ([]PremiumKeyEntry, error) {
	rows, err := k.Query(ctx, premiumKeysListAll, limit, offset)
	if err != nil {
		return nil, err
	}

	var entries []PremiumKeyEntry
	for rows.Next() {
		var entry PremiumKeyEntry
		if err := rows.Scan(
			&entry.Key,
			&entry.Length,
			&entry.SkuId,
			&entry.GeneratedAt,
			&entry.SkuLabel,
			&entry.Tier,
			&entry.GuildId,
			&entry.ActivatedBy,
			&entry.IsUsed,
		); err != nil {
			return nil, err
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

func (k *PremiumKeys) CountAll(ctx context.Context) (int, error) {
	var count int
	if err := k.QueryRow(ctx, premiumKeysCountAll).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}
