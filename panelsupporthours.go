package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type PanelSupportHours struct {
	Id        int       `json:"id"`
	PanelId   int       `json:"panel_id"`
	DayOfWeek int       `json:"day_of_week"` // 0 = Sunday, 1 = Monday, ..., 6 = Saturday
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Enabled   bool      `json:"enabled"`
}

type PanelSupportHoursTable struct {
	*pgxpool.Pool
}

func newPanelSupportHoursTable(db *pgxpool.Pool) *PanelSupportHoursTable {
	return &PanelSupportHoursTable{db}
}

func (p PanelSupportHoursTable) Schema() string {
	return `
CREATE TABLE IF NOT EXISTS panel_support_hours (
    "id" SERIAL PRIMARY KEY,
    "panel_id" INTEGER NOT NULL,
    "day_of_week" INTEGER NOT NULL CHECK (day_of_week >= 0 AND day_of_week <= 6),
    "start_time" TIME NOT NULL,
    "end_time" TIME NOT NULL,
    "enabled" BOOLEAN NOT NULL DEFAULT true,
    FOREIGN KEY ("panel_id") REFERENCES panels("panel_id") ON DELETE CASCADE,
    UNIQUE("panel_id", "day_of_week")
);

CREATE INDEX IF NOT EXISTS panel_support_hours_panel_id ON panel_support_hours("panel_id");
CREATE INDEX IF NOT EXISTS panel_support_hours_enabled ON panel_support_hours("panel_id", "enabled");`
}

// GetByPanelId retrieves all support hours for a specific panel
func (p *PanelSupportHoursTable) GetByPanelId(ctx context.Context, panelId int) ([]PanelSupportHours, error) {
	query := `
SELECT
    "id",
    "panel_id",
    "day_of_week",
    "start_time",
    "end_time",
    "enabled"
FROM panel_support_hours
WHERE "panel_id" = $1
ORDER BY "day_of_week" ASC;`

	rows, err := p.Query(ctx, query, panelId)
	defer rows.Close()
	if err != nil {
		return nil, err
	}

	var supportHours []PanelSupportHours
	for rows.Next() {
		var sh PanelSupportHours
		if err := rows.Scan(
			&sh.Id,
			&sh.PanelId,
			&sh.DayOfWeek,
			&sh.StartTime,
			&sh.EndTime,
			&sh.Enabled,
		); err != nil {
			return nil, err
		}
		supportHours = append(supportHours, sh)
	}

	return supportHours, nil
}

// Upsert creates or updates support hours for a specific panel and day
func (p *PanelSupportHoursTable) Upsert(ctx context.Context, supportHours PanelSupportHours) (int, error) {
	query := `
INSERT INTO panel_support_hours (
    "panel_id",
    "day_of_week",
    "start_time",
    "end_time",
    "enabled"
) VALUES ($1, $2, $3, $4, $5)
ON CONFLICT ("panel_id", "day_of_week")
DO UPDATE SET
    "start_time" = EXCLUDED."start_time",
    "end_time" = EXCLUDED."end_time",
    "enabled" = EXCLUDED."enabled"
RETURNING "id";`

	var id int
	err := p.QueryRow(ctx, query,
		supportHours.PanelId,
		supportHours.DayOfWeek,
		supportHours.StartTime,
		supportHours.EndTime,
		supportHours.Enabled,
	).Scan(&id)

	return id, err
}

