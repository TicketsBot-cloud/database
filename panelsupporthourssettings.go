package database

import (
	"context"

	"github.com/jackc/pgx/v4/pgxpool"
)

type OutOfHoursBehaviour string

const (
	OutOfHoursBehaviourBlockCreation   OutOfHoursBehaviour = "block_creation"
	OutOfHoursBehaviourAllowWithWarning OutOfHoursBehaviour = "allow_with_warning"
)

type PanelSupportHoursSettings struct {
	PanelId             int                 `json:"panel_id"`
	OutOfHoursBehaviour OutOfHoursBehaviour `json:"out_of_hours_behaviour"`
	OutOfHoursMessage   string              `json:"out_of_hours_message"`
}

type PanelSupportHoursSettingsTable struct {
	*pgxpool.Pool
}

func newPanelSupportHoursSettingsTable(db *pgxpool.Pool) *PanelSupportHoursSettingsTable {
	return &PanelSupportHoursSettingsTable{db}
}

func (t PanelSupportHoursSettingsTable) Schema() string {
	return `
CREATE TABLE IF NOT EXISTS panel_support_hours_settings (
    "panel_id" INTEGER NOT NULL PRIMARY KEY,
    "out_of_hours_behaviour" VARCHAR(50) NOT NULL DEFAULT 'block_creation',
    "out_of_hours_message" TEXT NOT NULL DEFAULT '',
    FOREIGN KEY ("panel_id") REFERENCES panels("panel_id") ON DELETE CASCADE
);`
}

// Get retrieves the support hours settings for a panel. Returns the settings, whether they exist, and any error.
func (t *PanelSupportHoursSettingsTable) Get(ctx context.Context, panelId int) (PanelSupportHoursSettings, bool, error) {
	query := `
SELECT "panel_id", "out_of_hours_behaviour", "out_of_hours_message"
FROM panel_support_hours_settings
WHERE "panel_id" = $1;`

	var settings PanelSupportHoursSettings
	err := t.QueryRow(ctx, query, panelId).Scan(
		&settings.PanelId,
		&settings.OutOfHoursBehaviour,
		&settings.OutOfHoursMessage,
	)

	if err != nil {
		if err.Error() == "no rows in result set" {
			return PanelSupportHoursSettings{}, false, nil
		}
		return PanelSupportHoursSettings{}, false, err
	}

	return settings, true, nil
}

// Set upserts the support hours settings for a panel.
func (t *PanelSupportHoursSettingsTable) Set(ctx context.Context, settings PanelSupportHoursSettings) error {
	query := `
INSERT INTO panel_support_hours_settings ("panel_id", "out_of_hours_behaviour", "out_of_hours_message")
VALUES ($1, $2, $3)
ON CONFLICT ("panel_id")
DO UPDATE SET
    "out_of_hours_behaviour" = EXCLUDED."out_of_hours_behaviour",
    "out_of_hours_message" = EXCLUDED."out_of_hours_message";`

	_, err := t.Exec(ctx, query, settings.PanelId, settings.OutOfHoursBehaviour, settings.OutOfHoursMessage)
	return err
}

// Delete removes the support hours settings for a panel.
func (t *PanelSupportHoursSettingsTable) Delete(ctx context.Context, panelId int) error {
	query := `DELETE FROM panel_support_hours_settings WHERE "panel_id" = $1;`
	_, err := t.Exec(ctx, query, panelId)
	return err
}
