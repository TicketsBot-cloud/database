package database

import (
	"context"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type PanelHereMention struct {
	*pgxpool.Pool
}

func newPanelHereMention(db *pgxpool.Pool) *PanelHereMention {
	return &PanelHereMention{
		db,
	}
}

func (p PanelHereMention) Schema() string {
	return `
CREATE TABLE IF NOT EXISTS panel_here_mentions(
	"panel_id" int NOT NULL,
	"should_mention_here" bool NOT NULL,
	FOREIGN KEY("panel_id") REFERENCES panels("panel_id") ON DELETE CASCADE ON UPDATE CASCADE,
	PRIMARY KEY("panel_id")
);
`
}

func (p *PanelHereMention) ShouldMentionHere(ctx context.Context, panelId int) (shouldMention bool, e error) {
	query := `SELECT "should_mention_here" from panel_here_mentions WHERE "panel_id"=$1;`

	if err := p.QueryRow(ctx, query, panelId).Scan(&shouldMention); err != nil && err != pgx.ErrNoRows {
		e = err
	}

	return
}

func (p *PanelHereMention) Set(ctx context.Context, panelId int, shouldMentionHere bool) error {
	tx, err := p.Begin(ctx)
	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)

	if err := p.SetWithTx(ctx, tx, panelId, shouldMentionHere); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (p *PanelHereMention) SetWithTx(ctx context.Context, tx pgx.Tx, panelId int, shouldMentionHere bool) (err error) {
	query := `INSERT INTO panel_here_mentions("panel_id", "should_mention_here") VALUES($1, $2) ON CONFLICT("panel_id") DO UPDATE SET "should_mention_here" = $2;`
	_, err = tx.Exec(ctx, query, panelId, shouldMentionHere)
	return
}
