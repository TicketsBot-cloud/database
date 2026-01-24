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
	SupportCanView            bool                     `json:"support_can_view"`
	SupportCanType            bool                     `json:"support_can_type"`
	SwitchPanelClaimBehavior  SwitchPanelClaimBehavior `json:"switch_panel_claim_behavior"`
}

var defaultClaimSettings = ClaimSettings{
	SupportCanView:           true,
	SupportCanType:           false,
	SwitchPanelClaimBehavior: SwitchPanelAutoUnclaim,
}

type ClaimSettingsTable struct {
	*pgxpool.Pool
}

func newClaimSettingsTable(db *pgxpool.Pool) *ClaimSettingsTable {
	return &ClaimSettingsTable{
		db,
	}
}

func (c ClaimSettingsTable) Schema() string {
	return `
CREATE TABLE IF NOT EXISTS claim_settings(
	"guild_id" int8 NOT NULL,
	"support_can_view" bool NOT NULL,
	"support_can_type" bool NOT NULL,
	"switch_panel_claim_behavior" int2 NOT NULL DEFAULT 0,
	PRIMARY KEY("guild_id")
);
`
}

func (c *ClaimSettingsTable) Get(ctx context.Context, guildId uint64) (settings ClaimSettings, e error) {
	query := `SELECT "support_can_view", "support_can_type", "switch_panel_claim_behavior" FROM claim_settings WHERE "guild_id" = $1;`
	if err := c.QueryRow(ctx, query, guildId).Scan(&settings.SupportCanView, &settings.SupportCanType, &settings.SwitchPanelClaimBehavior); err != nil {
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
INSERT INTO claim_settings("guild_id", "support_can_view", "support_can_type", "switch_panel_claim_behavior") VALUES($1, $2, $3, $4)
	ON CONFLICT("guild_id") DO UPDATE SET
	"support_can_view" = $2,
	"support_can_type" = $3,
	"switch_panel_claim_behavior" = $4;`

	_, err = c.Exec(ctx, query, guildId, settings.SupportCanView, settings.SupportCanType, settings.SwitchPanelClaimBehavior)
	return
}

func (c *ClaimSettingsTable) SetSwitchPanelClaimBehavior(ctx context.Context, guildId uint64, behavior SwitchPanelClaimBehavior) (err error) {
	query := `
INSERT INTO claim_settings("guild_id", "support_can_view", "support_can_type", "switch_panel_claim_behavior")
VALUES($1, $2, $3, $4)
ON CONFLICT("guild_id") DO UPDATE SET "switch_panel_claim_behavior" = $4;`

	defaults := defaultClaimSettings
	_, err = c.Exec(ctx, query, guildId, defaults.SupportCanView, defaults.SupportCanType, behavior)
	return
}