// UpsertBatch creates or updates multiple support hours entries
func (p *PanelSupportHoursTable) UpsertBatch(ctx context.Context, panelId int, supportHoursList []PanelSupportHours) error {
	tx, err := p.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, sh := range supportHoursList {
		sh.PanelId = panelId
		if _, err := p.UpsertWithTx(ctx, tx, sh); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// UpsertWithTx creates or updates support hours within a transaction
func (p *PanelSupportHoursTable) UpsertWithTx(ctx context.Context, tx pgx.Tx, supportHours PanelSupportHours) (int, error) {
	query := `
INSERT INTO panel_support_hours (
    "panel_id",
    "day_of_week",
    "start_time",
    "end_time",
    "enabled"
) VALUES ($1, $2, $3, $4, $5)
ON CONFLICT ("panel_id", "day_of_week")
DO UPDATE SET
    "start_time" = EXCLUDED."start_time",
    "end_time" = EXCLUDED."end_time",
    "enabled" = EXCLUDED."enabled"
RETURNING "id";`

	var id int
	err := tx.QueryRow(ctx, query,
		supportHours.PanelId,
		supportHours.DayOfWeek,
		supportHours.StartTime,
		supportHours.EndTime,
		supportHours.Enabled,
	).Scan(&id)

	return id, err
}

// DeleteByPanelId removes all support hours for a specific panel
func (p *PanelSupportHoursTable) DeleteByPanelId(ctx context.Context, panelId int) error {
	query := `DELETE FROM panel_support_hours WHERE "panel_id" = $1;`
	_, err := p.Exec(ctx, query, panelId)
	return err
}

// DeleteByPanelIdWithTx removes all support hours for a specific panel within a transaction
func (p *PanelSupportHoursTable) DeleteByPanelIdWithTx(ctx context.Context, tx pgx.Tx, panelId int) error {
	query := `DELETE FROM panel_support_hours WHERE "panel_id" = $1;`
	_, err := tx.Exec(ctx, query, panelId)
	return err
}

// IsActiveNow checks if the panel is currently within support hours
func (p *PanelSupportHoursTable) IsActiveNow(ctx context.Context, panelId int) (bool, error) {
	now := time.Now()
	dayOfWeek := int(now.Weekday())
	currentTime := now.Format("15:04:05")

	return p.IsActive(ctx, panelId, dayOfWeek, currentTime)
}

// IsActive checks if the panel is active at a specific day and time
func (p *PanelSupportHoursTable) IsActive(ctx context.Context, panelId int, dayOfWeek int, timeStr string) (bool, error) {
	query := `
SELECT EXISTS(
    SELECT 1
    FROM panel_support_hours
    WHERE "panel_id" = $1
    AND "day_of_week" = $2
    AND "enabled" = true
    AND $3::TIME BETWEEN "start_time" AND "end_time"
) AS is_active;`

	var isActive bool
	err := p.QueryRow(ctx, query, panelId, dayOfWeek, timeStr).Scan(&isActive)
	return isActive, err
}

// GetActivePanels returns all panels that are currently within support hours
func (p *PanelSupportHoursTable) GetActivePanels(ctx context.Context, guildId uint64) ([]int, error) {
	now := time.Now()
	dayOfWeek := int(now.Weekday())
	currentTime := now.Format("15:04:05")

	query := `
SELECT DISTINCT psh.panel_id
FROM panel_support_hours psh
INNER JOIN panels p ON p.panel_id = psh.panel_id
WHERE p.guild_id = $1
AND psh.day_of_week = $2
AND psh.enabled = true
AND $3::TIME BETWEEN psh.start_time AND psh.end_time;`

	rows, err := p.Query(ctx, query, guildId, dayOfWeek, currentTime)
	defer rows.Close()
	if err != nil {
		return nil, err
	}

	var panelIds []int
	for rows.Next() {
		var panelId int
		if err := rows.Scan(&panelId); err != nil {
			return nil, err
		}
		panelIds = append(panelIds, panelId)
	}

	return panelIds, nil
}

// HasSupportHours checks if a panel has any support hours configured
func (p *PanelSupportHoursTable) HasSupportHours(ctx context.Context, panelId int) (bool, error) {
	query := `
SELECT EXISTS(
    SELECT 1
    FROM panel_support_hours
    WHERE "panel_id" = $1
) AS has_hours;`

	var hasHours bool
	err := p.QueryRow(ctx, query, panelId).Scan(&hasHours)
	return hasHours, err
}
