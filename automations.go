package database

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

// ErrAutomationQuotaExceeded is returned by CreateWithLimit when a guild is at or
// above its tier's automation count limit. Callers surface a 402.
var ErrAutomationQuotaExceeded = errors.New("automation quota exceeded for this guild tier")

type AutomationGraph struct {
	Version int              `json:"version"`
	Nodes   []AutomationNode `json:"nodes"`
	Edges   []AutomationEdge `json:"edges"`
}

type AutomationNode struct {
	Id       string                    `json:"id"`
	Type     string                    `json:"type"`
	Kind     string                    `json:"kind"`
	Config   map[string]any            `json:"config,omitempty"`
	OnError  *AutomationNodeOnError    `json:"onError,omitempty"`
	Clauses  []AutomationClause        `json:"clauses,omitempty"`
	Mode     string                    `json:"mode,omitempty"`
	Position AutomationNodePosition    `json:"position"`
}

type AutomationNodePosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type AutomationClause struct {
	Left  string `json:"left"`
	Op    string `json:"op"`
	Right string `json:"right"`
}

type AutomationNodeOnError struct {
	Policy     string `json:"policy"`
	Retries    int    `json:"retries,omitempty"`
	BackoffSec int    `json:"backoffSec,omitempty"`
}

type AutomationEdge struct {
	From     string `json:"from"`
	FromPort string `json:"fromPort"`
	To       string `json:"to"`
}

type Automation struct {
	Id               int64            `json:"id,string"`
	GuildId          uint64           `json:"guild_id,string"`
	Name             string           `json:"name"`
	Description      *string          `json:"description,omitempty"`
	Enabled          bool             `json:"enabled"`
	DraftGraph       AutomationGraph  `json:"draft_graph"`
	DraftUpdatedAt   time.Time        `json:"draft_updated_at"`
	PublishedGraph   *AutomationGraph `json:"published_graph,omitempty"`
	PublishedVersion int              `json:"published_version"`
	PublishedAt      *time.Time       `json:"published_at,omitempty"`
	WebhookSecret    *string          `json:"webhook_secret,omitempty"`
	CreatedBy        uint64           `json:"created_by,string"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

type AutomationSummary struct {
	Id               int64      `json:"id,string"`
	GuildId          uint64     `json:"guild_id,string"`
	Name             string     `json:"name"`
	Description      *string    `json:"description,omitempty"`
	Enabled          bool       `json:"enabled"`
	TriggerKind      string     `json:"trigger_kind"`
	PublishedVersion int        `json:"published_version"`
	PublishedAt      *time.Time `json:"published_at,omitempty"`
	DraftUpdatedAt   time.Time  `json:"draft_updated_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type AutomationsTable struct {
	*pgxpool.Pool
}

func newAutomationsTable(db *pgxpool.Pool) *AutomationsTable {
	return &AutomationsTable{
		db,
	}
}

func (AutomationsTable) Schema() string {
	return `
CREATE TABLE IF NOT EXISTS automations(
	"id" BIGSERIAL NOT NULL,
	"guild_id" int8 NOT NULL,
	"name" VARCHAR(100) NOT NULL,
	"description" TEXT DEFAULT NULL,
	"enabled" bool NOT NULL DEFAULT false,
	"draft_graph" JSONB NOT NULL,
	"draft_updated_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
	"published_graph" JSONB DEFAULT NULL,
	"published_version" INT NOT NULL DEFAULT 0,
	"published_at" TIMESTAMPTZ DEFAULT NULL,
	"webhook_secret" TEXT DEFAULT NULL,
	"created_by" int8 NOT NULL,
	"created_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
	"updated_at" TIMESTAMPTZ NOT NULL DEFAULT now(),
	PRIMARY KEY("id")
);
CREATE INDEX IF NOT EXISTS automations_guild_id ON automations("guild_id");
CREATE INDEX IF NOT EXISTS automations_guild_enabled ON automations("guild_id", "enabled") WHERE "published_graph" IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS automations_webhook_secret ON automations("webhook_secret") WHERE "webhook_secret" IS NOT NULL;
`
}

