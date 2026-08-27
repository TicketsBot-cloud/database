package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type ServiceRatings struct {
	*pgxpool.Pool
}

func newServiceRatings(db *pgxpool.Pool) *ServiceRatings {
	return &ServiceRatings{
		db,
	}
}

func (ServiceRatings) Schema() string {
	return `
CREATE TABLE IF NOT EXISTS service_ratings(
	"guild_id" int8 NOT NULL,
	"ticket_id" int4 NOT NULL,
	"rating" int2 NOT NULL,
	FOREIGN KEY("guild_id", "ticket_id") REFERENCES tickets("guild_id", "id"),
	PRIMARY KEY("guild_id", "ticket_id")
);`
}

func (r *ServiceRatings) Get(ctx context.Context, guildId uint64, ticketId int) (rating uint8, ok bool, e error) {
	query := `SELECT "rating" from service_ratings WHERE "guild_id" = $1 AND "ticket_id" = $2;`

	err := r.QueryRow(ctx, query, guildId, ticketId).Scan(&rating)
	if err == nil {
		return rating, true, nil
	} else if err == pgx.ErrNoRows {
		return 0, false, nil
	} else {
		return 0, false, err
	}
}

// GetCount is the worker-compatible wrapper. Delegates to GetCountFiltered
// with days=0 and no panel filter.
func (r *ServiceRatings) GetCount(ctx context.Context, guildId uint64) (int, error) {
	return r.GetCountFiltered(ctx, guildId, 0, nil)
}

// GetCountFiltered returns the count of feedback ratings, optionally scoped by
// time window and panel filter.
func (r *ServiceRatings) GetCountFiltered(ctx context.Context, guildId uint64, days int, filter *PanelFilter) (int, error) {
	query := `
SELECT COUNT(*)
FROM service_ratings sr
INNER JOIN tickets t ON sr.guild_id = t.guild_id AND sr.ticket_id = t.id
WHERE sr.guild_id = $1
    AND ($2 = 0 OR t.open_time > NOW() - make_interval(days => $2))` +
		PanelPredicate("t", 3, 4)

	arr, unassigned, err := filter.Args()
	if err != nil {
		return 0, fmt.Errorf("feedback count: %w", err)
	}

	var count int
	if err := r.QueryRow(ctx, query, guildId, days, arr, unassigned).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (r *ServiceRatings) GetCountClaimedBy(ctx context.Context, guildId, userId uint64) (count int, err error) {
	query := `
SELECT COUNT(service_ratings.rating)
FROM service_ratings
INNER JOIN ticket_claims
ON service_ratings.guild_id = ticket_claims.guild_id AND service_ratings.ticket_id = ticket_claims.ticket_id
WHERE service_ratings.guild_id = $1 AND ticket_claims.user_id = $2;
`

	err = r.QueryRow(ctx, query, guildId, userId).Scan(&count)
	return
}

// GetAverage is the worker-compatible wrapper. Delegates to
// GetAverageFiltered with days=0 and no panel filter.
func (r *ServiceRatings) GetAverage(ctx context.Context, guildId uint64) (float32, error) {
	return r.GetAverageFiltered(ctx, guildId, 0, nil)
}

// GetAverageFiltered returns the average feedback rating, optionally scoped by
// time window and panel filter.
func (r *ServiceRatings) GetAverageFiltered(ctx context.Context, guildId uint64, days int, filter *PanelFilter) (float32, error) {
	query := `
SELECT AVG(sr.rating)
FROM service_ratings sr
INNER JOIN tickets t ON sr.guild_id = t.guild_id AND sr.ticket_id = t.id
WHERE sr.guild_id = $1
    AND ($2 = 0 OR t.open_time > NOW() - make_interval(days => $2))` +
		PanelPredicate("t", 3, 4)

	arr, unassigned, err := filter.Args()
	if err != nil {
		return 0, fmt.Errorf("average rating: %w", err)
	}

	var f *float32
	if err := r.QueryRow(ctx, query, guildId, days, arr, unassigned).Scan(&f); err != nil {
		return 0, err
	}
	if f != nil {
		return *f, nil
	}
	return 0, nil
}

func (r *ServiceRatings) GetAverageClaimedBy(ctx context.Context, guildId, userId uint64) (average float32, err error) {
	// Returns NULL if no tickets claimed
	var f *float32

	query := `
SELECT AVG(service_ratings.rating)
FROM service_ratings
INNER JOIN ticket_claims
ON service_ratings.guild_id = ticket_claims.guild_id AND service_ratings.ticket_id = ticket_claims.ticket_id
WHERE service_ratings.guild_id = $1 AND ticket_claims.user_id = $2;
`

	err = r.QueryRow(ctx, query, guildId, userId).Scan(&f)
	if f != nil {
		average = *f
	}

	return
}

func (r *ServiceRatings) GetMulti(ctx context.Context, guildId uint64, ticketIds []int) (map[int]uint8, error) {
	query := `SELECT "ticket_id", "rating" from service_ratings WHERE "guild_id" = $1 AND "ticket_id" = ANY($2);`

	idArray := &pgtype.Int4Array{}
	if err := idArray.Set(ticketIds); err != nil {
		return nil, err
	}

	ratings := make(map[int]uint8)

	rows, err := r.Query(ctx, query, guildId, idArray)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var ticketId int
		var rating uint8

		if err := rows.Scan(&ticketId, &rating); err != nil {
			return nil, err
		}

		ratings[ticketId] = rating
	}

	return ratings, nil
}

