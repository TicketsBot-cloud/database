package database

import (
	"context"

	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type CloseMetadata struct {
	Reason   *string `json:"reason"`
	ClosedBy *uint64 `json:"closed_by,string"` // Null if auto-closed
}

type CloseMetadataTable struct {
	*pgxpool.Pool
}

func newCloseReasonTable(db *pgxpool.Pool) *CloseMetadataTable {
	return &CloseMetadataTable{
		db,
	}
}

func (c CloseMetadataTable) Schema() string {
	return `
CREATE TABLE IF NOT EXISTS close_reason(
	"guild_id" int8 NOT NULL,
	"ticket_id" int4 NOT NULL,
	"close_reason" TEXT,
	"closed_by" int8,
	FOREIGN KEY("guild_id", "ticket_id") REFERENCES tickets("guild_id", "id"),
	PRIMARY KEY("guild_id", "ticket_id")
);
`
}

func (c *CloseMetadataTable) Get(ctx context.Context, guildId uint64, ticketId int) (CloseMetadata, bool, error) {
	query := `
SELECT "close_reason", "closed_by"
FROM close_reason
WHERE "guild_id" = $1 AND "ticket_id" = $2;
`

	var data CloseMetadata
	if err := c.QueryRow(ctx, query, guildId, ticketId).Scan(&data.Reason, &data.ClosedBy); err != nil {
		if err == pgx.ErrNoRows {
			return CloseMetadata{}, false, nil
		} else {
			return CloseMetadata{}, false, err
		}
	}

	return data, true, nil
}

func (c *CloseMetadataTable) GetMulti(ctx context.Context, guildId uint64, ticketIds []int) (map[int]CloseMetadata, error) {
	query := `
SELECT "ticket_id", "close_reason", "closed_by"
FROM close_reason
WHERE "guild_id" = $1 AND "ticket_id" = ANY($2);
`

	array := &pgtype.Int4Array{}
	if err := array.Set(ticketIds); err != nil {
		return nil, err
	}

	rows, err := c.Query(ctx, query, guildId, ticketIds)
	if err != nil {
		return nil, err
	}

	ticketMetadata := make(map[int]CloseMetadata)
	for rows.Next() {
		var ticketId int
		var data CloseMetadata
		if err := rows.Scan(&ticketId, &data.Reason, &data.ClosedBy); err != nil {
			return nil, err
		}

		ticketMetadata[ticketId] = data
	}

	return ticketMetadata, nil
}

func (c *CloseMetadataTable) Set(ctx context.Context, guildId uint64, ticketId int, data CloseMetadata) (err error) {
	query := `
INSERT INTO close_reason("guild_id", "ticket_id", "close_reason", "closed_by")
VALUES($1, $2, $3, $4)
ON CONFLICT("guild_id", "ticket_id") DO UPDATE SET "close_reason" = $3, "closed_by" = $4;
`

	_, err = c.Exec(ctx, query, guildId, ticketId, data.Reason, data.ClosedBy)
	return
}

func (c *CloseMetadataTable) SetBulk(ctx context.Context, guildId uint64, data map[int]CloseMetadata) (err error) {
	rows := make([][]interface{}, 0, len(data))

	for i := range data {
		rows = append(rows, []interface{}{
			guildId,
			i,
			data[i].Reason,
			data[i].ClosedBy,
		})
	}

	_, err = c.CopyFrom(ctx, pgx.Identifier{"close_reason"}, []string{"guild_id", "ticket_id", "close_reason", "closed_by"}, pgx.CopyFromRows(rows))
	return
}

func (c *CloseMetadataTable) Delete(ctx context.Context, guildId uint64, ticketId int) (err error) {
	query := `DELETE FROM close_reason WHERE "guild_id"=$1 AND "ticket_id"=$2;`
	_, err = c.Exec(ctx, query, guildId, ticketId)
	return
}

func (c *CloseMetadataTable) GetTopCloseReasons(ctx context.Context, guildId uint64, panelId *int, limit int) ([]string, error) {
	query := `
SELECT cr.close_reason
FROM close_reason cr
INNER JOIN tickets t ON cr.guild_id = t.guild_id AND cr.ticket_id = t.id
WHERE cr.guild_id = $1
  AND ($2::int IS NULL OR t.panel_id = $2)
  AND cr.close_reason IS NOT NULL
  AND cr.close_reason != ''
  AND cr.close_reason != 'Automatically closed due to inactivity'
GROUP BY cr.close_reason
ORDER BY COUNT(*) DESC
LIMIT $3;`

	rows, err := c.Query(ctx, query, guildId, panelId, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reasons []string
	for rows.Next() {
		var reason string
		if err := rows.Scan(&reason); err != nil {
			return nil, err
		}
		reasons = append(reasons, reason)
	}

	return reasons, nil
}

type CloseReasonCount struct {
	Reason string `json:"reason"`
	Count  int    `json:"count"`
}

func (c *CloseMetadataTable) GetTopCloseReasonsWithCount(ctx context.Context, guildId uint64, panelId *int, limit, days int) ([]CloseReasonCount, error) {
	query := `
SELECT cr.close_reason, COUNT(*)::int AS count
FROM close_reason cr
INNER JOIN tickets t ON cr.guild_id = t.guild_id AND cr.ticket_id = t.id
WHERE cr.guild_id = $1
  AND ($2::int IS NULL OR t.panel_id = $2)
  AND cr.close_reason IS NOT NULL
  AND cr.close_reason != ''
  AND cr.close_reason != 'Automatically closed due to inactivity'
  AND ($4 = 0 OR t.open_time > NOW() - make_interval(days => $4))
GROUP BY cr.close_reason
ORDER BY COUNT(*) DESC
LIMIT $3;`

	rows, err := c.Query(ctx, query, guildId, panelId, limit, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []CloseReasonCount
	for rows.Next() {
		var r CloseReasonCount
		if err := rows.Scan(&r.Reason, &r.Count); err != nil {
			return nil, err
		}
		results = append(results, r)
	}

	return results, nil
}

func (c *CloseMetadataTable) GetAutoCloseVsManualClose(ctx context.Context, guildId uint64, nDays int) (AutoCloseStats, error) {
	query := `
SELECT
    COUNT(*) FILTER (WHERE cr.closed_by IS NULL),
    COUNT(*) FILTER (WHERE cr.closed_by IS NOT NULL)
FROM close_reason cr
INNER JOIN tickets t ON cr.guild_id = t.guild_id AND cr.ticket_id = t.id
WHERE cr.guild_id = $1 AND t.close_time > CURRENT_DATE - ($2 - 1) * INTERVAL '1 day';`

	var stats AutoCloseStats
	if err := c.QueryRow(ctx, query, guildId, nDays).Scan(&stats.AutoClosed, &stats.ManualClosed); err != nil {
		return AutoCloseStats{}, err
	}

	return stats, nil
}

func (c *CloseMetadataTable) GetTopCloseReasonsContaining(ctx context.Context, guildId uint64, panelId *int, contains string, limit int) ([]string, error) {
	query := `
SELECT cr.close_reason
FROM close_reason cr
INNER JOIN tickets t ON cr.guild_id = t.guild_id AND cr.ticket_id = t.id
WHERE cr.guild_id = $1
  AND ($2::int IS NULL OR t.panel_id = $2)
  AND cr.close_reason IS NOT NULL
  AND cr.close_reason != ''
  AND cr.close_reason != 'Automatically closed due to inactivity'
  AND LOWER(cr.close_reason) LIKE '%' || LOWER($3) || '%'
GROUP BY cr.close_reason
ORDER BY CASE WHEN LOWER(cr.close_reason) LIKE LOWER($3) || '%' THEN 0 ELSE 1 END,
         COUNT(*) DESC
LIMIT $4;`

	rows, err := c.Query(ctx, query, guildId, panelId, contains, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reasons []string
	for rows.Next() {
		var reason string
		if err := rows.Scan(&reason); err != nil {
			return nil, err
		}
		reasons = append(reasons, reason)
	}

	return reasons, nil
}
