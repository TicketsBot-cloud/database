package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type AuditActionType int16

const (
	AuditActionSettingsUpdate AuditActionType = 1

	AuditActionPanelCreate         AuditActionType = 10
	AuditActionPanelUpdate         AuditActionType = 11
	AuditActionPanelDelete         AuditActionType = 12
	AuditActionPanelResend         AuditActionType = 13
	AuditActionPanelResetCooldowns AuditActionType = 14

	AuditActionMultiPanelCreate AuditActionType = 20
	AuditActionMultiPanelUpdate AuditActionType = 21
	AuditActionMultiPanelDelete AuditActionType = 22
	AuditActionMultiPanelResend AuditActionType = 23

	AuditActionSupportHoursSet    AuditActionType = 30
	AuditActionSupportHoursDelete AuditActionType = 31

	AuditActionFormCreate AuditActionType = 40
	AuditActionFormUpdate AuditActionType = 41
	AuditActionFormDelete AuditActionType = 42

	AuditActionFormInputsUpdate AuditActionType = 45

	AuditActionTagCreate AuditActionType = 50
	AuditActionTagDelete AuditActionType = 51

	AuditActionTeamCreate AuditActionType = 60
	AuditActionTeamDelete AuditActionType = 61

	AuditActionTeamMemberAdd    AuditActionType = 65
	AuditActionTeamMemberRemove AuditActionType = 66

	AuditActionStaffOverrideCreate AuditActionType = 70
	AuditActionStaffOverrideDelete AuditActionType = 71

	AuditActionBlacklistAdd        AuditActionType = 80
	AuditActionBlacklistRemoveUser AuditActionType = 81
	AuditActionBlacklistRemoveRole AuditActionType = 82

	AuditActionTicketSendMessage AuditActionType = 90
	AuditActionTicketSendTag     AuditActionType = 91
	AuditActionTicketClose       AuditActionType = 92

	AuditActionGuildIntegrationActivate   AuditActionType = 100
	AuditActionGuildIntegrationUpdate     AuditActionType = 101
	AuditActionGuildIntegrationDeactivate AuditActionType = 102

	AuditActionImportTrigger AuditActionType = 110

	AuditActionPremiumSetActiveGuilds AuditActionType = 120

	AuditActionTicketLabelCreate   AuditActionType = 130
	AuditActionTicketLabelUpdate   AuditActionType = 131
	AuditActionTicketLabelDelete   AuditActionType = 132
	AuditActionTicketLabelAssign   AuditActionType = 135
	AuditActionTicketLabelUnassign AuditActionType = 136

	AuditActionUserIntegrationCreate    AuditActionType = 200
	AuditActionUserIntegrationUpdate    AuditActionType = 201
	AuditActionUserIntegrationDelete    AuditActionType = 202
	AuditActionUserIntegrationSetPublic AuditActionType = 203

	AuditActionWhitelabelCreate             AuditActionType = 210
	AuditActionWhitelabelDelete             AuditActionType = 211
	AuditActionWhitelabelCreateInteractions AuditActionType = 212
	AuditActionWhitelabelStatusSet          AuditActionType = 213
	AuditActionWhitelabelStatusDelete       AuditActionType = 214

	AuditActionBotStaffAdd    AuditActionType = 300
	AuditActionBotStaffRemove AuditActionType = 301
)

type AuditResourceType int16

const (
	AuditResourceSettings              AuditResourceType = 1
	AuditResourcePanel                 AuditResourceType = 2
	AuditResourceMultiPanel            AuditResourceType = 3
	AuditResourceSupportHours          AuditResourceType = 4
	AuditResourceForm                  AuditResourceType = 5
	AuditResourceFormInput             AuditResourceType = 6
	AuditResourceTag                   AuditResourceType = 7
	AuditResourceTeam                  AuditResourceType = 8
	AuditResourceTeamMember            AuditResourceType = 9
	AuditResourceStaffOverride         AuditResourceType = 10
	AuditResourceBlacklist             AuditResourceType = 11
	AuditResourceTicket                AuditResourceType = 12
	AuditResourceGuildIntegration      AuditResourceType = 13
	AuditResourceImport                AuditResourceType = 14
	AuditResourcePremium               AuditResourceType = 15
	AuditResourceUserIntegration       AuditResourceType = 16
	AuditResourceWhitelabel            AuditResourceType = 17
	AuditResourceBotStaff              AuditResourceType = 18
	AuditResourceTicketLabel           AuditResourceType = 19
	AuditResourceTicketLabelAssignment AuditResourceType = 20
)

type AuditLogEntry struct {
	Id           int64
	GuildId      *uint64
	UserId       uint64
	ActionType   AuditActionType
	ResourceType AuditResourceType
	ResourceId   *string
	OldData      *string
	NewData      *string
	Metadata     *string
	CreatedAt    time.Time
}

