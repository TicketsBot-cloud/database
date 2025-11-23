package database

import (
	"context"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type PanelSupportHoursSettings struct {
	PanelId             int    `json:"panel_id"`
	OutOfHoursBehaviour string `json:"out_of_hours_behaviour"`
	OutOfHoursMessage   string `json:"out_of_hours_message"`
}

type PanelSupportHoursSettingsTable struct {
	*pgxpool.Pool
}

func newPanelSupportHoursSettingsTable(db *pgxpool.Pool) *PanelSupportHoursSettingsTable {
	return &PanelSupportHoursSettingsTable{db}
}

func (p PanelSupportHoursSettingsTable) Schema() string {
	return `
	CREATE TABLE IF NOT EXISTS panel_support_hours_settings (
		"panel_id" INTEGER PRIMARY KEY,
		"out_of_hours_behaviour" VARCHAR(50) NOT NULL DEFAULT 'block_creation',
		"out_of_hours_message" TEXT NOT NULL DEFAULT 'Our support team is currently offline. Please try again during our support hours.',
		FOREIGN KEY ("panel_id") REFERENCES panels("panel_id") ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS panel_support_hours_settings_panel_id ON panel_support_hours_settings("panel_id");
`
}

// GetByPanelId retrieves all support hours settings for a specific panel
func (p *PanelSupportHoursSettingsTable) GetByPanelId(ctx context.Context, panelId int) (*PanelSupportHoursSettings, error) {
	query := `
		SELECT
			"out_of_hours_behaviour",
			"out_of_hours_message"
		FROM panel_support_hours_settings
		WHERE "panel_id" = $1;`

	rows, err := p.Query(ctx, query, panelId)
	defer rows.Close()
	if err != nil {
		return nil, err
	}

	var supportHoursSettings PanelSupportHoursSettings
	for rows.Next() {
		if err := rows.Scan(
			&supportHoursSettings.OutOfHoursBehaviour,
			&supportHoursSettings.OutOfHoursMessage,
		); err != nil {
			return nil, err
		}
	}

	return &supportHoursSettings, nil
}

// Upsert creates or updates support hours settings for a panel
func (p *PanelSupportHoursSettingsTable) Upsert(ctx context.Context, supportHoursSettings PanelSupportHoursSettings) error {
	query := `
INSERT INTO panel_support_hours_settings (
    "panel_id",
    "out_of_hours_behaviour",
	"out_of_hours_message"
) VALUES ($1, $2, $3)
ON CONFLICT ("panel_id")
DO UPDATE SET
	"out_of_hours_behaviour" = EXCLUDED."out_of_hours_behaviour",
	"out_of_hours_message" = EXCLUDED."out_of_hours_message";`

	_, err := p.Exec(ctx, query,
		supportHoursSettings.PanelId,
		supportHoursSettings.OutOfHoursBehaviour,
		supportHoursSettings.OutOfHoursMessage,
	)
	return err
}

// UpsertWithTx creates or updates support hours settings for a panel within a transaction
func (p *PanelSupportHoursSettingsTable) UpsertWithTx(ctx context.Context, tx pgx.Tx, supportHoursSettings PanelSupportHoursSettings) error {
	query := `
INSERT INTO panel_support_hours_settings (
	"panel_id",
	"out_of_hours_behaviour",
	"out_of_hours_message"
) VALUES ($1, $2, $3)
ON CONFLICT ("panel_id")
DO UPDATE SET
	"out_of_hours_behaviour" = EXCLUDED."out_of_hours_behaviour",
	"out_of_hours_message" = EXCLUDED."out_of_hours_message";`

	_, err := tx.Exec(ctx, query,
		supportHoursSettings.PanelId,
		supportHoursSettings.OutOfHoursBehaviour,
		supportHoursSettings.OutOfHoursMessage,
	)

	return err
}

// DeleteByPanelId removes all support hours settings for a specific panel
func (p *PanelSupportHoursSettingsTable) DeleteByPanelId(ctx context.Context, panelId int) error {
	query := `DELETE FROM panel_support_hours_settings WHERE "panel_id" = $1;`
	_, err := p.Exec(ctx, query, panelId)
	return err
}

// DeleteByPanelIdWithTx removes all support hours settings for a specific panel within a transaction
func (p *PanelSupportHoursSettingsTable) DeleteByPanelIdWithTx(ctx context.Context, tx pgx.Tx, panelId int) error {
	query := `DELETE FROM panel_support_hours_settings WHERE "panel_id" = $1;`
	_, err := tx.Exec(ctx, query, panelId)
	return err
}