func (a *AutomationsTable) GetByGuild(ctx context.Context, guildId uint64) ([]AutomationSummary, error) {
	// Prefer the published trigger when one exists so the list shows what's actually
	// running in production, falling back to the draft for never-published automations.
	query := `
SELECT
	"id", "guild_id", "name", "description", "enabled",
	"published_version", "published_at", "draft_updated_at", "updated_at",
	COALESCE(
		"published_graph" -> 'nodes' -> 0 ->> 'kind',
		"draft_graph" -> 'nodes' -> 0 ->> 'kind',
		''
	) AS trigger_kind
FROM automations
WHERE "guild_id" = $1
ORDER BY "updated_at" DESC;
`

	rows, err := a.Query(ctx, query, guildId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	summaries := make([]AutomationSummary, 0)
	for rows.Next() {
		var s AutomationSummary
		if err := rows.Scan(
			&s.Id, &s.GuildId, &s.Name, &s.Description, &s.Enabled,
			&s.PublishedVersion, &s.PublishedAt, &s.DraftUpdatedAt, &s.UpdatedAt,
			&s.TriggerKind,
		); err != nil {
			return nil, err
		}
		summaries = append(summaries, s)
	}

	return summaries, nil
}

func (a *AutomationsTable) Get(ctx context.Context, automationId int64) (Automation, bool, error) {
	query := `
SELECT
	"id", "guild_id", "name", "description", "enabled",
	"draft_graph", "draft_updated_at",
	"published_graph", "published_version", "published_at",
	"webhook_secret", "created_by", "created_at", "updated_at"
FROM automations
WHERE "id" = $1;
`

	var auto Automation
	// pgx v4 treats JSONB columns as typed destinations when scanned into []byte,
	// trying to json.Unmarshal into the slice and failing with "cannot unmarshal
	// object into Go value of type []uint8". Scan into *string to get the raw
	// JSON text back, then unmarshal ourselves — same pattern as multipanels.go.
	var draftRaw string
	var publishedRaw *string

	err := a.QueryRow(ctx, query, automationId).Scan(
		&auto.Id, &auto.GuildId, &auto.Name, &auto.Description, &auto.Enabled,
		&draftRaw, &auto.DraftUpdatedAt,
		&publishedRaw, &auto.PublishedVersion, &auto.PublishedAt,
		&auto.WebhookSecret, &auto.CreatedBy, &auto.CreatedAt, &auto.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Automation{}, false, nil
		}
		return Automation{}, false, err
	}

	if err := json.Unmarshal([]byte(draftRaw), &auto.DraftGraph); err != nil {
		return Automation{}, false, err
	}

	if publishedRaw != nil && *publishedRaw != "" {
		var g AutomationGraph
		if err := json.Unmarshal([]byte(*publishedRaw), &g); err != nil {
			return Automation{}, false, err
		}
		auto.PublishedGraph = &g
	}

	return auto, true, nil
}

func (a *AutomationsTable) Create(ctx context.Context, guildId uint64, name string, description *string, draft AutomationGraph, createdBy uint64) (int64, error) {
	draftRaw, err := json.Marshal(draft)
	if err != nil {
		return 0, err
	}

	query := `
INSERT INTO automations("guild_id", "name", "description", "draft_graph", "created_by")
VALUES($1, $2, $3, $4, $5)
RETURNING "id";
`

	var id int64
	if err := a.QueryRow(ctx, query, guildId, name, description, draftRaw, createdBy).Scan(&id); err != nil {
		return 0, err
	}

	return id, nil
}

