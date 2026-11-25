package database

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type GdprExportType string

const (
	GdprExportTypeServer GdprExportType = "server"
	GdprExportTypeUser   GdprExportType = "user"
)

const GdprExportCategoryTranscripts = "transcripts"

var gdprExportCategories = map[GdprExportType][]string{
	GdprExportTypeServer: {
		"settings", "panels", "support_hours", "multi_panels", "forms", "tags", "staff", "blacklist",
		"knowledge_base", "integrations", "tickets", GdprExportCategoryTranscripts, "audit_log", "gallery",
	},
	GdprExportTypeUser: {
		"settings", "account", "notifications", "tickets", "staff_activity", "premium", "affiliate", "whitelabel",
		"integrations", "gallery", "feedback", "blacklists", "audit_log", "gdpr_requests", "discord_data", GdprExportCategoryTranscripts,
	},
}

func (t GdprExportType) Categories() []string {
	return slices.Clone(gdprExportCategories[t])
}

func (t GdprExportType) NormaliseCategories(c []string) ([]string, bool) {
	if len(c) == 0 {
		return nil, false
	}

	all := gdprExportCategories[t]
	for _, key := range c {
		if !slices.Contains(all, key) {
			return nil, false
		}
	}

	var normalised []string
	for _, key := range all {
		if slices.Contains(c, key) {
			normalised = append(normalised, key)
		}
	}

	return normalised, true
}

type GdprExportStatus string

const (
	GdprExportStatusQueued     GdprExportStatus = "queued"
	GdprExportStatusProcessing GdprExportStatus = "processing"
	GdprExportStatusReady      GdprExportStatus = "ready"
	GdprExportStatusFailed     GdprExportStatus = "failed"
	GdprExportStatusNoData     GdprExportStatus = "no_data"
)

type GdprExport struct {
	Id          int64
	Type        GdprExportType
	RequesterId uint64
	GuildIds    []uint64
	Categories  []string
	TicketIds   []int
	Status      GdprExportStatus
	ObjectKey   *string
	GdprLogId   int
	CreatedAt   time.Time
	ExpiresAt   time.Time
}

type GdprExportsTable struct {
	*pgxpool.Pool
}

func newGdprExportsTable(db *pgxpool.Pool) *GdprExportsTable {
	return &GdprExportsTable{
		db,
	}
}

func (GdprExportsTable) Schema() string {
	return `
CREATE TABLE IF NOT EXISTS gdpr_exports (
	id BIGSERIAL PRIMARY KEY,
	type TEXT NOT NULL,
	requester_id INT8 NOT NULL,
	guild_ids INT8[] NOT NULL DEFAULT '{}',
	categories TEXT[] NOT NULL,
	ticket_ids INT4[] DEFAULT NULL,
	status TEXT NOT NULL,
	object_key TEXT DEFAULT NULL,
	gdpr_log_id INT4 NOT NULL REFERENCES gdpr_logs(id),
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	expires_at TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '30 days'
);
CREATE INDEX IF NOT EXISTS gdpr_exports_requester_type_created_at ON gdpr_exports(requester_id, type, created_at DESC);
CREATE INDEX IF NOT EXISTS gdpr_exports_expires_at ON gdpr_exports(expires_at);
`
}

const gdprExportColumns = `id, type, requester_id, guild_ids, categories, ticket_ids, status, object_key, gdpr_log_id, created_at, expires_at`

const gdprExportNextAllowedAtQuery = `
SELECT MAX(created_at) + INTERVAL '24 hours'
FROM gdpr_exports
WHERE requester_id = $1 AND type = $2 AND status IN ('queued', 'processing', 'ready') AND created_at > NOW() - INTERVAL '24 hours';`

func scanGdprExport(row pgx.Row) (GdprExport, error) {
	var e GdprExport
	err := row.Scan(&e.Id, &e.Type, &e.RequesterId, &e.GuildIds, &e.Categories, &e.TicketIds, &e.Status, &e.ObjectKey, &e.GdprLogId, &e.CreatedAt, &e.ExpiresAt)
	return e, err
}

