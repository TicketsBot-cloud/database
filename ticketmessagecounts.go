package database

import (
	"context"
	"time"

	"github.com/jackc/pgtype"
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

func (t *TicketMessageCounts) GetAverageMessageCounts(ctx context.Context, guildId uint64, days int) (avg AverageMessageCounts, e error) {
	if days == 0 {
		query := `
SELECT
    AVG(tmc."staff_messages"),
    AVG(tmc."user_messages"),
    AVG(tmc."staff_messages" + tmc."user_messages")
FROM ticket_message_counts tmc
INNER JOIN tickets t ON tmc."guild_id" = t."guild_id" AND tmc."ticket_id" = t."id"
WHERE tmc."guild_id" = $1 AND t."open" = false;`
		if err := t.QueryRow(ctx, query, guildId).Scan(&avg.AvgStaffMessages, &avg.AvgUserMessages, &avg.AvgTotalMessages); err != nil && err != pgx.ErrNoRows {
			e = err
		}
		return
	}

	parsedInterval := pgtype.Interval{}
	if err := parsedInterval.Set(time.Duration(days) * 24 * time.Hour); err != nil {
		return avg, err
	}

	query := `
SELECT
    AVG(tmc."staff_messages"),
    AVG(tmc."user_messages"),
    AVG(tmc."staff_messages" + tmc."user_messages")
FROM ticket_message_counts tmc
INNER JOIN tickets t ON tmc."guild_id" = t."guild_id" AND tmc."ticket_id" = t."id"
WHERE tmc."guild_id" = $1 AND t."open" = false AND t."open_time" > NOW() - $2::interval;`
	if err := t.QueryRow(ctx, query, guildId, parsedInterval).Scan(&avg.AvgStaffMessages, &avg.AvgUserMessages, &avg.AvgTotalMessages); err != nil && err != pgx.ErrNoRows {
		e = err
	}

	return
}

type OneTouchResolution struct {
	OneTouchCount int `json:"one_touch_count"`
	TotalClosed   int `json:"total_closed"`
}

func (t *TicketMessageCounts) GetOneTouchResolutionRate(ctx context.Context, guildId uint64, days int) (result OneTouchResolution, e error) {
	if days == 0 {
		query := `
SELECT
    COUNT(*) FILTER (WHERE tmc."staff_messages" = 1),
    COUNT(*)
FROM ticket_message_counts tmc
INNER JOIN tickets t ON tmc."guild_id" = t."guild_id" AND tmc."ticket_id" = t."id"
WHERE tmc."guild_id" = $1 AND t."open" = false;`
		if err := t.QueryRow(ctx, query, guildId).Scan(&result.OneTouchCount, &result.TotalClosed); err != nil && err != pgx.ErrNoRows {
			e = err
		}
		return
	}

	parsedInterval := pgtype.Interval{}
	if err := parsedInterval.Set(time.Duration(days) * 24 * time.Hour); err != nil {
		return result, err
	}

	query := `
SELECT
    COUNT(*) FILTER (WHERE tmc."staff_messages" = 1),
    COUNT(*)
FROM ticket_message_counts tmc
INNER JOIN tickets t ON tmc."guild_id" = t."guild_id" AND tmc."ticket_id" = t."id"
WHERE tmc."guild_id" = $1 AND t."open" = false AND t."open_time" > NOW() - $2::interval;`
	if err := t.QueryRow(ctx, query, guildId, parsedInterval).Scan(&result.OneTouchCount, &result.TotalClosed); err != nil && err != pgx.ErrNoRows {
		e = err
	}

	return
}