// CreateWithLimit atomically checks the per-guild automation quota and inserts a new
// row only if the guild is under maxCount. Returns ErrAutomationQuotaExceeded if not.
//
// Serialises concurrent creates per-guild via a transactional advisory lock, so three
// parallel POSTs from the same guild cannot all pass the count check and overshoot.
// Different guilds hash to different lock keys (with astronomical collision probability);
// an accidental collision just briefly serialises two guilds, which is harmless.
func (a *AutomationsTable) CreateWithLimit(
	ctx context.Context,
	guildId uint64,
	maxCount int,
	name string,
	description *string,
	draft AutomationGraph,
	createdBy uint64,
) (int64, error) {
	draftRaw, err := json.Marshal(draft)
	if err != nil {
		return 0, err
	}

	tx, err := a.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() {
		// Rollback is a no-op after successful commit; safe to defer unconditionally.
		rbCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = tx.Rollback(rbCtx)
	}()

	// uint64 -> int64 bit reinterpretation for pg_advisory_xact_lock(bigint).
	// Collision domain is all 2^63 int64 values; collisions are OK (briefly serialises
	// two guilds). Released automatically on commit or rollback.
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", int64(guildId)); err != nil {
		return 0, err
	}

	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM automations WHERE "guild_id" = $1`, guildId).Scan(&count); err != nil {
		return 0, err
	}
	if count >= maxCount {
		return 0, ErrAutomationQuotaExceeded
	}

	var id int64
	if err := tx.QueryRow(
		ctx,
		`INSERT INTO automations("guild_id", "name", "description", "draft_graph", "created_by") VALUES($1, $2, $3, $4, $5) RETURNING "id";`,
		guildId, name, description, draftRaw, createdBy,
	).Scan(&id); err != nil {
		return 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return id, nil
}

func (a *AutomationsTable) UpdateDraft(ctx context.Context, automationId int64, name string, description *string, draft AutomationGraph) error {
	draftRaw, err := json.Marshal(draft)
	if err != nil {
		return err
	}

	query := `
UPDATE automations
SET "name" = $2,
    "description" = $3,
    "draft_graph" = $4,
    "draft_updated_at" = now(),
    "updated_at" = now()
WHERE "id" = $1;
`

	_, err = a.Exec(ctx, query, automationId, name, description, draftRaw)
	return err
}

func (a *AutomationsTable) Publish(ctx context.Context, automationId int64) error {
	query := `
UPDATE automations
SET "published_graph" = "draft_graph",
    "published_version" = "published_version" + 1,
    "published_at" = now(),
    "updated_at" = now()
WHERE "id" = $1;
`

	_, err := a.Exec(ctx, query, automationId)
	return err
}

// PublishGraph promotes the supplied, already-validated graph to published state.
// Prefer this over Publish() so the validation and the write are atomic with respect
// to the graph bytes — a concurrent PATCH to draft_graph between the handler's read
// and the publish can't smuggle an invalid graph into published_graph.
func (a *AutomationsTable) PublishGraph(ctx context.Context, automationId int64, validated AutomationGraph) error {
	raw, err := json.Marshal(validated)
	if err != nil {
		return err
	}

	query := `
UPDATE automations
SET "published_graph" = $2,
    "published_version" = "published_version" + 1,
    "published_at" = now(),
    "updated_at" = now()
WHERE "id" = $1;
`

	_, err = a.Exec(ctx, query, automationId, raw)
	return err
}

func (a *AutomationsTable) Revert(ctx context.Context, automationId int64) error {
	query := `
UPDATE automations
SET "draft_graph" = COALESCE("published_graph", "draft_graph"),
    "draft_updated_at" = now(),
    "updated_at" = now()