type AuditLogQueryOptions struct {
	GuildId      *uint64
	UserId       *uint64
	ActionType   *int16
	ResourceType *int16
	Before       *time.Time
	After        *time.Time
	Limit        int
	Offset       int
}

type AuditLogTable struct {
	*pgxpool.Pool
}

func newAuditLogTable(pool *pgxpool.Pool) *AuditLogTable {
	return &AuditLogTable{
		Pool: pool,
	}
}

func (t AuditLogTable) Schema() string {
	return `
CREATE TABLE IF NOT EXISTS audit_logs (
	"id"            BIGSERIAL       PRIMARY KEY,
	"guild_id"      INT8            DEFAULT NULL,
	"user_id"       INT8            NOT NULL,
	"action_type"   INT2            NOT NULL,
	"resource_type" INT2            NOT NULL,
	"resource_id"   TEXT            DEFAULT NULL,
	"old_data"      JSONB           DEFAULT NULL,
	"new_data"      JSONB           DEFAULT NULL,
	"metadata"      JSONB           DEFAULT NULL,
	"created_at"    TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS audit_logs_guild_id_created_at_idx ON audit_logs("guild_id", "created_at" DESC);
CREATE INDEX IF NOT EXISTS audit_logs_user_id_idx ON audit_logs("user_id");
CREATE INDEX IF NOT EXISTS audit_logs_action_type_idx ON audit_logs("action_type");
CREATE INDEX IF NOT EXISTS audit_logs_resource_type_idx ON audit_logs("resource_type");
CREATE INDEX IF NOT EXISTS audit_logs_created_at_idx ON audit_logs("created_at" DESC);
`
}

func (t *AuditLogTable) Insert(ctx context.Context, entry AuditLogEntry) error {
	query := `
INSERT INTO audit_logs ("guild_id", "user_id", "action_type", "resource_type", "resource_id", "old_data", "new_data", "metadata")
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);`

	_, err := t.Exec(ctx, query,
		entry.GuildId,
		entry.UserId,
		entry.ActionType,
		entry.ResourceType,
		entry.ResourceId,
		entry.OldData,
		entry.NewData,
		entry.Metadata,
	)
	return err
}

func (t *AuditLogTable) Query(ctx context.Context, opts AuditLogQueryOptions) ([]AuditLogEntry, error) {
	query, args := buildAuditLogQuery("SELECT \"id\", \"guild_id\", \"user_id\", \"action_type\", \"resource_type\", \"resource_id\", \"old_data\", \"new_data\", \"metadata\", \"created_at\" FROM audit_logs", opts)
	query += " ORDER BY \"created_at\" DESC"

	if opts.Limit > 0 {
		args = append(args, opts.Limit)
		query += fmt.Sprintf(" LIMIT $%d", len(args))
	}

	if opts.Offset > 0 {
		args = append(args, opts.Offset)
		query += fmt.Sprintf(" OFFSET $%d", len(args))
	}

	rows, err := t.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []AuditLogEntry
	for rows.Next() {
		var entry AuditLogEntry
		if err := rows.Scan(
			&entry.Id,
			&entry.GuildId,
			&entry.UserId,
			&entry.ActionType,
			&entry.ResourceType,
			&entry.ResourceId,
			&entry.OldData,
			&entry.NewData,
			&entry.Metadata,
			&entry.CreatedAt,
		); err != nil {
			return nil, err
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

func (t *AuditLogTable) Count(ctx context.Context, opts AuditLogQueryOptions) (int, error) {
	query, args := buildAuditLogQuery("SELECT COUNT(*) FROM audit_logs", opts)

	var count int
	err := t.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil && err != pgx.ErrNoRows {
		return 0, err
	}

	return count, nil
}

func buildAuditLogQuery(base string, opts AuditLogQueryOptions) (string, []interface{}) {
	var conditions []string
	var args []interface{}

	if opts.GuildId != nil {
		args = append(args, *opts.GuildId)
		conditions = append(conditions, fmt.Sprintf("\"guild_id\" = $%d", len(args)))
	}

	if opts.UserId != nil {
		args = append(args, *opts.UserId)
		conditions = append(conditions, fmt.Sprintf("\"user_id\" = $%d", len(args)))
	}

	if opts.ActionType != nil {
		args = append(args, *opts.ActionType)
		conditions = append(conditions, fmt.Sprintf("\"action_type\" = $%d", len(args)))
	}

	if opts.ResourceType != nil {
		args = append(args, *opts.ResourceType)
		conditions = append(conditions, fmt.Sprintf("\"resource_type\" = $%d", len(args)))
	}

	if opts.Before != nil {
		args = append(args, *opts.Before)
		conditions = append(conditions, fmt.Sprintf("\"created_at\" < $%d", len(args)))
	}

	if opts.After != nil {
		args = append(args, *opts.After)
		conditions = append(conditions, fmt.Sprintf("\"created_at\" > $%d", len(args)))
	}

	if len(conditions) > 0 {
		base += " WHERE " + strings.Join(conditions, " AND ")
	}

	return base, args
}
