package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type TicketMessageCounts struct {
	*pgxpool.Pool
}

func newTicketMessageCounts(db *pgxpool.Pool) *TicketMessageCounts {
	return &TicketMessageCounts{db}
}

func (t TicketMessageCounts) Schema() string {
	return `
CREATE TABLE IF NOT EXISTS ticket_message_counts(
	"guild_id" int8 NOT NULL,
	"ticket_id" int4 NOT NULL,
	"staff_messages" int4 NOT NULL DEFAULT 0,
	"user_messages" int4 NOT NULL DEFAULT 0,
	FOREIGN KEY("guild_id", "ticket_id") REFERENCES tickets("guild_id", "id"),
	PRIMARY KEY("guild_id", "ticket_id")
);
CREATE INDEX IF NOT EXISTS ticket_message_counts_guild_id ON ticket_message_counts("guild_id");
`
}

type MessageCounts struct {
	GuildId       uint64 `json:"guild_id"`
	TicketId      int    `json:"ticket_id"`
	StaffMessages int    `json:"staff_messages"`
	UserMessages  int    `json:"user_messages"`
}

func (t *TicketMessageCounts) Get(ctx context.Context, guildId uint64, ticketId int) (counts MessageCounts, e error) {
	query := `SELECT "guild_id", "ticket_id", "staff_messages", "user_messages" FROM ticket_message_counts WHERE "guild_id" = $1 AND "ticket_id" = $2;`
	if err := t.QueryRow(ctx, query, guildId, ticketId).Scan(&counts.GuildId, &counts.TicketId, &counts.StaffMessages, &counts.UserMessages); err != nil && err != pgx.ErrNoRows {
		e = err
	}

	return
}

func (t *TicketMessageCounts) IncrementStaffMessages(ctx context.Context, guildId uint64, ticketId int) (err error) {
	query := `
INSERT INTO ticket_message_counts("guild_id", "ticket_id", "staff_messages", "user_messages")
VALUES($1, $2, 1, 0)
ON CONFLICT("guild_id", "ticket_id") DO UPDATE SET "staff_messages" = ticket_message_counts."staff_messages" + 1;`
	_, err = t.Exec(ctx, query, guildId, ticketId)
	return
}

func (t *TicketMessageCounts) IncrementUserMessages(ctx context.Context, guildId uint64, ticketId int) (err error) {
	query := `
INSERT INTO ticket_message_counts("guild_id", "ticket_id", "staff_messages", "user_messages")
VALUES($1, $2, 0, 1)
ON CONFLICT("guild_id", "ticket_id") DO UPDATE SET "user_messages" = ticket_message_counts."user_messages" + 1;`
	_, err = t.Exec(ctx, query, guildId, ticketId)
	return
}

func (t *TicketMessageCounts) Set(ctx context.Context, guildId uint64, ticketId int, staffMessages, userMessages int) (err error) {
	query := `
INSERT INTO ticket_message_counts("guild_id", "ticket_id", "staff_messages", "user_messages")
VALUES($1, $2, $3, $4)
ON CONFLICT("guild_id", "ticket_id") DO UPDATE SET "staff_messages" = $3, "user_messages" = $4;`
	_, err = t.Exec(ctx, query, guildId, ticketId, staffMessages, userMessages)
	return
}

type AverageMessageCounts struct {
	AvgStaffMessages *float64 `json:"avg_staff_messages"`
	AvgUserMessages  *float64 `json:"avg_user_messages"`
	AvgTotalMessages *float64 `json:"avg_total_messages"`
}

// GetAverageMessageCounts is the worker-compatible wrapper. Delegates to
// GetAverageMessageCountsFiltered with no panel filter.
func (t *TicketMessageCounts) GetAverageMessageCounts(ctx context.Context, guildId uint64, days int) (AverageMessageCounts, error) {
	return t.GetAverageMessageCountsFiltered(ctx, guildId, days, nil)
}

func (t *TicketMessageCounts) GetAverageMessageCountsFiltered(ctx context.Context, guildId uint64, days int, filter *PanelFilter) (avg AverageMessageCounts, e error) {
	query := `
SELECT
    AVG(tmc."staff_messages"),
    AVG(tmc."user_messages"),
    AVG(tmc."staff_messages" + tmc."user_messages")
FROM ticket_message_counts tmc
INNER JOIN tickets t ON tmc."guild_id" = t."guild_id" AND tmc."ticket_id" = t."id"
WHERE tmc."guild_id" = $1 AND t."open" = false
    AND ($2 = 0 OR t."open_time" > NOW() - make_interval(days => $2))` +
		PanelPredicate("t", 3, 4)

	arr, unassigned, err := filter.Args()
	if err != nil {
		return avg, fmt.Errorf("average message counts: %w", err)
	}

	if err := t.QueryRow(ctx, query, guildId, days, arr, unassigned).Scan(&avg.AvgStaffMessages, &avg.AvgUserMessages, &avg.AvgTotalMessages); err != nil && err != pgx.ErrNoRows {
		e = err
	}
	return
}

type OneTouchResolution struct {
	OneTouchCount int `json:"one_touch_count"`
	TotalClosed   int `json:"total_closed"`
}

// GetOneTouchResolutionRate is the worker-compatible wrapper. Delegates to
// GetOneTouchResolutionRateFiltered with no panel filter.
func (t *TicketMessageCounts) GetOneTouchResolutionRate(ctx context.Context, guildId uint64, days int) (OneTouchResolution, error) {
	return t.GetOneTouchResolutionRateFiltered(ctx, guildId, days, nil)
}

func (t *TicketMessageCounts) GetOneTouchResolutionRateFiltered(ctx context.Context, guildId uint64, days int, filter *PanelFilter) (result OneTouchResolution, e error) {
	query := `
SELECT
    COUNT(*) FILTER (WHERE tmc."staff_messages" = 1),
    COUNT(*)
FROM ticket_message_counts tmc
INNER JOIN tickets t ON tmc."guild_id" = t."guild_id" AND tmc."ticket_id" = t."id"
WHERE tmc."guild_id" = $1 AND t."open" = false
    AND ($2 = 0 OR t."open_time" > NOW() - make_interval(days => $2))` +
		PanelPredicate("t", 3, 4)

	arr, unassigned, err := filter.Args()
	if err != nil {
		return result, fmt.Errorf("one touch resolution: %w", err)
	}

	if err := t.QueryRow(ctx, query, guildId, days, arr, unassigned).Scan(&result.OneTouchCount, &result.TotalClosed); err != nil && err != pgx.ErrNoRows {
		e = err
	}
	return
}
