package database

import (
	"context"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

// SwitchPanelClaimBehavior defines behavior when switching a claimed ticket to a panel the claimer can't access
type SwitchPanelClaimBehavior int

const (
	// SwitchPanelAutoUnclaim automatically unclaims the ticket if the claimer
	// doesn't have access to the new panel (default behavior)
	SwitchPanelAutoUnclaim SwitchPanelClaimBehavior = iota

	// SwitchPanelBlockSwitch prevents switching to a panel if the claimer
	// doesn't have access to it
	SwitchPanelBlockSwitch

	// SwitchPanelRemoveOnUnclaim allows the switch, but removes the claimer's
	// access to the ticket when they unclaim
	SwitchPanelRemoveOnUnclaim

	// SwitchPanelKeepAccess allows the switch and keeps the claimer's access
	// to the ticket even after unclaiming
	SwitchPanelKeepAccess
)

type ClaimSettings struct {
	SwitchPanelClaimBehavior SwitchPanelClaimBehavior `json:"switch_panel_claim_behavior"`
}

var defaultClaimSettings = ClaimSettings{
	SwitchPanelClaimBehavior: SwitchPanelAutoUnclaim,
}

type ClaimSettingsTable struct {
	*pgxpool.Pool
}

func newClaimSettingsTable(db *pgxpool.Pool) *ClaimSettingsTable {
	return &ClaimSettingsTable{db}
}

func (c ClaimSettingsTable) Schema() string {
	return `
CREATE TABLE IF NOT EXISTS claim_settings(
	"guild_id" int8 NOT NULL,
	"switch_panel_claim_behavior" int2 NOT NULL DEFAULT 0,
	PRIMARY KEY("guild_id")
);
`
}

func (c *ClaimSettingsTable) Get(ctx context.Context, guildId uint64) (settings ClaimSettings, e error) {
	query := `SELECT "switch_panel_claim_behavior" FROM claim_settings WHERE "guild_id" = $1;`
	if err := c.QueryRow(ctx, query, guildId).Scan(&settings.SwitchPanelClaimBehavior); err != nil {
		if err == pgx.ErrNoRows {
			settings = defaultClaimSettings
		} else {
			e = err
		}
	}
	return
}

func (c *ClaimSettingsTable) Set(ctx context.Context, guildId uint64, settings ClaimSettings) (err error) {
	query := `
INSERT INTO claim_settings("guild_id", "switch_panel_claim_behavior") VALUES($1, $2)
ON CONFLICT("guild_id") DO UPDATE SET "switch_panel_claim_behavior" = $2;`

	_, err = c.Exec(ctx, query, guildId, settings.SwitchPanelClaimBehavior)
	return
}