func (t *GdprExportsTable) CreateQueued(ctx context.Context, e GdprExport, requestType string) (GdprExport, *time.Time, error) {
	tx, err := t.Begin(ctx)
	if err != nil {
		return GdprExport{}, nil, err
	}
	defer tx.Rollback(ctx)

	// Without this lock, concurrent creates could all pass the limit check.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0));`, fmt.Sprintf("gdpr_export:%s:%d", e.Type, e.RequesterId)); err != nil {
		return GdprExport{}, nil, err
	}

	var next *time.Time
	if err := tx.QueryRow(ctx, gdprExportNextAllowedAtQuery, e.RequesterId, e.Type).Scan(&next); err != nil {
		return GdprExport{}, nil, err
	}

	if next != nil {
		return GdprExport{}, next, nil
	}

	logQuery := `INSERT INTO gdpr_logs (requester, request_type, status) VALUES ($1, $2, 'Queued') RETURNING id;`
	if err := tx.QueryRow(ctx, logQuery, HashGdprRequester(e.RequesterId), requestType).Scan(&e.GdprLogId); err != nil {
		return GdprExport{}, nil, err
	}

	exportQuery := `
INSERT INTO gdpr_exports (type, requester_id, guild_ids, categories, ticket_ids, status, gdpr_log_id)
VALUES ($1, $2, COALESCE($3::int8[], '{}'), $4, $5, 'queued', $6)
RETURNING id, created_at, expires_at;`

	if err := tx.QueryRow(ctx, exportQuery, e.Type, e.RequesterId, e.GuildIds, e.Categories, e.TicketIds, e.GdprLogId).Scan(&e.Id, &e.CreatedAt, &e.ExpiresAt); err != nil {
		return GdprExport{}, nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return GdprExport{}, nil, err
	}

	e.Status = GdprExportStatusQueued
	return e, nil, nil
}

func (t *GdprExportsTable) NextAllowedAt(ctx context.Context, requesterId uint64, exportType GdprExportType) (*time.Time, error) {
	var next *time.Time
	if err := t.QueryRow(ctx, gdprExportNextAllowedAtQuery, requesterId, exportType).Scan(&next); err != nil {
		return nil, err
	}

	return next, nil
}

func (t *GdprExportsTable) Get(ctx context.Context, id int64) (*GdprExport, error) {
	e, err := scanGdprExport(t.QueryRow(ctx, `SELECT `+gdprExportColumns+` FROM gdpr_exports WHERE id = $1;`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &e, nil
}

func (t *GdprExportsTable) SetStatus(ctx context.Context, id int64, status GdprExportStatus) error {
	_, err := t.Exec(ctx, `UPDATE gdpr_exports SET status = $2 WHERE id = $1;`, id, status)
	return err
}

func (t *GdprExportsTable) SetReady(ctx context.Context, id int64, objectKey string) error {
	_, err := t.Exec(ctx, `UPDATE gdpr_exports SET status = 'ready', object_key = $2 WHERE id = $1;`, id, objectKey)
	return err
}

func (t *GdprExportsTable) ListExpired(ctx context.Context, limit int) ([]GdprExport, error) {
	rows, err := t.Query(ctx, `SELECT `+gdprExportColumns+` FROM gdpr_exports WHERE expires_at <= NOW() ORDER BY expires_at LIMIT $1;`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var exports []GdprExport
	for rows.Next() {
		e, err := scanGdprExport(rows)
		if err != nil {
			return nil, err
		}

		exports = append(exports, e)
	}

	return exports, rows.Err()
}

func (t *GdprExportsTable) ListByRequester(ctx context.Context, requesterId uint64) ([]GdprExport, error) {
	rows, err := t.Query(ctx, `SELECT `+gdprExportColumns+` FROM gdpr_exports WHERE requester_id = $1 ORDER BY created_at DESC;`, requesterId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var exports []GdprExport
	for rows.Next() {
		e, err := scanGdprExport(rows)
		if err != nil {
			return nil, err
		}

		exports = append(exports, e)
	}

	return exports, rows.Err()
}

func (t *GdprExportsTable) Delete(ctx context.Context, id int64) error {
	_, err := t.Exec(ctx, `DELETE FROM gdpr_exports WHERE id = $1;`, id)
	return err
}
