package database

import (
	"context"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type TicketLabel struct {
	GuildId uint64 `json:"guild_id"`
	LabelId int    `json:"label_id"`
	Name    string `json:"name"`
	Colour  int32  `json:"colour"`
}

type TicketLabelsTable struct {
	*pgxpool.Pool
}

func newTicketLabelsTable(db *pgxpool.Pool) *TicketLabelsTable {
	return &TicketLabelsTable{
		db,
	}
}

func (t TicketLabelsTable) Schema() string {
	return `
CREATE TABLE IF NOT EXISTS ticket_labels(
	"guild_id" int8 NOT NULL,
	"label_id" SERIAL,
	"name" varchar(32) NOT NULL,
	"colour" int4 NOT NULL DEFAULT 4869178,
	PRIMARY KEY("guild_id", "label_id"),
	UNIQUE("guild_id", "name")
);
CREATE INDEX IF NOT EXISTS ticket_labels_guild_id_idx ON ticket_labels("guild_id");
`
}

func (t *TicketLabelsTable) GetByGuild(ctx context.Context, guildId uint64) ([]TicketLabel, error) {
	query := `SELECT "guild_id", "label_id", "name", "colour" FROM ticket_labels WHERE "guild_id" = $1 ORDER BY "label_id" ASC;`

	rows, err := t.Query(ctx, query, guildId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var labels []TicketLabel
	for rows.Next() {
		var label TicketLabel
		if err := rows.Scan(&label.GuildId, &label.LabelId, &label.Name, &label.Colour); err != nil {
			return nil, err
		}
		labels = append(labels, label)
	}

	return labels, nil
}

func (t *TicketLabelsTable) Get(ctx context.Context, guildId uint64, labelId int) (TicketLabel, bool, error) {
	query := `SELECT "guild_id", "label_id", "name", "colour" FROM ticket_labels WHERE "guild_id" = $1 AND "label_id" = $2;`

	var label TicketLabel
	err := t.QueryRow(ctx, query, guildId, labelId).Scan(&label.GuildId, &label.LabelId, &label.Name, &label.Colour)
	if err != nil {
		if err == pgx.ErrNoRows {
			return TicketLabel{}, false, nil
		}
		return TicketLabel{}, false, err
	}

	return label, true, nil
}

func (t *TicketLabelsTable) Create(ctx context.Context, guildId uint64, name string, colour int32) (int, error) {
	query := `INSERT INTO ticket_labels("guild_id", "name", "colour") VALUES($1, $2, $3) RETURNING "label_id";`

	var labelId int
	err := t.QueryRow(ctx, query, guildId, name, colour).Scan(&labelId)
	return labelId, err
}

func (t *TicketLabelsTable) Update(ctx context.Context, guildId uint64, labelId int, name string, colour int32) error {
	query := `UPDATE ticket_labels SET "name" = $3, "colour" = $4 WHERE "guild_id" = $1 AND "label_id" = $2;`
	_, err := t.Exec(ctx, query, guildId, labelId, name, colour)
	return err
}

func (t *TicketLabelsTable) Delete(ctx context.Context, guildId uint64, labelId int) error {
	query := `DELETE FROM ticket_labels WHERE "guild_id" = $1 AND "label_id" = $2;`
	_, err := t.Exec(ctx, query, guildId, labelId)
	return err
}

func (t *TicketLabelsTable) GetCount(ctx context.Context, guildId uint64) (int, error) {
	query := `SELECT COUNT(*) FROM ticket_labels WHERE "guild_id" = $1;`

	var count int
	err := t.QueryRow(ctx, query, guildId).Scan(&count)
	return count, err
}
