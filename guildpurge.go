package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4"
	"go.uber.org/zap"
)

// Runs before guildPurgeTables so the panel children go while panels still exists.
var guildPurgeIndirect = []struct {
	table string
	query string
}{
	// No foreign key to panels, so deleting panels would orphan these.
	{"panel_ticket_permissions", `DELETE FROM panel_ticket_permissions WHERE panel_id IN (SELECT panel_id FROM panels WHERE guild_id = $1)`},
	{"panel_kb_categories", `DELETE FROM panel_kb_categories WHERE panel_id IN (SELECT panel_id FROM panels WHERE guild_id = $1)`},

	{"gallery_listings", `DELETE FROM gallery_listings WHERE source_guild_id = $1`},
	{"experiment_exposures", `DELETE FROM experiment_exposures WHERE identifier_type = 'guild_id' AND identifier = ($1::int8)::text`},
}

// Order is load-bearing: most children of tickets have no ON DELETE action, so
// listing one after its parent raises SQLSTATE 23503.
var guildPurgeTables = []string{
	"archive_dm_messages",
	"archive_messages",
	"auto_close_exclude",
	"category_update_queue",
	"close_reason",
	"close_request",
	"exit_survey_responses",
	"first_response_time",
	"participant",
	"service_ratings",
	"ticket_claims",
	"ticket_label_assignments",
	"ticket_last_message",
	"ticket_members",
	"ticket_message_counts",
	"webhooks",

	"tickets",
	"guild_ticket_counters",
	"ticket_labels",

	// Panels reference embeds and forms.
	"panels",
	"multi_panels",

	"support_team",
	"forms",
	"embeds",

	"custom_integration_secret_values",
	"custom_integration_guilds",

	"kb_article_feedback",
	"kb_articles",
	"kb_categories",
	"kb_settings",

	"active_language",
	"audit_logs",
	"blacklist",
	"channel_category",
	"claim_settings",
	"custom_colours",
	"dashboard_onboarding",
	"guild_metadata",
	"legacy_premium_entitlement_guilds",
	"on_call",
	"permissions",
	"premium_guilds",
	"role_blacklist",
	"role_permissions",
	"settings",
	"staff_override",
	"tags",
	"used_keys",
	"user_guilds",
	"whitelabel_allowed_guilds",
	"whitelabel_guilds",
}

// Dropped from the schema, but dropping a Go wrapper does not drop the table, so
// deployments may still hold rows. Remove an entry once production is confirmed clean.
var guildPurgeLegacy = []string{
	"archive_channel",
	"auto_close",
	"close_confirmation",
	"feedback_enabled",
	"import_logs",
	"import_mapping",
	"naming_scheme",
	"ticket_limit",
	"ticket_permissions",
	"users_can_close",
	"welcome_messages",
}

// Guild-scoped but deliberately retained; keeps the drift test honest.
var guildPurgeExempt = map[string]string{
	"guild_leave_time": "drives the purge itself; the caller deletes the row once the purge succeeds",
	"entitlements":     "billing record, retained beyond the guild",
	"server_blacklist": "global ban list; purging it would silently un-ban the guild",
}

// PurgeGuildData deletes all data associated with a guild from all tables.
func (d *Database) PurgeGuildData(ctx context.Context, guildId uint64, logger *zap.Logger) error {
	logger.Info("Starting guild data purge", zap.Uint64("guild_id", guildId))

	tx, err := d.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer tx.Rollback(ctx)

	for _, t := range guildPurgeIndirect {
		if err := purgeTable(ctx, tx, logger, guildId, t.table, t.query); err != nil {
			return err
		}
	}

	for _, table := range guildPurgeTables {
		query := fmt.Sprintf(`DELETE FROM %s WHERE guild_id = $1`, table)
		if err := purgeTable(ctx, tx, logger, guildId, table, query); err != nil {
			return err
		}
	}

	for _, table := range guildPurgeLegacy {
		query := fmt.Sprintf(`DELETE FROM %s WHERE guild_id = $1`, table)
		if err := purgeTable(ctx, tx, logger, guildId, table, query); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.Info("Successfully completed guild data purge", zap.Uint64("guild_id", guildId))
	return nil
}

func purgeTable(ctx context.Context, tx pgx.Tx, logger *zap.Logger, guildId uint64, table, query string) error {
	// A missing relation aborts the whole transaction, blocking every purge.
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT to_regclass($1) IS NOT NULL`, table).Scan(&exists); err != nil {
		return fmt.Errorf("failed to check existence of %s: %w", table, err)
	}

	if !exists {
		logger.Warn(
			"Skipping table that no longer exists",
			zap.String("table", table),
			zap.Uint64("guild_id", guildId),
		)
		return nil
	}

	result, err := tx.Exec(ctx, query, guildId)
	if err != nil {
		logger.Error(
			"Failed to delete from table",
			zap.String("table", table),
			zap.Uint64("guild_id", guildId),
			zap.Error(err),
		)
		return fmt.Errorf("failed to delete from %s: %w", table, err)
	}

	if rowsAffected := result.RowsAffected(); rowsAffected > 0 {
		logger.Info(
			"Deleted rows from table",
			zap.String("table", table),
			zap.Uint64("guild_id", guildId),
			zap.Int64("rows_deleted", rowsAffected),
		)
	}

	return nil
}
