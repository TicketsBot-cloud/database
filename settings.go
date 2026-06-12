package database

import (
	"context"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Settings struct {
	ContextMenuPermissionLevel  int  `json:"context_menu_permission_level,string"`
	ContextMenuAddSender        bool `json:"context_menu_add_sender"`
	ContextMenuPanel            *int `json:"context_menu_panel"`
	AnonymiseDashboardResponses bool `json:"anonymise_dashboard_responses"`
}

func defaultSettings() Settings {
	return Settings{
		ContextMenuPermissionLevel:  0,
		ContextMenuAddSender:        true,
		ContextMenuPanel:            nil,
		AnonymiseDashboardResponses: false,
	}
}

type SettingsTable struct {
	*pgxpool.Pool
}

func newSettingsTable(db *pgxpool.Pool) *SettingsTable {
	return &SettingsTable{
		db,
	}
}

func (s SettingsTable) Schema() string {
	return `
CREATE TABLE IF NOT EXISTS settings(
	"guild_id" int8 NOT NULL,
	"context_menu_permission_level" int DEFAULT '0',
	"context_menu_add_sender" bool DEFAULT 't',
	"context_menu_panel" int DEFAULT NULL,
	"anonymise_dashboard_responses" bool DEFAULT 'f',
	FOREIGN KEY("context_menu_panel") REFERENCES panels("panel_id") ON DELETE SET NULL,
	PRIMARY KEY("guild_id")
);
`
}

func (s *SettingsTable) Get(ctx context.Context, guildId uint64) (Settings, error) {
	query := `
SELECT
	"context_menu_permission_level",
	"context_menu_add_sender",
	"context_menu_panel",
	"anonymise_dashboard_responses"
FROM settings
WHERE "guild_id" = $1;
`

	var settings Settings
	err := s.QueryRow(ctx, query, guildId).Scan(
		&settings.ContextMenuPermissionLevel,
		&settings.ContextMenuAddSender,
		&settings.ContextMenuPanel,
		&settings.AnonymiseDashboardResponses,
	)

	if err == nil {
		return settings, nil
	} else if err == pgx.ErrNoRows {
		return defaultSettings(), nil
	} else {
		return settings, err
	}
}

func (s *SettingsTable) Set(ctx context.Context, guildId uint64, settings Settings) (err error) {
	query := `
INSERT INTO settings(
	"guild_id",
	"context_menu_permission_level",
	"context_menu_add_sender",
	"context_menu_panel",
	"anonymise_dashboard_responses"
)
VALUES($1, $2, $3, $4, $5)
ON CONFLICT("guild_id")
DO UPDATE SET
	"context_menu_permission_level" = $2,
	"context_menu_add_sender" = $3,
	"context_menu_panel" = $4,
	"anonymise_dashboard_responses" = $5
;
`

	_, err = s.Exec(ctx, query,
		guildId,
		settings.ContextMenuPermissionLevel,
		settings.ContextMenuAddSender,
		settings.ContextMenuPanel,
		settings.AnonymiseDashboardResponses,
	)

	return
}

func (s *SettingsTable) SetContextMenuPermissionLevel(ctx context.Context, guildId uint64, permissionLevel int) (err error) {
	query := `
INSERT INTO settings("guild_id", "context_menu_permission_level")
VALUES($1, $2)
ON CONFLICT("guild_id")
DO UPDATE SET "context_menu_permission_level" = $2;
`

	_, err = s.Exec(ctx, query, guildId, permissionLevel)
	return
}

