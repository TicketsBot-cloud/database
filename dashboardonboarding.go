package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type DashboardOnboarding struct {
	GuildId              uint64     `json:"guild_id,string"`
	OnboardingCompleted  bool       `json:"onboarding_completed"`
	OnboardingCompletedAt *time.Time `json:"onboarding_completed_at"`
	CurrentStep          int16      `json:"current_step"`
	Skipped              bool       `json:"skipped"`
}

type DashboardOnboardingTable struct {
	*pgxpool.Pool
}

func newDashboardOnboardingTable(pool *pgxpool.Pool) *DashboardOnboardingTable {
	return &DashboardOnboardingTable{
		pool,
	}
}

func (t DashboardOnboardingTable) Schema() string {
	return `
CREATE TABLE IF NOT EXISTS dashboard_onboarding(
	"guild_id"                INT8 NOT NULL,
	"onboarding_completed"    BOOL NOT NULL DEFAULT false,
	"onboarding_completed_at" TIMESTAMPTZ DEFAULT NULL,
	"current_step"            INT2 NOT NULL DEFAULT 0,
	"skipped"                 BOOL NOT NULL DEFAULT false,
	PRIMARY KEY("guild_id")
);
`
}

func (t *DashboardOnboardingTable) Get(ctx context.Context, guildId uint64) (DashboardOnboarding, error) {
	query := `
SELECT "onboarding_completed", "onboarding_completed_at", "current_step", "skipped"
FROM dashboard_onboarding
WHERE "guild_id" = $1;
`

	var onboarding DashboardOnboarding
	onboarding.GuildId = guildId

	err := t.QueryRow(ctx, query, guildId).Scan(
		&onboarding.OnboardingCompleted,
		&onboarding.OnboardingCompletedAt,
		&onboarding.CurrentStep,
		&onboarding.Skipped,
	)

	if err == pgx.ErrNoRows {
		return DashboardOnboarding{GuildId: guildId}, nil
	} else if err != nil {
		return DashboardOnboarding{}, err
	}

	return onboarding, nil
}

func (t *DashboardOnboardingTable) SetStep(ctx context.Context, guildId uint64, step int16) error {
	query := `
INSERT INTO dashboard_onboarding("guild_id", "current_step")
VALUES($1, $2)
ON CONFLICT("guild_id")
DO UPDATE SET "current_step" = $2;
`

	_, err := t.Exec(ctx, query, guildId, step)
	return err
}

func (t *DashboardOnboardingTable) Complete(ctx context.Context, guildId uint64) error {
	query := `
INSERT INTO dashboard_onboarding("guild_id", "onboarding_completed", "onboarding_completed_at", "current_step")
VALUES($1, true, NOW(), 5)
ON CONFLICT("guild_id")
DO UPDATE SET "onboarding_completed" = true, "onboarding_completed_at" = NOW(), "current_step" = 5;
`

	_, err := t.Exec(ctx, query, guildId)
	return err
}

func (t *DashboardOnboardingTable) Skip(ctx context.Context, guildId uint64) error {
	query := `
INSERT INTO dashboard_onboarding("guild_id", "skipped")
VALUES($1, true)
ON CONFLICT("guild_id")
DO UPDATE SET "skipped" = true;
`

	_, err := t.Exec(ctx, query, guildId)
	return err
}
