package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type FirstResponseTime struct {
	*pgxpool.Pool
}

func newFirstResponseTime(db *pgxpool.Pool) *FirstResponseTime {
	return &FirstResponseTime{
		db,
	}
}

func (f FirstResponseTime) Schema() string {
	return `
CREATE TABLE IF NOT EXISTS first_response_time(
	"guild_id" int8 NOT NULL,
	"ticket_id" int4 NOT NULL,
	"user_id" int8 NOT NULL,
	"response_time" interval NOT NULL,
	FOREIGN KEY("guild_id", "ticket_id") REFERENCES tickets("guild_id", "id"),
	PRIMARY KEY("guild_id", "ticket_id")
);
CREATE INDEX IF NOT EXISTS first_response_time_guild_id ON first_response_time("guild_id");
`
}

func (f *FirstResponseTime) HasResponse(ctx context.Context, guildId uint64, ticketId int) (hasResponse bool, e error) {
	query := `SELECT EXISTS(SELECT 1 FROM first_response_time WHERE "guild_id" = $1 AND "ticket_id" = $2);`
	if err := f.QueryRow(ctx, query, guildId, ticketId).Scan(&hasResponse); err != nil && err != pgx.ErrNoRows {
		e = err
	}

	return
}

func (f *FirstResponseTime) GetAverage(ctx context.Context, guildId uint64, interval time.Duration) (responseTime *time.Duration, e error) {
	parsedInterval := pgtype.Interval{}
	if err := parsedInterval.Set(interval); err != nil {
		return nil, err
	}

	query := `
SELECT AVG(first_response_time.response_time)
FROM first_response_time
INNER JOIN tickets
ON first_response_time.guild_id = tickets.guild_id AND first_response_time.ticket_id = tickets.id
WHERE tickets.open_time > NOW() - $1::interval AND first_response_time.guild_id = $2;
`

	if err := f.QueryRow(ctx, query, parsedInterval, guildId).Scan(&responseTime); err != nil && err != pgx.ErrNoRows {
		e = err
	}

	return
}

func (f *FirstResponseTime) GetAverageAllTime(ctx context.Context, guildId uint64) (responseTime *time.Duration, e error) {
	query := `SELECT AVG(response_time) FROM first_response_time WHERE first_response_time.guild_id = $1;`
	if err := f.QueryRow(ctx, query, guildId).Scan(&responseTime); err != nil && err != pgx.ErrNoRows {
		e = err
	}

	return
}

func (f *FirstResponseTime) GetAverageUser(ctx context.Context, guildId, userId uint64, interval time.Duration) (responseTime *time.Duration, e error) {
	parsedInterval := pgtype.Interval{}
	if err := parsedInterval.Set(interval); err != nil {
		return nil, err
	}

	query := `
SELECT AVG(first_response_time.response_time)
FROM first_response_time
INNER JOIN tickets
ON first_response_time.guild_id = tickets.guild_id AND first_response_time.ticket_id = tickets.id
WHERE tickets.open_time > NOW() - $1::interval AND first_response_time.guild_id = $2 AND first_response_time.user_id = $3;`

	if err := f.QueryRow(ctx, query, parsedInterval, guildId, userId).Scan(&responseTime); err != nil && err != pgx.ErrNoRows {
		e = err
	}

	return
}

func (f *FirstResponseTime) GetAverageAllTimeUser(ctx context.Context, guildId, userId uint64) (responseTime *time.Duration, e error) {
	query := `SELECT AVG(response_time) FROM first_response_time WHERE first_response_time.guild_id = $1 AND first_response_time.user_id = $2;`
	if err := f.QueryRow(ctx, query, guildId, userId).Scan(&responseTime); err != nil && err != pgx.ErrNoRows {
		e = err
	}

	return
}

// GetAverageTripleWindow returns the three fixed-window averages. It delegates
// to GetAverageWindows with days=0 and no panel filter, preserving the
// existing signature for the worker module.
func (f *FirstResponseTime) GetAverageTripleWindow(ctx context.Context, guildId uint64) (TripleWindow, error) {
	mw, err := f.GetAverageWindows(ctx, guildId, 0, nil)
	if err != nil {
		return TripleWindow{}, err
	}
	return mw.TripleWindow(), nil
}

// GetAverageWindows returns four first-response-time averages in one round trip:
// the exact selected range, all time, monthly, and weekly. The selected window
// equals all-time when days == 0.
func (f *FirstResponseTime) GetAverageWindows(ctx context.Context, guildId uint64, days int, filter *PanelFilter) (MetricWindows, error) {
	query := `
SELECT
    AVG(frt.response_time) FILTER (WHERE $2 = 0 OR t.open_time > NOW() - make_interval(days => $2)),
    AVG(frt.response_time),
    AVG(frt.response_time) FILTER (WHERE t.open_time > NOW() - INTERVAL '30 days'),
    AVG(frt.response_time) FILTER (WHERE t.open_time > NOW() - INTERVAL '7 days')
FROM first_response_time frt
INNER JOIN tickets t ON frt.guild_id = t.guild_id AND frt.ticket_id = t.id
WHERE frt.guild_id = $1` + PanelPredicate("t", 3, 4)

	arr, unassigned, err := filter.Args()
	if err != nil {
		return MetricWindows{}, fmt.Errorf("first response time windows: %w", err)
	}

	var mw MetricWindows
	if err := f.QueryRow(ctx, query, guildId, days, arr, unassigned).Scan(
		&mw.Selected, &mw.AllTime, &mw.Monthly, &mw.Weekly,
	); err != nil {
		return MetricWindows{}, err
	}

	return mw, nil
}

func (f *FirstResponseTime) GetAverageByHour(ctx context.Context, guildId uint64, days int, filter *PanelFilter) ([]ResponseTimeByHour, error) {
	query := `
SELECT
    EXTRACT(HOUR FROM t.open_time)::int AS hour_of_day,
    AVG(frt.response_time) AS avg_response_time
FROM first_response_time frt
INNER JOIN tickets t ON frt.guild_id = t.guild_id AND frt.ticket_id = t.id
WHERE frt.guild_id = $1
    AND ($2 = 0 OR t.open_time > NOW() - make_interval(days => $2))` +
		PanelPredicate("t", 3, 4) + `
GROUP BY hour_of_day
ORDER BY hour_of_day;`

	arr, unassigned, err := filter.Args()
	if err != nil {
		return nil, fmt.Errorf("response time by hour: %w", err)
	}

	rows, err := f.Query(ctx, query, guildId, days, arr, unassigned)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ResponseTimeByHour
	for rows.Next() {
		var r ResponseTimeByHour
		if err := rows.Scan(&r.HourOfDay, &r.AvgResponseTime); err != nil {
			return nil, err
		}
		results = append(results, r)
	}

	return results, nil
}

func (f *FirstResponseTime) Set(ctx context.Context, guildId, userId uint64, ticketId int, responseTime time.Duration) (err error) {
	query := `INSERT INTO first_response_time("guild_id", "ticket_id", "user_id", "response_time") VALUES($1, $2, $3, $4) ON CONFLICT("guild_id", "ticket_id") DO NOTHING;`
	_, err = f.Exec(ctx, query, guildId, ticketId, userId, responseTime)
	return
}
