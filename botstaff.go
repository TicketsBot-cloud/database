package database

import (
	"context"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type BotStaffTier string

const (
	BotStaffTierAdmin  BotStaffTier = "admin"
	BotStaffTierHelper BotStaffTier = "helper"
)

type BotStaffEntry struct {
	UserId     uint64       `json:"id,string"`
	Tier       BotStaffTier `json:"tier"`
	GlobalView bool         `json:"global_view"`
}

type BotStaff struct {
	*pgxpool.Pool
}

func newBotStaff(db *pgxpool.Pool) *BotStaff {
	return &BotStaff{
		db,
	}
}

func (s BotStaff) Schema() string {
	return `
CREATE TABLE IF NOT EXISTS bot_staff(
	"user_id" int8 NOT NULL UNIQUE,
	"tier" TEXT NOT NULL DEFAULT 'helper',
	PRIMARY KEY("user_id")
);

ALTER TABLE bot_staff ADD COLUMN IF NOT EXISTS "global_view" bool NOT NULL DEFAULT false;`
}

func (s *BotStaff) IsStaff(ctx context.Context, userId uint64) (isStaff bool, err error) {
	query := `
SELECT EXISTS (
	SELECT 1
	FROM bot_staff
	WHERE "user_id" = $1
);
`

	err = s.QueryRow(ctx, query, userId).Scan(&isStaff)
	return
}

func (s *BotStaff) GetTier(ctx context.Context, userId uint64) (BotStaffTier, error) {
	query := `SELECT "tier" FROM bot_staff WHERE "user_id" = $1`

	var tier BotStaffTier
	err := s.QueryRow(ctx, query, userId).Scan(&tier)
	if err == pgx.ErrNoRows {
		return "", nil
	}

	return tier, err
}

// Tier is in the query so no caller can report true for a helper row.
func (s *BotStaff) HasGlobalView(ctx context.Context, userId uint64) (bool, error) {
	query := `SELECT "global_view" FROM bot_staff WHERE "user_id" = $1 AND "tier" = 'admin'`

	var globalView bool
	err := s.QueryRow(ctx, query, userId).Scan(&globalView)
	if err == pgx.ErrNoRows {
		return false, nil
	}

	return globalView, err
}

func (s *BotStaff) SetGlobalView(ctx context.Context, userId uint64, enabled bool) (bool, error) {
	query := `
UPDATE bot_staff
SET "global_view" = $2
WHERE "user_id" = $1 AND "tier" = 'admin';`

	tag, err := s.Exec(ctx, query, userId, enabled)
	if err != nil {
		return false, err
	}

	return tag.RowsAffected() > 0, nil
}

func (s *BotStaff) GetAll(ctx context.Context) ([]BotStaffEntry, error) {
	query := `SELECT "user_id", "tier", "global_view" FROM bot_staff ORDER BY "tier", "user_id";`

	rows, err := s.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var entries []BotStaffEntry
	for rows.Next() {
		var entry BotStaffEntry
		if err = rows.Scan(&entry.UserId, &entry.Tier, &entry.GlobalView); err != nil {
			return nil, err
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

func (s *BotStaff) Add(ctx context.Context, userId uint64, tier BotStaffTier) (err error) {
	query := `
INSERT INTO bot_staff("user_id", "tier")
VALUES($1, $2)
ON CONFLICT("user_id") DO UPDATE
SET "tier" = $2,
    "global_view" = (bot_staff."global_view" AND $2::text = 'admin');
`

	_, err = s.Exec(ctx, query, userId, tier)
	return
}

func (s *BotStaff) UpdateTier(ctx context.Context, userId uint64, tier BotStaffTier) (err error) {
	query := `
UPDATE bot_staff
SET "tier" = $2,
    "global_view" = ("global_view" AND $2::text = 'admin')
WHERE "user_id" = $1;`

	_, err = s.Exec(ctx, query, userId, tier)
	return
}

func (s *BotStaff) Delete(ctx context.Context, userId uint64) (err error) {
	query := `
DELETE FROM bot_staff
WHERE "user_id" = $1;`

	_, err = s.Exec(ctx, query, userId)
	return
}
