package database

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// PurgeGuildData deletes all data associated with a guild from all tables.
func (d *Database) PurgeGuildData(ctx context.Context, guildId uint64, logger *zap.Logger) error {
	logger.Info("Starting guild data purge", zap.Uint64("guild_id", guildId))

	// Start a transaction for atomicity
	tx, err := d.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer tx.Rollback(ctx)

	// Tables with direct guild_id column
	// will be automatically deleted via ON DELETE CASCADE foreign key constraints
	directGuildIdTables := []string{
		// Ticket-related child tables (must be deleted before tickets)
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
		"ticket_last_message",
		"ticket_members",

		// Tickets table and its counter
		"tickets",
		"guild_ticket_counters",

		// Panels table
		"panels",
		"multi_panels",

		// Support team related
		"support_team",

		// Form-related
		"forms",

		// Embed-related
		"embeds",

		// Custom integration related
		"custom_integration_secret_values",
		"custom_integration_guilds",

		// Other guild-specific tables
		"active_language",
		"archive_channel",
		"auto_close",
		"blacklist",
		"channel_category",
		"claim_settings",
		"close_confirmation",
		"custom_colours",
		"feedback_enabled",
		"guild_metadata",
		"import_logs",
		"import_mapping",
		"legacy_premium_entitlement_guilds",
		"naming_scheme",
		"on_call",
		"permissions",
		"premium_guilds",
		"role_blacklist",
		"role_permissions",
		"settings",
		"staff_override",
		"tags",
		"ticket_limit",
		"ticket_permissions",
		"users_can_close",
		"user_guilds",
		"webhooks",
		"welcome_messages",
		"whitelabel_guilds",
	}

	// Delete from tables with direct guild_id column
	// Child tables are automatically deleted via CASCADE
	for _, table := range directGuildIdTables {
		query := fmt.Sprintf(`DELETE FROM %s WHERE guild_id = $1`, table)
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

		rowsAffected := result.RowsAffected()
		if rowsAffected > 0 {
			logger.Info(
				"Deleted rows from table",
				zap.String("table", table),
				zap.Uint64("guild_id", guildId),
				zap.Int64("rows_deleted", rowsAffected),
			)
		}
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.Info("Successfully completed guild data purge", zap.Uint64("guild_id", guildId))
	return nil
}
