package database

import (
	"context"

	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4/pgxpool"
)

type TicketLabelAssignmentsTable struct {
	*pgxpool.Pool
}

func newTicketLabelAssignmentsTable(db *pgxpool.Pool) *TicketLabelAssignmentsTable {
	return &TicketLabelAssignmentsTable{
		db,
	}
}

func (t TicketLabelAssignmentsTable) Schema() string {
	return `
CREATE TABLE IF NOT EXISTS ticket_label_assignments(
	"guild_id" int8 NOT NULL,
	"ticket_id" int4 NOT NULL,
	"label_id" int4 NOT NULL,
	FOREIGN KEY("guild_id", "ticket_id") REFERENCES tickets("guild_id", "id") ON DELETE CASCADE,
	FOREIGN KEY("guild_id", "label_id") REFERENCES ticket_labels("guild_id", "label_id") ON DELETE CASCADE,
	PRIMARY KEY("guild_id", "ticket_id", "label_id")
);
CREATE INDEX IF NOT EXISTS tkla_guild_ticket_idx ON ticket_label_assignments("guild_id", "ticket_id");
CREATE INDEX IF NOT EXISTS tkla_guild_label_idx ON ticket_label_assignments("guild_id", "label_id");
`
}

func (t *TicketLabelAssignmentsTable) GetLabelNameByTicket(ctx context.Context, guildId uint64, ticketId int) (map[int]string, error) {
	query := `SELECT tla."label_id", tl."name" FROM ticket_label_assignments tla JOIN ticket_labels tl ON tla.guild_id = tl.guild_id AND tla.label_id = tl.label_id WHERE tla.guild_id = $1 AND tla.ticket_id = $2;`

	rows, err := t.Query(ctx, query, guildId, ticketId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int]string)
	for rows.Next() {
		var labelId int
		var labelName string
		if err := rows.Scan(&labelId, &labelName); err != nil {
			return nil, err
		}
		result[labelId] = labelName
	}

	return result, nil
}

func (t *TicketLabelAssignmentsTable) GetByTicket(ctx context.Context, guildId uint64, ticketId int) ([]int, error) {
	query := `SELECT "label_id" FROM ticket_label_assignments WHERE "guild_id" = $1 AND "ticket_id" = $2;`

	rows, err := t.Query(ctx, query, guildId, ticketId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var labelIds []int
	for rows.Next() {
		var labelId int
		if err := rows.Scan(&labelId); err != nil {
			return nil, err
		}
		labelIds = append(labelIds, labelId)
	}

	return labelIds, nil
}

func (t *TicketLabelAssignmentsTable) GetByTickets(ctx context.Context, guildId uint64, ticketIds []int) (map[int][]int, error) {
	if len(ticketIds) == 0 {
		return make(map[int][]int), nil
	}

	ticketIdArray := &pgtype.Int4Array{}
	if err := ticketIdArray.Set(ticketIds); err != nil {
		return nil, err
	}

	query := `SELECT "ticket_id", "label_id" FROM ticket_label_assignments WHERE "guild_id" = $1 AND "ticket_id" = ANY($2);`

	rows, err := t.Query(ctx, query, guildId, ticketIdArray)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int][]int)
	for rows.Next() {
		var ticketId, labelId int
		if err := rows.Scan(&ticketId, &labelId); err != nil {
			return nil, err
		}
		result[ticketId] = append(result[ticketId], labelId)
	}

	return result, nil
}

func (t *TicketLabelAssignmentsTable) Add(ctx context.Context, guildId uint64, ticketId, labelId int) error {
	query := `INSERT INTO ticket_label_assignments("guild_id", "ticket_id", "label_id") VALUES($1, $2, $3) ON CONFLICT("guild_id", "ticket_id", "label_id") DO NOTHING;`
	_, err := t.Exec(ctx, query, guildId, ticketId, labelId)
	return err
}

func (t *TicketLabelAssignmentsTable) Delete(ctx context.Context, guildId uint64, ticketId, labelId int) error {
	query := `DELETE FROM ticket_label_assignments WHERE "guild_id" = $1 AND "ticket_id" = $2 AND "label_id" = $3;`
	_, err := t.Exec(ctx, query, guildId, ticketId, labelId)
	return err
}

func (t *TicketLabelAssignmentsTable) Replace(ctx context.Context, guildId uint64, ticketId int, labelIds []int) error {
	tx, err := t.Begin(ctx)
	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)

	// Remove existing assignments
	if _, err := tx.Exec(ctx, `DELETE FROM ticket_label_assignments WHERE "guild_id" = $1 AND "ticket_id" = $2;`, guildId, ticketId); err != nil {
		return err
	}

	// Add new assignments
	for _, labelId := range labelIds {
		query := `INSERT INTO ticket_label_assignments("guild_id", "ticket_id", "label_id") VALUES($1, $2, $3) ON CONFLICT("guild_id", "ticket_id", "label_id") DO NOTHING;`
		if _, err := tx.Exec(ctx, query, guildId, ticketId, labelId); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}
