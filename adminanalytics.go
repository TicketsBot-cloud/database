package database

import (
	"context"
	stdjson "encoding/json"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type AdminAnalyticsTable struct {
	*pgxpool.Pool
}

func newAdminAnalyticsTable(db *pgxpool.Pool) *AdminAnalyticsTable {
	return &AdminAnalyticsTable{db}
}

func (a AdminAnalyticsTable) Schema() string {
	return `
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_admin_usage AS
SELECT
    COUNT(*) FILTER (WHERE open_time > NOW() - INTERVAL '1 day') AS tickets_created_today,
    COUNT(DISTINCT guild_id) FILTER (WHERE open_time > NOW() - INTERVAL '1 day') AS active_guilds_daily,
    COUNT(DISTINCT guild_id) FILTER (WHERE open_time > NOW() - INTERVAL '7 days') AS active_guilds_weekly,
    COUNT(DISTINCT guild_id) FILTER (WHERE open_time > NOW() - INTERVAL '30 days') AS active_guilds_monthly,
    COUNT(DISTINCT guild_id) AS total_guilds
FROM tickets;

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_admin_tickets_per_day AS
SELECT date_trunc('day', open_time AT TIME ZONE 'UTC')::date AS date, COUNT(*)::int AS count
FROM tickets
WHERE open_time > NOW() - INTERVAL '90 days'
GROUP BY 1
ORDER BY 1;

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_admin_feature_adoption AS
SELECT 'Panels' AS feature, COUNT(DISTINCT guild_id)::int AS guild_count FROM panels
UNION ALL SELECT 'Multipanels', COUNT(DISTINCT guild_id)::int FROM multi_panels
UNION ALL SELECT 'Forms', COUNT(DISTINCT guild_id)::int FROM forms
UNION ALL SELECT 'Teams', COUNT(DISTINCT guild_id)::int FROM support_team
UNION ALL SELECT 'Labels', COUNT(DISTINCT guild_id)::int FROM ticket_labels
UNION ALL SELECT 'Feedback', COUNT(DISTINCT guild_id)::int FROM service_ratings
UNION ALL SELECT 'Integrations', COUNT(DISTINCT guild_id)::int FROM custom_integration_guilds
UNION ALL SELECT 'Close Reasons', COUNT(DISTINCT guild_id)::int FROM close_reason WHERE close_reason IS NOT NULL;

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_admin_retention AS
WITH active AS (
    SELECT DISTINCT guild_id FROM tickets WHERE open_time > NOW() - INTERVAL '30 days'
), previously_active AS (
    SELECT DISTINCT guild_id FROM tickets
    WHERE open_time > NOW() - INTERVAL '60 days' AND open_time <= NOW() - INTERVAL '30 days'
), churned AS (
    SELECT pa.guild_id, (SELECT MAX(open_time) FROM tickets t WHERE t.guild_id = pa.guild_id) AS last_ticket_time
    FROM previously_active pa
    WHERE NOT EXISTS (SELECT 1 FROM active a WHERE a.guild_id = pa.guild_id)
)
SELECT
    (SELECT COUNT(*) FROM active) AS active_guilds_30d,
    (SELECT COUNT(*) FROM churned) AS churned_guilds_30d,
    COALESCE(json_agg(json_build_object('guild_id', c.guild_id, 'last_ticket_time', c.last_ticket_time) ORDER BY c.last_ticket_time DESC) FILTER (WHERE c.guild_id IS NOT NULL), '[]'::json) AS recently_churned
FROM (SELECT * FROM churned ORDER BY last_ticket_time DESC LIMIT 50) c;

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_admin_config_patterns AS
SELECT setting, value, count FROM (
    SELECT 'Feedback Enabled' AS setting, 'Yes' AS value, COUNT(DISTINCT guild_id)::int AS count FROM service_ratings
    UNION ALL
    SELECT 'Panels per Guild', panel_bucket, COUNT(*)::int FROM (
        SELECT CASE
            WHEN cnt = 1 THEN '1'
            WHEN cnt <= 3 THEN '2-3'
            WHEN cnt <= 5 THEN '4-5'
            WHEN cnt <= 10 THEN '6-10'
            ELSE '11+'
        END AS panel_bucket
        FROM (SELECT guild_id, COUNT(*)::int AS cnt FROM panels GROUP BY guild_id) sub
    ) bucketed GROUP BY panel_bucket
) combined;

CREATE UNIQUE INDEX IF NOT EXISTS mv_admin_tickets_per_day_date ON mv_admin_tickets_per_day(date);
CREATE UNIQUE INDEX IF NOT EXISTS mv_admin_feature_adoption_feature ON mv_admin_feature_adoption(feature);
CREATE UNIQUE INDEX IF NOT EXISTS mv_admin_config_patterns_idx ON mv_admin_config_patterns(setting, value);
`
}

func (a *AdminAnalyticsTable) RefreshViews(ctx context.Context) error {
	views := []string{
		"REFRESH MATERIALIZED VIEW CONCURRENTLY mv_admin_tickets_per_day",
		"REFRESH MATERIALIZED VIEW CONCURRENTLY mv_admin_feature_adoption",
		"REFRESH MATERIALIZED VIEW CONCURRENTLY mv_admin_config_patterns",
		"REFRESH MATERIALIZED VIEW mv_admin_usage",
		"REFRESH MATERIALIZED VIEW mv_admin_retention",
	}

	for _, q := range views {
		if _, err := a.Exec(ctx, q); err != nil {
			return err
		}
	}

	return nil
}

type GlobalUsageMetrics struct {
	TicketsCreatedToday int `json:"tickets_created_today"`
	ActiveGuildsDaily   int `json:"active_guilds_daily"`
	ActiveGuildsWeekly  int `json:"active_guilds_weekly"`
	ActiveGuildsMonthly int `json:"active_guilds_monthly"`
	TotalGuilds         int `json:"total_guilds"`
}

func (a *AdminAnalyticsTable) GetGlobalUsageMetrics(ctx context.Context) (metrics GlobalUsageMetrics, e error) {
	query := `SELECT tickets_created_today, active_guilds_daily, active_guilds_weekly, active_guilds_monthly, total_guilds FROM mv_admin_usage;`
	err := a.QueryRow(ctx, query).Scan(
		&metrics.TicketsCreatedToday,
		&metrics.ActiveGuildsDaily,
		&metrics.ActiveGuildsWeekly,
		&metrics.ActiveGuildsMonthly,
		&metrics.TotalGuilds,
	)
	if err != nil && err != pgx.ErrNoRows {
		e = err
	}
	return
}

type GlobalTicketsPerDay struct {
	Date  time.Time `json:"date"`
	Count int       `json:"count"`
}

func (a *AdminAnalyticsTable) GetGlobalTicketsPerDay(ctx context.Context) ([]GlobalTicketsPerDay, error) {
	query := `SELECT date, count FROM mv_admin_tickets_per_day ORDER BY date;`
	rows, err := a.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []GlobalTicketsPerDay
	for rows.Next() {
		var r GlobalTicketsPerDay
		if err := rows.Scan(&r.Date, &r.Count); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, nil
}

type FeatureAdoption struct {
	Feature    string `json:"feature"`
	GuildCount int    `json:"guild_count"`
}

func (a *AdminAnalyticsTable) GetFeatureAdoption(ctx context.Context) ([]FeatureAdoption, error) {
	query := `SELECT feature, guild_count FROM mv_admin_feature_adoption ORDER BY guild_count DESC;`
	rows, err := a.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []FeatureAdoption
	for rows.Next() {
		var r FeatureAdoption
		if err := rows.Scan(&r.Feature, &r.GuildCount); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, nil
}

type ChurnedGuild struct {
	GuildId        uint64    `json:"guild_id"`
	LastTicketTime time.Time `json:"last_ticket_time"`
}

type RetentionMetrics struct {
	ActiveGuilds30d  int            `json:"active_guilds_30d"`
	ChurnedGuilds30d int            `json:"churned_guilds_30d"`
	RecentlyChurned  []ChurnedGuild `json:"recently_churned"`
}

func (a *AdminAnalyticsTable) GetRetentionMetrics(ctx context.Context) (metrics RetentionMetrics, e error) {
	var churnedJson []byte
	query := `SELECT active_guilds_30d, churned_guilds_30d, recently_churned::text FROM mv_admin_retention;`
	err := a.QueryRow(ctx, query).Scan(&metrics.ActiveGuilds30d, &metrics.ChurnedGuilds30d, &churnedJson)
	if err != nil {
		if err == pgx.ErrNoRows {
			metrics.RecentlyChurned = make([]ChurnedGuild, 0)
			return metrics, nil
		}
		return metrics, err
	}

	if len(churnedJson) > 0 {
		if err := stdjson.Unmarshal(churnedJson, &metrics.RecentlyChurned); err != nil {
			return metrics, err
		}
	}

	if metrics.RecentlyChurned == nil {
		metrics.RecentlyChurned = make([]ChurnedGuild, 0)
	}
	return
}

type ConfigPatternEntry struct {
	Setting string `json:"setting"`
	Value   string `json:"value"`
	Count   int    `json:"count"`
}

func (a *AdminAnalyticsTable) GetConfigPatterns(ctx context.Context) ([]ConfigPatternEntry, error) {
	query := `SELECT setting, value, count FROM mv_admin_config_patterns ORDER BY setting, count DESC;`
	rows, err := a.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ConfigPatternEntry
	for rows.Next() {
		var r ConfigPatternEntry
		if err := rows.Scan(&r.Setting, &r.Value, &r.Count); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, nil
}