// [lower,upper]
func (r *ServiceRatings) GetRange(ctx context.Context, guildId uint64, lowerId, upperId int) (map[int]uint8, error) {
	query := `SELECT "ticket_id", "rating" from service_ratings WHERE "guild_id" = $1 AND "ticket_id" >= $2 AND "ticket_id" <= 3;`

	ratings := make(map[int]uint8)

	rows, err := r.Query(ctx, query, guildId, lowerId, upperId)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var ticketId int
		var rating uint8

		if err := rows.Scan(&ticketId, &rating); err != nil {
			return nil, err
		}

		ratings[ticketId] = rating
	}

	return ratings, nil
}

func (r *ServiceRatings) Set(ctx context.Context, guildId uint64, ticketId int, rating uint8) (err error) {
	query := `
INSERT INTO service_ratings("guild_id", "ticket_id", "rating")
VALUES($1, $2, $3)
ON CONFLICT("guild_id", "ticket_id") DO UPDATE SET "rating" = $3;`

	_, err = r.Exec(ctx, query, guildId, ticketId, rating)
	return
}

func (r *ServiceRatings) GetDistributionClaimedBy(ctx context.Context, guildId, userId uint64) ([5]int, error) {
	query := `
SELECT sr.rating, COUNT(*)
FROM service_ratings sr
INNER JOIN ticket_claims tc ON sr.guild_id = tc.guild_id AND sr.ticket_id = tc.ticket_id
WHERE sr.guild_id = $1 AND tc.user_id = $2
GROUP BY sr.rating ORDER BY sr.rating;`

	rows, err := r.Query(ctx, query, guildId, userId)
	if err != nil {
		return [5]int{}, err
	}
	defer rows.Close()

	var dist [5]int
	for rows.Next() {
		var rating, count int
		if err := rows.Scan(&rating, &count); err != nil {
			return [5]int{}, err
		}
		if rating >= 1 && rating <= 5 {
			dist[rating-1] = count
		}
	}

	return dist, nil
}

// GetDistribution is the worker-compatible wrapper. Delegates to
// GetDistributionFiltered with days=0 and no panel filter.
func (r *ServiceRatings) GetDistribution(ctx context.Context, guildId uint64) ([5]int, error) {
	return r.GetDistributionFiltered(ctx, guildId, 0, nil)
}

// GetDistributionFiltered returns the 1-5 star distribution, optionally
// scoped by time window and panel filter.
func (r *ServiceRatings) GetDistributionFiltered(ctx context.Context, guildId uint64, days int, filter *PanelFilter) ([5]int, error) {
	query := `
SELECT sr.rating, COUNT(*)
FROM service_ratings sr
INNER JOIN tickets t ON sr.guild_id = t.guild_id AND sr.ticket_id = t.id
WHERE sr.guild_id = $1
    AND ($2 = 0 OR t.open_time > NOW() - make_interval(days => $2))` +
		PanelPredicate("t", 3, 4) + `
GROUP BY sr.rating ORDER BY sr.rating;`

	arr, unassigned, err := filter.Args()
	if err != nil {
		return [5]int{}, fmt.Errorf("feedback distribution: %w", err)
	}

	rows, err := r.Query(ctx, query, guildId, days, arr, unassigned)
	if err != nil {
		return [5]int{}, err
	}
	defer rows.Close()

	var dist [5]int
	for rows.Next() {
		var rating, count int
		if err := rows.Scan(&rating, &count); err != nil {
			return [5]int{}, err
		}
		if rating >= 1 && rating <= 5 {
			dist[rating-1] = count
		}
	}

	return dist, nil
}

// GetResponseRate is the worker-compatible wrapper. Delegates to
// GetResponseRateFiltered with no panel filter.
func (r *ServiceRatings) GetResponseRate(ctx context.Context, guildId uint64, nDays int) (FeedbackResponseRate, error) {
	return r.GetResponseRateFiltered(ctx, guildId, nDays, nil)
}

func (r *ServiceRatings) GetResponseRateFiltered(ctx context.Context, guildId uint64, nDays int, filter *PanelFilter) (FeedbackResponseRate, error) {
	query := `
SELECT COUNT(t.id) AS closed, COUNT(sr.rating) AS rated
FROM tickets t
LEFT JOIN service_ratings sr ON t.guild_id = sr.guild_id AND t.id = sr.ticket_id
WHERE t.guild_id = $1 AND t.open = false
    AND t.close_time > CURRENT_DATE - ($2 - 1) * INTERVAL '1 day'` +
		PanelPredicate("t", 3, 4)

	arr, unassigned, err := filter.Args()
	if err != nil {
		return FeedbackResponseRate{}, fmt.Errorf("feedback response rate: %w", err)
	}

	var result FeedbackResponseRate
	if err := r.QueryRow(ctx, query, guildId, nDays, arr, unassigned).Scan(&result.ClosedTickets, &result.RatedTickets); err != nil {
		return FeedbackResponseRate{}, err
	}

	if result.ClosedTickets > 0 {
		result.Rate = float64(result.RatedTickets) / float64(result.ClosedTickets)
	}

	return result, nil
}
