package database

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type AutomationCronSchedule struct {
	AutomationId   int64      `json:"automation_id,string"`
	GuildId        uint64     `json:"guild_id,string"`
	CronExpression string     `json:"cron_expression"`
	Timezone       string     `json:"timezone"`
	LastFiredAt    *time.Time `json:"last_fired_at,omitempty"`
}

type AutomationCronSchedulesTable struct {
	*pgxpool.Pool
}

func newAutomationCronSchedulesTable(db *pgxpool.Pool) *AutomationCronSchedulesTable {
	return &AutomationCronSchedulesTable{
		db,
	}
}

func (AutomationCronSchedulesTable) Schema() string {
	return `
CREATE TABLE IF NOT EXISTS automation_cron_schedules(
	"automation_id" int8 NOT NULL,
	"guild_id" int8 NOT NULL,
	"cron_expression" TEXT NOT NULL,
	"timezone" TEXT NOT NULL DEFAULT 'UTC',
	"last_fired_at" TIMESTAMPTZ DEFAULT NULL,
	PRIMARY KEY("automation_id"),
	FOREIGN KEY ("automation_id") REFERENCES automations("id") ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS automation_cron_schedules_guild ON automation_cron_schedules("guild_id");
`
}

func (t *AutomationCronSchedulesTable) Get(ctx context.Context, automationId int64) (AutomationCronSchedule, bool, error) {
	query := `
SELECT "automation_id", "guild_id", "cron_expression", "timezone", "last_fired_at"
FROM automation_cron_schedules
WHERE "automation_id" = $1;
`

	var s AutomationCronSchedule
	err := t.QueryRow(ctx, query, automationId).Scan(
		&s.AutomationId, &s.GuildId, &s.CronExpression, &s.Timezone, &s.LastFiredAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AutomationCronSchedule{}, false, nil
		}
		return AutomationCronSchedule{}, false, err
	}
	return s, true, nil
}

func (t *AutomationCronSchedulesTable) GetAll(ctx context.Context) ([]AutomationCronSchedule, error) {
	query := `
SELECT "automation_id", "guild_id", "cron_expression", "timezone", "last_fired_at"
FROM automation_cron_schedules;
`

	rows, err := t.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	schedules := make([]AutomationCronSchedule, 0)
	for rows.Next() {
		var s AutomationCronSchedule
		if err := rows.Scan(
			&s.AutomationId, &s.GuildId, &s.CronExpression, &s.Timezone, &s.LastFiredAt,
		); err != nil {
			return nil, err
		}
		schedules = append(schedules, s)
	}
	return schedules, nil
}

func (t *AutomationCronSchedulesTable) Upsert(ctx context.Context, automationId int64, guildId uint64, cronExpression, timezone string) error {
	query := `
INSERT INTO automation_cron_schedules("automation_id", "guild_id", "cron_expression", "timezone")
VALUES($1, $2, $3, $4)
ON CONFLICT ("automation_id") DO UPDATE
	SET "cron_expression" = EXCLUDED."cron_expression",
	    "timezone" = EXCLUDED."timezone",
	    "guild_id" = EXCLUDED."guild_id";
`
	_, err := t.Exec(ctx, query, automationId, guildId, cronExpression, timezone)
	return err
}

func (t *AutomationCronSchedulesTable) SetLastFiredAt(ctx context.Context, automationId int64, firedAt time.Time) error {
	query := `UPDATE automation_cron_schedules SET "last_fired_at" = $2 WHERE "automation_id" = $1;`
	_, err := t.Exec(ctx, query, automationId, firedAt)
	return err
}

func (t *AutomationCronSchedulesTable) Delete(ctx context.Context, automationId int64) error {
	query := `DELETE FROM automation_cron_schedules WHERE "automation_id" = $1;`
	_, err := t.Exec(ctx, query, automationId)
	return err
}
