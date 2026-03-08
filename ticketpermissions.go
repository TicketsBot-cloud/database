package database

import (
	"context"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type TicketPermissions struct {
	AddReactions        bool `json:"add_reactions"`
	SendTTSMessages     bool `json:"send_tts_messages"`
	EmbedLinks          bool `json:"embed_links"`
	AttachFiles         bool `json:"attach_files"`
	UseExternalEmojis   bool `json:"use_external_emojis"`
	UseExternalStickers bool `json:"use_external_stickers"`
	SendVoiceMessages   bool `json:"send_voice_messages"`
}

type TicketPermissionsTable struct {
	*pgxpool.Pool
}

func newTicketPermissionsTable(db *pgxpool.Pool) *TicketPermissionsTable {
	return &TicketPermissionsTable{
		db,
	}
}

func (c TicketPermissionsTable) Schema() string {
	return `
CREATE TABLE IF NOT EXISTS ticket_permissions(
	"guild_id" int8 NOT NULL,
	"add_reactions" bool NOT NULL DEFAULT 't',
	"send_tts_messages" bool NOT NULL DEFAULT 't',
	"embed_links" bool NOT NULL DEFAULT 't',
	"attach_files" bool NOT NULL DEFAULT 't',
	"use_external_emojis" bool NOT NULL DEFAULT 't',
	"use_external_stickers" bool NOT NULL DEFAULT 't',
	"send_voice_messages" bool NOT NULL DEFAULT 't',
	PRIMARY KEY("guild_id")
);
`
}

func (c *TicketPermissionsTable) Get(ctx context.Context, guildId uint64) (TicketPermissions, error) {
	query := `
SELECT "add_reactions", "send_tts_messages", "embed_links", "attach_files", "use_external_emojis", "use_external_stickers", "send_voice_messages"
FROM ticket_permissions
WHERE "guild_id" = $1;`

	var permissions TicketPermissions
	err := c.QueryRow(ctx, query, guildId).Scan(
		&permissions.AddReactions,
		&permissions.SendTTSMessages,
		&permissions.EmbedLinks,
		&permissions.AttachFiles,
		&permissions.UseExternalEmojis,
		&permissions.UseExternalStickers,
		&permissions.SendVoiceMessages,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return TicketPermissions{
				AddReactions:        true,
				SendTTSMessages:     true,
				EmbedLinks:          true,
				AttachFiles:         true,
				UseExternalEmojis:   true,
				UseExternalStickers: true,
				SendVoiceMessages:   true,
			}, nil
		} else {
			return TicketPermissions{}, err
		}
	}

	return permissions, nil
}

func (c *TicketPermissionsTable) Set(ctx context.Context, guildId uint64, permissions TicketPermissions) (err error) {
	query := `
INSERT INTO ticket_permissions("guild_id", "add_reactions", "send_tts_messages", "embed_links", "attach_files", "use_external_emojis", "use_external_stickers", "send_voice_messages")
VALUES($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT("guild_id") DO UPDATE SET "add_reactions" = $2, "send_tts_messages" = $3, "embed_links" = $4, "attach_files" = $5, "use_external_emojis" = $6, "use_external_stickers" = $7, "send_voice_messages" = $8;`

	_, err = c.Exec(ctx, query, guildId, permissions.AddReactions, permissions.SendTTSMessages, permissions.EmbedLinks, permissions.AttachFiles, permissions.UseExternalEmojis, permissions.UseExternalStickers, permissions.SendVoiceMessages)
	return
}

func (c *TicketPermissionsTable) Delete(ctx context.Context, guildId uint64) error {
	query := `DELETE FROM ticket_permissions WHERE "guild_id"=$1;`
	_, err := c.Exec(ctx, query, guildId)
	return err
}
