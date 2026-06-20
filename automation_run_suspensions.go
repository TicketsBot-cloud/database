package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

type AutomationRunSuspension struct {
	RunId           int64
	AutomationId    int64
	GuildId         uint64
	ResumeAt        time.Time
	ResumeNodeId    string
	StepOutputs     []byte
	VisitedNodes    []string
	StepsExecuted   int
	CausationId     string
	WorkflowVersion int
	CreatedAt       time.Time
}

type AutomationRunSuspensionsTable struct {
	*pgxpool.Pool
}

func newAutomationRunSuspensionsTable(db *pgxpool.Pool) *AutomationRunSuspensionsTable {
	return &AutomationRunSuspensionsTable{
		db,
	}
}

func (AutomationRunSuspensionsTable) Schema() string {
	return `
CREATE TABLE IF NOT EXISTS automation_run_suspensions(
	"run_id" BIGINT PRIMARY KEY,
	"automation_id" BIGINT NOT NULL,
	"guild_id" BIGINT NOT NULL,
	"resume_at" TIMESTAMPTZ NOT NULL,
	"resume_node_id" TEXT NOT NULL,
	"step_outputs" JSONB NOT NULL DEFAULT '{}',
	"visited_nodes" TEXT[] NOT NULL DEFAULT '{}',
	"steps_executed" INT NOT NULL DEFAULT 0,
	"causation_id" TEXT NOT NULL,
	"workflow_version" INT NOT NULL,
	"created_at" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	FOREIGN KEY("run_id") REFERENCES automation_runs("id") ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS automation_run_suspensions_resume ON automation_run_suspensions("resume_at");
CREATE INDEX IF NOT EXISTS automation_run_suspensions_guild ON automation_run_suspensions("guild_id");
`
}

func (t *AutomationRunSuspensionsTable) Insert(ctx context.Context, s AutomationRunSuspension) error {
	query := `
INSERT INTO automation_run_suspensions("run_id", "automation_id", "guild_id", "resume_at", "resume_node_id", "step_outputs", "visited_nodes", "steps_executed", "causation_id", "workflow_version")
VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := t.Pool.Exec(ctx, query, s.RunId, s.AutomationId, s.GuildId, s.ResumeAt, s.ResumeNodeId, s.StepOutputs, s.VisitedNodes, s.StepsExecuted, s.CausationId, s.WorkflowVersion)
	return err
}

func (t *AutomationRunSuspensionsTable) GetDue(ctx context.Context, limit int) ([]AutomationRunSuspension, error) {
	query := `
SELECT "run_id", "automation_id", "guild_id", "resume_at", "resume_node_id", "step_outputs", "visited_nodes", "steps_executed", "causation_id", "workflow_version", "created_at"
FROM automation_run_suspensions
WHERE "resume_at" <= NOW()
ORDER BY "resume_at" ASC
LIMIT $1`
	rows, err := t.Pool.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []AutomationRunSuspension
	for rows.Next() {
		var s AutomationRunSuspension
		if err := rows.Scan(&s.RunId, &s.AutomationId, &s.GuildId, &s.ResumeAt, &s.ResumeNodeId, &s.StepOutputs, &s.VisitedNodes, &s.StepsExecuted, &s.CausationId, &s.WorkflowVersion, &s.CreatedAt); err != nil {
			return nil, err
		}
		results = append(results, s)
	}
	return results, nil
}

func (t *AutomationRunSuspensionsTable) Delete(ctx context.Context, runId int64) error {
	query := `DELETE FROM automation_run_suspensions WHERE "run_id" = $1`
	_, err := t.Pool.Exec(ctx, query, runId)
	return err
}

// CountByGuild returns the number of outstanding (not yet resumed) suspensions for
// a guild. Used to cap pending delays and prevent a guild from flooding the table.
func (t *AutomationRunSuspensionsTable) CountByGuild(ctx context.Context, guildId uint64) (int, error) {
	query := `SELECT COUNT(*) FROM automation_run_suspensions WHERE "guild_id" = $1`
	var count int
	err := t.Pool.QueryRow(ctx, query, guildId).Scan(&count)
	return count, err
}