WHERE "id" = $1;
`

	_, err := a.Exec(ctx, query, automationId)
	return err
}

func (a *AutomationsTable) SetEnabled(ctx context.Context, automationId int64, enabled bool) error {
	query := `UPDATE automations SET "enabled" = $2, "updated_at" = now() WHERE "id" = $1;`
	_, err := a.Exec(ctx, query, automationId, enabled)
	return err
}

func (a *AutomationsTable) Delete(ctx context.Context, automationId int64) error {
	query := `DELETE FROM automations WHERE "id" = $1;`
	_, err := a.Exec(ctx, query, automationId)
	return err
}

func (a *AutomationsTable) CountByGuild(ctx context.Context, guildId uint64) (int, error) {
	query := `SELECT COUNT(*) FROM automations WHERE "guild_id" = $1;`
	var count int
	if err := a.QueryRow(ctx, query, guildId).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (a *AutomationsTable) GetByWebhookSecret(ctx context.Context, secret string) (Automation, bool, error) {
	query := `
SELECT
	"id", "guild_id", "name", "description", "enabled",
	"draft_graph", "draft_updated_at",
	"published_graph", "published_version", "published_at",
	"webhook_secret", "created_by", "created_at", "updated_at"
FROM automations
WHERE "webhook_secret" = $1;
`
	var auto Automation
	var draftRaw string
	var publishedRaw *string

	err := a.QueryRow(ctx, query, secret).Scan(
		&auto.Id, &auto.GuildId, &auto.Name, &auto.Description, &auto.Enabled,
		&draftRaw, &auto.DraftUpdatedAt,
		&publishedRaw, &auto.PublishedVersion, &auto.PublishedAt,
		&auto.WebhookSecret, &auto.CreatedBy, &auto.CreatedAt, &auto.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Automation{}, false, nil
		}
		return Automation{}, false, err
	}

	if err := json.Unmarshal([]byte(draftRaw), &auto.DraftGraph); err != nil {
		return Automation{}, false, err
	}
	if publishedRaw != nil && *publishedRaw != "" {
		var g AutomationGraph
		if err := json.Unmarshal([]byte(*publishedRaw), &g); err != nil {
			return Automation{}, false, err
		}
		auto.PublishedGraph = &g
	}
	return auto, true, nil
}

func (a *AutomationsTable) SetWebhookSecret(ctx context.Context, automationId int64, secret *string) error {
	query := `UPDATE automations SET "webhook_secret" = $2, "updated_at" = now() WHERE "id" = $1;`
	_, err := a.Exec(ctx, query, automationId, secret)
	return err
}

func (a *AutomationsTable) GetEnabledForTrigger(ctx context.Context, guildId uint64, triggerKind string) ([]Automation, error) {
	query := `
SELECT
	"id", "guild_id", "name", "description", "enabled",
	"draft_graph", "draft_updated_at",
	"published_graph", "published_version", "published_at",
	"webhook_secret", "created_by", "created_at", "updated_at"
FROM automations
WHERE "guild_id" = $1
  AND "enabled" = true
  AND "published_graph" IS NOT NULL
  AND "published_graph" -> 'nodes' -> 0 ->> 'kind' = $2;
`

	rows, err := a.Query(ctx, query, guildId, triggerKind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	automations := make([]Automation, 0)
	for rows.Next() {
		var auto Automation
		var draftRaw string
		var publishedRaw *string

		if err := rows.Scan(
			&auto.Id, &auto.GuildId, &auto.Name, &auto.Description, &auto.Enabled,
			&draftRaw, &auto.DraftUpdatedAt,
			&publishedRaw, &auto.PublishedVersion, &auto.PublishedAt,
			&auto.WebhookSecret, &auto.CreatedBy, &auto.CreatedAt, &auto.UpdatedAt,
		); err != nil {
			return nil, err
		}

		if err := json.Unmarshal([]byte(draftRaw), &auto.DraftGraph); err != nil {
			return nil, err
		}

		if publishedRaw != nil && *publishedRaw != "" {
			var g AutomationGraph
			if err := json.Unmarshal([]byte(*publishedRaw), &g); err != nil {
				return nil, err
			}
			auto.PublishedGraph = &g
		}

		automations = append(automations, auto)
	}

	return automations, nil
}
