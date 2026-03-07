package database

import (
	"context"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type SupportTeamPermissions struct {
	SendMessages           bool `json:"send_messages"`
	EmbedLinks             bool `json:"embed_links"`
	AttachFiles            bool `json:"attach_files"`
	AddReactions           bool `json:"add_reactions"`
	SendVoiceMessages      bool `json:"send_voice_messages"`
	SendTTSMessages        bool `json:"send_tts_messages"`
	UseApplicationCommands bool `json:"use_application_commands"`
	MentionEveryone        bool `json:"mention_everyone"`
	UseExternalEmojis      bool `json:"use_external_emojis"`
	UseExternalStickers    bool `json:"use_external_stickers"`
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
	"send_messages"           bool NOT NULL DEFAULT 't',
	"embed_links"             bool NOT NULL DEFAULT 't',
	"attach_files"            bool NOT NULL DEFAULT 't',
	"add_reactions"           bool NOT NULL DEFAULT 't',
	"send_voice_messages"     bool NOT NULL DEFAULT 't',
	"send_tts_messages"       bool NOT NULL DEFAULT 't',
	"use_application_commands" bool NOT NULL DEFAULT 't',
	"mention_everyone"         bool NOT NULL DEFAULT 'f',
	"use_external_emojis"      bool NOT NULL DEFAULT 't',
	"use_external_stickers"    bool NOT NULL DEFAULT 't',
	FOREIGN KEY("team_id") REFERENCES support_team("id") ON DELETE CASCADE ON UPDATE CASCADE,
	PRIMARY KEY("team_id")
);
`
}

func defaultSupportTeamPermissions() SupportTeamPermissions {
	return SupportTeamPermissions{
		SendMessages:           true,
		EmbedLinks:             true,
		AttachFiles:            true,
		AddReactions:           true,
		SendVoiceMessages:      true,
		SendTTSMessages:        true,
		UseApplicationCommands: true,
		MentionEveryone:        false,
		UseExternalEmojis:      true,
		UseExternalStickers:    true,
	}
}

func (c *SupportTeamPermissionsTable) Get(ctx context.Context, teamId int) (SupportTeamPermissions, error) {
	query := `
SELECT "send_messages", "embed_links", "attach_files", "add_reactions", "send_voice_messages", "send_tts_messages", "use_application_commands", "mention_everyone", "use_external_emojis", "use_external_stickers"
FROM support_team_permissions
WHERE "team_id" = $1;`

	var perms SupportTeamPermissions
	err := c.QueryRow(ctx, query, teamId).Scan(
		&perms.SendMessages,
		&perms.EmbedLinks,
		&perms.AttachFiles,
		&perms.AddReactions,
		&perms.SendVoiceMessages,
		&perms.SendTTSMessages,
		&perms.UseApplicationCommands,
		&perms.MentionEveryone,
		&perms.UseExternalEmojis,
		&perms.UseExternalStickers,
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
INSERT INTO support_team_permissions("team_id", "send_messages", "embed_links", "attach_files", "add_reactions", "send_voice_messages", "send_tts_messages", "use_application_commands", "mention_everyone", "use_external_emojis", "use_external_stickers")
VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT("team_id") DO UPDATE SET "send_messages" = $2, "embed_links" = $3, "attach_files" = $4, "add_reactions" = $5, "send_voice_messages" = $6, "send_tts_messages" = $7, "use_application_commands" = $8, "mention_everyone" = $9, "use_external_emojis" = $10, "use_external_stickers" = $11;`

	_, err := c.Exec(ctx, query, teamId, perms.SendMessages, perms.EmbedLinks, perms.AttachFiles, perms.AddReactions, perms.SendVoiceMessages, perms.SendTTSMessages, perms.UseApplicationCommands, perms.MentionEveryone, perms.UseExternalEmojis, perms.UseExternalStickers)
	return err
}

func (c *SupportTeamPermissionsTable) GetForTeams(ctx context.Context, teamIds []int) (map[int]SupportTeamPermissions, error) {
	result := make(map[int]SupportTeamPermissions)

	if len(teamIds) == 0 {
		return result, nil
	}

	query := `
SELECT "team_id", "send_messages", "embed_links", "attach_files", "add_reactions", "send_voice_messages", "send_tts_messages", "use_application_commands", "mention_everyone", "use_external_emojis", "use_external_stickers"
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
		if err := rows.Scan(&teamId, &perms.SendMessages, &perms.EmbedLinks, &perms.AttachFiles, &perms.AddReactions, &perms.SendVoiceMessages, &perms.SendTTSMessages, &perms.UseApplicationCommands, &perms.MentionEveryone, &perms.UseExternalEmojis, &perms.UseExternalStickers); err != nil {
			return nil, err
		}
		result[teamId] = perms
	}

	return result, rows.Err()
}
