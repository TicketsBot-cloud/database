package database

import (
	"context"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type TicketPermissions struct {
	AttachFiles         bool `json:"attach_files"`
	EmbedLinks          bool `json:"embed_links"`
	AddReactions        bool `json:"add_reactions"`
	SendVoiceMessages   bool `json:"send_voice_messages"`
	SendTTSMessages     bool `json:"send_tts_messages"`
	UseExternalEmojis   bool `json:"use_external_emojis"`
	UseExternalStickers bool `json:"use_external_stickers"`
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
	"attach_files" bool NOT NULL DEFAULT 't',
	"embed_links" bool NOT NULL DEFAULT 't',
	"add_reactions" bool NOT NULL DEFAULT 't',
	"send_voice_messages" bool NOT NULL DEFAULT 't',
	"send_tts_messages" bool NOT NULL DEFAULT 't',
	"use_external_emojis" bool NOT NULL DEFAULT 't',
	"use_external_stickers" bool NOT NULL DEFAULT 't',
	PRIMARY KEY("guild_id")
);
`
}

func (c *TicketPermissionsTable) Get(ctx context.Context, guildId uint64) (TicketPermissions, error) {
	query := `
SELECT "attach_files", "embed_links", "add_reactions", "send_voice_messages", "send_tts_messages", "use_external_emojis", "use_external_stickers"
FROM ticket_permissions
WHERE "guild_id" = $1;`

	var permissions TicketPermissions
	err := c.QueryRow(ctx, query, guildId).Scan(
		&permissions.AttachFiles,
		&permissions.EmbedLinks,
		&permissions.AddReactions,
		&permissions.SendVoiceMessages,
		&permissions.SendTTSMessages,
		&permissions.UseExternalEmojis,
		&permissions.UseExternalStickers,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return TicketPermissions{
				AttachFiles:         true,
				EmbedLinks:          true,
				AddReactions:        true,
				SendVoiceMessages:   true,
				SendTTSMessages:     true,
				UseExternalEmojis:   true,
				UseExternalStickers: true,
			}, nil
		} else {
			return TicketPermissions{}, err
		}
	}

	return permissions, nil
}

func (c *TicketPermissionsTable) Set(ctx context.Context, guildId uint64, permissions TicketPermissions) (err error) {
	query := `
INSERT INTO ticket_permissions("guild_id", "attach_files", "embed_links", "add_reactions", "send_voice_messages", "send_tts_messages", "use_external_emojis", "use_external_stickers")
VALUES($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT("guild_id") DO UPDATE SET "attach_files" = $2, "embed_links" = $3, "add_reactions" = $4, "send_voice_messages" = $5, "send_tts_messages" = $6, "use_external_emojis" = $7, "use_external_stickers" = $8;`

	_, err = c.Exec(ctx, query, guildId, permissions.AttachFiles, permissions.EmbedLinks, permissions.AddReactions, permissions.SendVoiceMessages, permissions.SendTTSMessages, permissions.UseExternalEmojis, permissions.UseExternalStickers)
	return
}

func (c *TicketPermissionsTable) Delete(ctx context.Context, guildId uint64) error {
	query := `DELETE FROM ticket_permissions WHERE "guild_id"=$1;`
	_, err := c.Exec(ctx, query, guildId)
	return err
}
