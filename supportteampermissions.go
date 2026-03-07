package database

import (
	"context"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type SupportTeamPermissions struct {
	AddReactions           bool `json:"add_reactions"`
	SendMessages           bool `json:"send_messages"`
	SendTTSMessages        bool `json:"send_tts_messages"`
	EmbedLinks             bool `json:"embed_links"`
	AttachFiles            bool `json:"attach_files"`
	MentionEveryone        bool `json:"mention_everyone"`
	UseExternalEmojis      bool `json:"use_external_emojis"`
	UseApplicationCommands bool `json:"use_application_commands"`
	UseExternalStickers    bool `json:"use_external_stickers"`
	SendVoiceMessages      bool `json:"send_voice_messages"`
}

type SupportTeamPermissionsTable struct {
	*pgxpool.Pool
}

func newSupportTeamPermissionsTable(db *pgxpool.Pool) *SupportTeamPermissionsTable {
	return &SupportTeamPermissionsTable{
		db,
	}
}

func (c SupportTeamPermissionsTable) Schema() string {
	return `
CREATE TABLE IF NOT EXISTS support_team_permissions(
	"team_id"                 int  NOT NULL,
	"add_reactions"            bool NOT NULL DEFAULT 't',
	"send_messages"            bool NOT NULL DEFAULT 't',
	"send_tts_messages"        bool NOT NULL DEFAULT 't',
	"embed_links"              bool NOT NULL DEFAULT 't',
	"attach_files"             bool NOT NULL DEFAULT 't',
	"mention_everyone"         bool NOT NULL DEFAULT 'f',
	"use_external_emojis"      bool NOT NULL DEFAULT 't',
	"use_application_commands" bool NOT NULL DEFAULT 't',
	"use_external_stickers"    bool NOT NULL DEFAULT 't',
	"send_voice_messages"      bool NOT NULL DEFAULT 't',
	FOREIGN KEY("team_id") REFERENCES support_team("id") ON DELETE CASCADE ON UPDATE CASCADE,
	PRIMARY KEY("team_id")
);
`
}

func defaultSupportTeamPermissions() SupportTeamPermissions {
	return SupportTeamPermissions{
		AddReactions:           true,
		SendMessages:           true,
		SendTTSMessages:        true,
		EmbedLinks:             true,
		AttachFiles:            true,
		MentionEveryone:        false,
		UseExternalEmojis:      true,
		UseApplicationCommands: true,
		UseExternalStickers:    true,
		SendVoiceMessages:      true,
	}
}

func (c *SupportTeamPermissionsTable) Get(ctx context.Context, teamId int) (SupportTeamPermissions, error) {
	query := `
SELECT "add_reactions", "send_messages", "send_tts_messages", "embed_links", "attach_files", "mention_everyone", "use_external_emojis", "use_application_commands", "use_external_stickers", "send_voice_messages"
FROM support_team_permissions
WHERE "team_id" = $1;`

	var perms SupportTeamPermissions
	err := c.QueryRow(ctx, query, teamId).Scan(
		&perms.AddReactions,
		&perms.SendMessages,
		&perms.SendTTSMessages,
		&perms.EmbedLinks,
		&perms.AttachFiles,
		&perms.MentionEveryone,
		&perms.UseExternalEmojis,
		&perms.UseApplicationCommands,
		&perms.UseExternalStickers,
		&perms.SendVoiceMessages,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return defaultSupportTeamPermissions(), nil
		}
		return SupportTeamPermissions{}, err
	}

	return perms, nil
}

func (c *SupportTeamPermissionsTable) Set(ctx context.Context, teamId int, perms SupportTeamPermissions) error {
	query := `
INSERT INTO support_team_permissions("team_id", "add_reactions", "send_messages", "send_tts_messages", "embed_links", "attach_files", "mention_everyone", "use_external_emojis", "use_application_commands", "use_external_stickers", "send_voice_messages")
VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT("team_id") DO UPDATE SET "add_reactions" = $2, "send_messages" = $3, "send_tts_messages" = $4, "embed_links" = $5, "attach_files" = $6, "mention_everyone" = $7, "use_external_emojis" = $8, "use_application_commands" = $9, "use_external_stickers" = $10, "send_voice_messages" = $11;`

	_, err := c.Exec(ctx, query, teamId, perms.AddReactions, perms.SendMessages, perms.SendTTSMessages, perms.EmbedLinks, perms.AttachFiles, perms.MentionEveryone, perms.UseExternalEmojis, perms.UseApplicationCommands, perms.UseExternalStickers, perms.SendVoiceMessages)
	return err
}

func (c *SupportTeamPermissionsTable) GetForTeams(ctx context.Context, teamIds []int) (map[int]SupportTeamPermissions, error) {
	result := make(map[int]SupportTeamPermissions)

	if len(teamIds) == 0 {
		return result, nil
	}

	query := `
SELECT "team_id", "add_reactions", "send_messages", "send_tts_messages", "embed_links", "attach_files", "mention_everyone", "use_external_emojis", "use_application_commands", "use_external_stickers", "send_voice_messages"
FROM support_team_permissions
WHERE "team_id" = ANY($1);`

	rows, err := c.Query(ctx, query, teamIds)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var teamId int
		var perms SupportTeamPermissions
		if err := rows.Scan(&teamId, &perms.AddReactions, &perms.SendMessages, &perms.SendTTSMessages, &perms.EmbedLinks, &perms.AttachFiles, &perms.MentionEveryone, &perms.UseExternalEmojis, &perms.UseApplicationCommands, &perms.UseExternalStickers, &perms.SendVoiceMessages); err != nil {
			return nil, err
		}
		result[teamId] = perms
	}

	return result, rows.Err()
}
