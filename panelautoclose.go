package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type PanelAutoCloseSettings struct {
	Enabled                 bool           `json:"enabled"`
	SinceOpenWithNoResponse *time.Duration `json:"since_open_with_no_response"`
	SinceLastMessage        *time.Duration `json:"since_last_message"`
	OnUserLeave             *bool          `json:"on_user_leave"`
}

type PanelAutoCloseTable struct {
	*pgxpool.Pool
}

func newPanelAutoCloseTable(db *pgxpool.Pool) *PanelAutoCloseTable {
	return &PanelAutoCloseTable{db}
}

func (PanelAutoCloseTable) Schema() string {
	return `
CREATE TABLE IF NOT EXISTS panel_auto_close(
	"panel_id" int NOT NULL,
	"enabled" bool NOT NULL DEFAULT false,
	"since_open_with_no_response" interval,
	"since_last_message" interval,
	"on_user_leave" bool,
	FOREIGN KEY("panel_id") REFERENCES panels("panel_id") ON DELETE CASCADE,
	PRIMARY KEY("panel_id")
);`
}

func (t *PanelAutoCloseTable) Get(ctx context.Context, panelId int) (settings PanelAutoCloseSettings, e error) {
	query := `SELECT "enabled", "since_open_with_no_response", "since_last_message", "on_user_leave" FROM panel_auto_close WHERE "panel_id" = $1;`
	if err := t.QueryRow(ctx, query, panelId).Scan(&settings.Enabled, &settings.SinceOpenWithNoResponse, &settings.SinceLastMessage, &settings.OnUserLeave); err != nil && err != pgx.ErrNoRows {
		e = err
	}
	return
}

func (t *PanelAutoCloseTable) Set(ctx context.Context, panelId int, settings PanelAutoCloseSettings) (err error) {
	query := `
INSERT INTO panel_auto_close("panel_id", "enabled", "since_open_with_no_response", "since_last_message", "on_user_leave")
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT("panel_id") DO UPDATE SET
	"enabled" = $2,
	"since_open_with_no_response" = $3,
	"since_last_message" = $4,
	"on_user_leave" = $5;`

	_, err = t.Exec(ctx, query, panelId, settings.Enabled, settings.SinceOpenWithNoResponse, settings.SinceLastMessage, settings.OnUserLeave)
	return
}

func (t *PanelAutoCloseTable) Reset(ctx context.Context, guildId uint64) (err error) {
	query := `
UPDATE panel_auto_close
SET "since_open_with_no_response" = NULL, "since_last_message" = NULL
WHERE "panel_id" IN (SELECT "panel_id" FROM panels WHERE "guild_id" = $1);`

	_, err = t.Exec(ctx, query, guildId)
	return
}

func (t *PanelAutoCloseTable) Delete(ctx context.Context, panelId int) (err error) {
	_, err = t.Exec(ctx, `DELETE FROM panel_auto_close WHERE "panel_id" = $1;`, panelId)
	return
}
