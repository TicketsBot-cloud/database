package database

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type PanelCloseReasons struct {
	Reasons     []string `json:"reasons"`
	AllowCustom bool     `json:"allow_custom"`
}

type PanelCloseReasonsTable struct {
	*pgxpool.Pool
}

func newPanelCloseReasonsTable(db *pgxpool.Pool) *PanelCloseReasonsTable {
	return &PanelCloseReasonsTable{db}
}

func DefaultPanelCloseReasons() PanelCloseReasons {
	return PanelCloseReasons{
		Reasons:     []string{},
		AllowCustom: true,
	}
}

func (r PanelCloseReasons) Match(reason string) (string, bool) {
	reason = strings.TrimSpace(reason)
	for _, preset := range r.Reasons {
		if strings.EqualFold(preset, reason) {
			return preset, true
		}
	}

	return "", false
}

func (r PanelCloseReasons) Resolve(reason string) (string, bool) {
	if strings.TrimSpace(reason) == "" || len(r.Reasons) == 0 {
		return reason, true
	}

	if preset, ok := r.Match(reason); ok {
		return preset, true
	}

	return reason, r.AllowCustom
}

func (PanelCloseReasonsTable) Schema() string {
	return `
CREATE TABLE IF NOT EXISTS panel_close_reasons(
	"panel_id" int NOT NULL,
	"reasons" text[] NOT NULL DEFAULT '{}',
	"allow_custom" bool NOT NULL DEFAULT true,
	FOREIGN KEY("panel_id") REFERENCES panels("panel_id") ON DELETE CASCADE ON UPDATE CASCADE,
	PRIMARY KEY("panel_id")
);
`
}

func (t *PanelCloseReasonsTable) Get(ctx context.Context, panelId int) (PanelCloseReasons, error) {
	query := `SELECT "reasons", "allow_custom" FROM panel_close_reasons WHERE "panel_id" = $1;`

	reasons := DefaultPanelCloseReasons()
	if err := t.QueryRow(ctx, query, panelId).Scan(&reasons.Reasons, &reasons.AllowCustom); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return PanelCloseReasons{}, err
	}

	return reasons, nil
}

func (t *PanelCloseReasonsTable) GetByTicket(ctx context.Context, guildId uint64, ticketId int) (PanelCloseReasons, error) {
	query := `
SELECT COALESCE(panel_close_reasons."reasons", '{}'), COALESCE(panel_close_reasons."allow_custom", true)
FROM tickets
LEFT JOIN panel_close_reasons ON panel_close_reasons."panel_id" = tickets."panel_id"
WHERE tickets."guild_id" = $1 AND tickets."id" = $2;`

	reasons := DefaultPanelCloseReasons()
	if err := t.QueryRow(ctx, query, guildId, ticketId).Scan(&reasons.Reasons, &reasons.AllowCustom); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return PanelCloseReasons{}, err
	}

	return reasons, nil
}

func (t *PanelCloseReasonsTable) GetAllForGuild(ctx context.Context, guildId uint64) (map[int]PanelCloseReasons, error) {
	query := `
SELECT panel_close_reasons."panel_id", panel_close_reasons."reasons", panel_close_reasons."allow_custom"
FROM panel_close_reasons
INNER JOIN panels ON panels."panel_id" = panel_close_reasons."panel_id"
WHERE panels."guild_id" = $1;`

	rows, err := t.Query(ctx, query, guildId)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	reasons := make(map[int]PanelCloseReasons)
	for rows.Next() {
		var panelId int
		var data PanelCloseReasons
		if err := rows.Scan(&panelId, &data.Reasons, &data.AllowCustom); err != nil {
			return nil, err
		}

		reasons[panelId] = data
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return reasons, nil
}

func (t *PanelCloseReasonsTable) Set(ctx context.Context, panelId int, data PanelCloseReasons) error {
	tx, err := t.Begin(ctx)
	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)

	if err := t.SetWithTx(ctx, tx, panelId, data); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (t *PanelCloseReasonsTable) SetWithTx(ctx context.Context, tx pgx.Tx, panelId int, data PanelCloseReasons) error {
	query := `
INSERT INTO panel_close_reasons("panel_id", "reasons", "allow_custom")
VALUES($1, $2, $3)
ON CONFLICT("panel_id") DO UPDATE SET "reasons" = $2, "allow_custom" = $3;`

	_, err := tx.Exec(ctx, query, panelId, toTextArray(data.Reasons), data.AllowCustom)
	return err
}
