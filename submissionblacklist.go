package database

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type SubmissionFeature string

const (
	SubmissionFeatureGallery      SubmissionFeature = "gallery"
	SubmissionFeatureIntegrations SubmissionFeature = "integrations"
)

type SubmissionTargetType string

const (
	SubmissionTargetUser  SubmissionTargetType = "user"
	SubmissionTargetGuild SubmissionTargetType = "guild"
)

func (f SubmissionFeature) AllowsTarget(t SubmissionTargetType) bool {
	switch f {
	case SubmissionFeatureGallery:
		return t == SubmissionTargetUser || t == SubmissionTargetGuild
	case SubmissionFeatureIntegrations:
		return t == SubmissionTargetUser
	default:
		return false
	}
}

type SubmissionBlacklistEntry struct {
	TargetType SubmissionTargetType
	TargetId   uint64
	Reason     *string
}

type SubmissionBlacklist struct {
	*pgxpool.Pool
}

func newSubmissionBlacklist(db *pgxpool.Pool) *SubmissionBlacklist {
	return &SubmissionBlacklist{
		db,
	}
}

func (b SubmissionBlacklist) Schema() string {
	return `
CREATE TABLE IF NOT EXISTS submission_blacklist(
	"feature" TEXT NOT NULL,
	"target_type" TEXT NOT NULL,
	"target_id" int8 NOT NULL,
	"reason" TEXT,
	PRIMARY KEY("feature", "target_type", "target_id")
);
`
}

func (b *SubmissionBlacklist) IsBlacklisted(ctx context.Context, feature SubmissionFeature, userId, guildId uint64) (bool, SubmissionTargetType, error) {
	query := `
SELECT "target_type" FROM submission_blacklist
WHERE "feature" = $1
AND (("target_type" = 'user' AND "target_id" = $2) OR ("target_type" = 'guild' AND "target_id" = $3))
ORDER BY "target_type" = 'user' DESC
LIMIT 1;
`

	var targetType SubmissionTargetType
	if err := b.QueryRow(ctx, query, feature, userId, guildId).Scan(&targetType); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, "", nil
		}
		return false, "", err
	}

	return true, targetType, nil
}

func (b *SubmissionBlacklist) List(ctx context.Context, feature SubmissionFeature) ([]SubmissionBlacklistEntry, error) {
	query := `
SELECT "target_type", "target_id", "reason"
FROM submission_blacklist
WHERE "feature" = $1
ORDER BY "target_type", "target_id";
`

	rows, err := b.Query(ctx, query, feature)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []SubmissionBlacklistEntry
	for rows.Next() {
		var entry SubmissionBlacklistEntry
		if err := rows.Scan(&entry.TargetType, &entry.TargetId, &entry.Reason); err != nil {
			return nil, err
		}

		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

func (b *SubmissionBlacklist) Add(ctx context.Context, feature SubmissionFeature, targetType SubmissionTargetType, targetId uint64, reason *string) (err error) {
	query := `
INSERT INTO submission_blacklist("feature", "target_type", "target_id", "reason")
VALUES($1, $2, $3, $4)
ON CONFLICT("feature", "target_type", "target_id") DO UPDATE SET "reason" = EXCLUDED."reason";
`

	_, err = b.Exec(ctx, query, feature, targetType, targetId, reason)
	return
}

func (b *SubmissionBlacklist) Delete(ctx context.Context, feature SubmissionFeature, targetType SubmissionTargetType, targetId uint64) (err error) {
	_, err = b.Exec(ctx, `DELETE FROM submission_blacklist WHERE "feature" = $1 AND "target_type" = $2 AND "target_id" = $3;`, feature, targetType, targetId)
	return
}
