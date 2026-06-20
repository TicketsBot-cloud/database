package database

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

const (
	AutomationRunStatusRunning     = "running"
	AutomationRunStatusSuccess     = "success"
	AutomationRunStatusFailed      = "failed"
	AutomationRunStatusPartial     = "partial"
	AutomationRunStatusRateLimited = "rate_limited"
	AutomationRunStatusTimedOut    = "timed_out"
	AutomationRunStatusSkipped     = "skipped"
	AutomationRunStatusSuspended   = "suspended"
)

type AutomationRun struct {
	Id              int64      `json:"id,string"`
	AutomationId    *int64     `json:"automation_id,string,omitempty"`
	AutomationName  string     `json:"automation_name"`
	GuildId         uint64     `json:"guild_id,string"`
	TriggerType     string     `json:"trigger_type"`
	TriggerPayload  []byte     `json:"trigger_payload,omitempty"`
	Status          string     `json:"status"`
	StartedAt       time.Time  `json:"started_at"`
	FinishedAt      *time.Time `json:"finished_at,omitempty"`
	DurationMs      *int       `json:"duration_ms,omitempty"`
	Error           *string    `json:"error,omitempty"`
	CausationId     string     `json:"causation_id"`
	WorkflowVersion int        `json:"workflow_version"`
}

type AutomationRunsTable struct {
	*pgxpool.Pool
}

func newAutomationRunsTable(db *pgxpool.Pool) *AutomationRunsTable {
	return &AutomationRunsTable{
		db,
	}
}

func (AutomationRunsTable) Schema() string {
	return `
CREATE TABLE IF NOT EXISTS automation_runs(
	"id" BIGSERIAL NOT NULL,
	"automation_id" int8 DEFAULT NULL,
	"automation_name" TEXT NOT NULL,
	"guild_id" int8 NOT NULL,
	"trigger_type" TEXT NOT NULL,
	"trigger_payload" JSONB NOT NULL,
	"status" VARCHAR(32) NOT NULL,
	"started_at" TIMESTAMPTZ NOT NULL,
	"finished_at" TIMESTAMPTZ DEFAULT NULL,
	"duration_ms" INT DEFAULT NULL,
	"error" TEXT DEFAULT NULL,
	"causation_id" UUID NOT NULL,
	"workflow_version" INT NOT NULL,
	PRIMARY KEY("id"),
	FOREIGN KEY ("automation_id") REFERENCES automations("id") ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS automation_runs_guild_started ON automation_runs("guild_id", "started_at" DESC);
CREATE INDEX IF NOT EXISTS automation_runs_automation_started ON automation_runs("automation_id", "started_at" DESC);
CREATE INDEX IF NOT EXISTS automation_runs_causation ON automation_runs("causation_id");
CREATE UNIQUE INDEX IF NOT EXISTS automation_runs_causation_automation_unique ON automation_runs("causation_id", "automation_id");
`
}

// StartRun inserts a row for an automation execution at whatever status the caller
// specifies (usually AutomationRunStatusRunning for real runs, or a terminal status
// like AutomationRunStatusRateLimited for single-shot records).
//
// Returns (id, true) if inserted, or (0, false) if another row for the same
// (causation_id, automation_id) already exists. Callers treat a false return as a
// recursion/redelivery short-circuit — do NOT run the automation in that case.
//
// For runs started with AutomationRunStatusRunning, callers must call FinishRun at
// completion to set the terminal status, finished_at, duration_ms, and error fields.
func (t *AutomationRunsTable) StartRun(ctx context.Context, run AutomationRun) (int64, bool, error) {
	query := `
INSERT INTO automation_runs(
	"automation_id", "automation_name", "guild_id", "trigger_type", "trigger_payload",
	"status", "started_at", "finished_at", "duration_ms", "error",
	"causation_id", "workflow_version"
)
VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
ON CONFLICT ("causation_id", "automation_id") DO NOTHING
RETURNING "id";
`

	var id int64
	err := t.QueryRow(ctx, query,
		run.AutomationId, run.AutomationName, run.GuildId, run.TriggerType, run.TriggerPayload,
		run.Status, run.StartedAt, run.FinishedAt, run.DurationMs, run.Error,
		run.CausationId, run.WorkflowVersion,
	).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, false, nil // conflict
		}
		return 0, false, err
	}
	return id, true, nil
}

// FinishRun updates a row previously inserted by StartRun with the terminal status,
// timing, and optional error message. Callers should pass nil errMsg on success.
func (t *AutomationRunsTable) FinishRun(ctx context.Context, id int64, status string, finishedAt time.Time, durationMs int, errMsg *string) error {
	query := `
UPDATE automation_runs
SET "status" = $2,
    "finished_at" = $3,
    "duration_ms" = $4,
    "error" = $5
WHERE "id" = $1;
`
	_, err := t.Exec(ctx, query, id, status, finishedAt, durationMs, errMsg)
	return err
}

