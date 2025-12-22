package database

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// PurgeGuildData deletes all data associated with a guild from all tables.
// This should be called when a bot is removed from a guild.
func (d *Database) PurgeGuildData(ctx context.Context, guildId uint64, logger *zap.Logger) error {
	logger.Info("Starting guild data purge", zap.Uint64("guild_id", guildId))

	// Start a transaction for atomicity
	tx, err := d.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer tx.Rollback(ctx)

	// Tables with direct guild_id column
	directGuildIdTables := []string{
		// Ticket-related child tables (must be deleted before tickets)
		"archive_messages",
		"auto_close_exclude",
		"category_update_queue",
		"close_reason",
		"close_request",
		"exit_survey_responses",
		"first_response_time",
		"participants",
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
		"guild_leave_time",
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
		"server_blacklist",
		"settings",
		"staff_override",
		"tag",
		"ticket_limit",
		"ticket_permissions",
		"used_keys",
		"user_can_close",
		"user_guilds",
		"webhooks",
		"welcome_messages",
		"whitelabel_guilds",
	}

	// Tables that need special handling (linked through foreign keys)
	// Format: table -> (column, parent_table, parent_column, parent_guild_column)
	linkedTables := map[string]struct {
		column            string
		parentTable       string
		parentColumn      string
		parentGuildColumn string
	}{
		"multi_panel_targets": {
			column:            "multi_panel_id",
			parentTable:       "multi_panels",
			parentColumn:      "id",
			parentGuildColumn: "guild_id",
		},
		"panel_access_control_rules": {
			column:            "panel_id",
			parentTable:       "panels",
			parentColumn:      "panel_id",
			parentGuildColumn: "guild_id",
		},
		"panel_here_mention": {
			column:            "panel_id",
			parentTable:       "panels",
			parentColumn:      "panel_id",
			parentGuildColumn: "guild_id",
		},
		"panel_role_mentions": {
			column:            "panel_id",
			parentTable:       "panels",
			parentColumn:      "panel_id",
			parentGuildColumn: "guild_id",
		},
		"panel_support_hours": {
			column:            "panel_id",
			parentTable:       "panels",
			parentColumn:      "panel_id",
			parentGuildColumn: "guild_id",
		},
		"panel_teams": {
			column:            "panel_id",
			parentTable:       "panels",
			parentColumn:      "panel_id",
			parentGuildColumn: "guild_id",
		},
		"panel_user_mention": {
			column:            "panel_id",
			parentTable:       "panels",
			parentColumn:      "panel_id",
			parentGuildColumn: "guild_id",
		},
		"support_team_members": {
			column:            "team_id",
			parentTable:       "support_team",
			parentColumn:      "id",
			parentGuildColumn: "guild_id",
		},
		"support_team_roles": {
			column:            "team_id",
			parentTable:       "support_team",
			parentColumn:      "id",
			parentGuildColumn: "guild_id",
		},
		"embed_fields": {
			column:            "embed_id",
			parentTable:       "embeds",
			parentColumn:      "id",
			parentGuildColumn: "guild_id",
		},
		"form_input_api_headers": {
			column:            "input_id",
			parentTable:       "form_input",
			parentColumn:      "id",
			parentGuildColumn: "guild_id",
		},
		"form_input_api_config": {
			column:            "input_id",
			parentTable:       "form_input",
			parentColumn:      "id",
			parentGuildColumn: "guild_id",
		},
		"form_input_options": {
			column:            "input_id",
			parentTable:       "form_input",
			parentColumn:      "id",
			parentGuildColumn: "guild_id",
		},
		"form_input": {
			column:            "form_id",
			parentTable:       "forms",
			parentColumn:      "form_id",
			parentGuildColumn: "guild_id",
		},
	}

	// Delete from linked tables first (using subqueries)
	for table, link := range linkedTables {
		query := fmt.Sprintf(
			`DELETE FROM %s WHERE %s IN (SELECT %s FROM %s WHERE %s = $1)`,
			table,
			link.column,
			link.parentColumn,
			link.parentTable,
			link.parentGuildColumn,
		)

		result, err := tx.Exec(ctx, query, guildId)
		if err != nil {
			logger.Error(
				"Failed to delete from linked table",
				zap.String("table", table),
				zap.Uint64("guild_id", guildId),
				zap.Error(err),
			)
			return fmt.Errorf("failed to delete from %s: %w", table, err)
		}

		rowsAffected := result.RowsAffected()
		if rowsAffected > 0 {
			logger.Info(
				"Deleted rows from linked table",
				zap.String("table", table),
				zap.Uint64("guild_id", guildId),
				zap.Int64("rows_deleted", rowsAffected),
			)
		}
	}

	// Delete from tables with direct guild_id column
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