func (t *AutomationRunsTable) Create(ctx context.Context, run AutomationRun) (int64, error) {
	query := `
INSERT INTO automation_runs(
	"automation_id", "automation_name", "guild_id", "trigger_type", "trigger_payload",
	"status", "started_at", "finished_at", "duration_ms", "error",
	"causation_id", "workflow_version"
)
VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING "id";
`

	var id int64
	err := t.QueryRow(ctx, query,
		run.AutomationId, run.AutomationName, run.GuildId, run.TriggerType, run.TriggerPayload,
		run.Status, run.StartedAt, run.FinishedAt, run.DurationMs, run.Error,
		run.CausationId, run.WorkflowVersion,
	).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (t *AutomationRunsTable) GetByAutomation(ctx context.Context, automationId int64, limit, offset int) ([]AutomationRun, error) {
	query := `
SELECT
	"id", "automation_id", "automation_name", "guild_id", "trigger_type", "trigger_payload",
	"status", "started_at", "finished_at", "duration_ms", "error",
	"causation_id", "workflow_version"
FROM automation_runs
WHERE "automation_id" = $1
ORDER BY "started_at" DESC
LIMIT $2 OFFSET $3;
`

	rows, err := t.Query(ctx, query, automationId, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	runs := make([]AutomationRun, 0)
	for rows.Next() {
		var r AutomationRun
		// JSONB columns must be scanned into string then converted — see the
		// comment in automations.Get for why.
		var payload string
		if err := rows.Scan(
			&r.Id, &r.AutomationId, &r.AutomationName, &r.GuildId, &r.TriggerType, &payload,
			&r.Status, &r.StartedAt, &r.FinishedAt, &r.DurationMs, &r.Error,
			&r.CausationId, &r.WorkflowVersion,
		); err != nil {
			return nil, err
		}
		r.TriggerPayload = []byte(payload)
		runs = append(runs, r)
	}

	return runs, nil
}

func (t *AutomationRunsTable) GetByCausation(ctx context.Context, causationId string) (AutomationRun, bool, error) {
	query := `
SELECT
	"id", "automation_id", "automation_name", "guild_id", "trigger_type", "trigger_payload",
	"status", "started_at", "finished_at", "duration_ms", "error",
	"causation_id", "workflow_version"
FROM automation_runs
WHERE "causation_id" = $1
LIMIT 1;
`

	var r AutomationRun
	var payload string
	err := t.QueryRow(ctx, query, causationId).Scan(
		&r.Id, &r.AutomationId, &r.AutomationName, &r.GuildId, &r.TriggerType, &payload,
		&r.Status, &r.StartedAt, &r.FinishedAt, &r.DurationMs, &r.Error,
		&r.CausationId, &r.WorkflowVersion,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AutomationRun{}, false, nil
		}
		return AutomationRun{}, false, err
	}
	r.TriggerPayload = []byte(payload)
	return r, true, nil
}

func (t *AutomationRunsTable) CountByGuildSince(ctx context.Context, guildId uint64, since time.Time) (int, error) {
	query := `SELECT COUNT(*) FROM automation_runs WHERE "guild_id" = $1 AND "started_at" >= $2;`
	var count int
	if err := t.QueryRow(ctx, query, guildId, since).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// AutomationDailyStat is one day's run aggregate for an automation.
// Days without runs still appear (total/success/failed all zero) because the
// query generates a full date range via generate_series.
type AutomationDailyStat struct {
	Day     time.Time `json:"day"`
	Total   int       `json:"total"`
	Success int       `json:"success"`
	Failed  int       `json:"failed"`
}

// StatsForAutomation returns per-day run counts for the trailing `days` window,
// oldest day first, newest last. Zero-run days are filled via generate_series so
// the frontend can render a complete sparkline without any date-gap logic.
//
// `failed` buckets anything that did not succeed and isn't still running —
// failed / timed_out / rate_limited / partial all roll up visually.
func (t *AutomationRunsTable) StatsForAutomation(ctx context.Context, automationId int64, days int) ([]AutomationDailyStat, error) {
	if days <= 0 {
		days = 7
	}
	query := `
WITH day_range AS (
    SELECT generate_series(
        date_trunc('day', now() AT TIME ZONE 'UTC') - make_interval(days => $2 - 1),
        date_trunc('day', now() AT TIME ZONE 'UTC'),
        interval '1 day'
    ) AS day
)
SELECT
    day_range.day,
    COUNT(r.id)::int AS total,
    COUNT(*) FILTER (WHERE r.status = 'success')::int AS success,
    COUNT(*) FILTER (WHERE r.status IN ('failed', 'timed_out', 'rate_limited', 'partial'))::int AS failed
FROM day_range
LEFT JOIN automation_runs r
    ON date_trunc('day', r.started_at AT TIME ZONE 'UTC') = day_range.day
   AND r.automation_id = $1
GROUP BY day_range.day
ORDER BY day_range.day ASC;
`

	rows, err := t.Query(ctx, query, automationId, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make([]AutomationDailyStat, 0, days)
	for rows.Next() {
		var s AutomationDailyStat
		if err := rows.Scan(&s.Day, &s.Total, &s.Success, &s.Failed); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, nil
}

func (t *AutomationRunsTable) DeleteOlderThan(ctx context.Context, cutoff time.Time) error {
	query := `DELETE FROM automation_runs WHERE "started_at" < $1;`
	_, err := t.Exec(ctx, query, cutoff)
	return err
}
