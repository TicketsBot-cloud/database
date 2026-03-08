package database

import (
	"context"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type PanelTicketPermissionsTable struct {
	*pgxpool.Pool
}

func newPanelTicketPermissionsTable(db *pgxpool.Pool) *PanelTicketPermissionsTable {
	return &PanelTicketPermissionsTable{
		db,
	}
}

func (c PanelTicketPermissionsTable) Schema() string {
	return `
CREATE TABLE IF NOT EXISTS panel_ticket_permissions(
	"panel_id" int8 NOT NULL,
	"add_reactions" bool NOT NULL DEFAULT false,
	"send_tts_messages" bool NOT NULL DEFAULT false,
	"embed_links" bool NOT NULL DEFAULT false,
	"attach_files" bool NOT NULL DEFAULT false,
	"use_external_emojis" bool NOT NULL DEFAULT false,
	"use_external_stickers" bool NOT NULL DEFAULT false,
	"send_voice_messages" bool NOT NULL DEFAULT false,
	PRIMARY KEY("panel_id")
);
`
}

func (c *PanelTicketPermissionsTable) Get(ctx context.Context, panelId int) (TicketPermissions, error) {
	query := `
SELECT "add_reactions", "send_tts_messages", "embed_links", "attach_files", "use_external_emojis", "use_external_stickers", "send_voice_messages"
FROM panel_ticket_permissions
WHERE "panel_id" = $1;`

	var permissions TicketPermissions
	err := c.QueryRow(ctx, query, panelId).Scan(
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
			return TicketPermissions{}, nil
		} else {
			return TicketPermissions{}, err
		}
	}

	return permissions, nil
}

func (c *PanelTicketPermissionsTable) Set(ctx context.Context, panelId int, permissions TicketPermissions) (err error) {
	query := `
INSERT INTO panel_ticket_permissions("panel_id", "add_reactions", "send_tts_messages", "embed_links", "attach_files", "use_external_emojis", "use_external_stickers", "send_voice_messages")
VALUES($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT("panel_id") DO UPDATE SET "add_reactions" = $2, "send_tts_messages" = $3, "embed_links" = $4, "attach_files" = $5, "use_external_emojis" = $6, "use_external_stickers" = $7, "send_voice_messages" = $8;`

	_, err = c.Exec(ctx, query, panelId, permissions.AddReactions, permissions.SendTTSMessages, permissions.EmbedLinks, permissions.AttachFiles, permissions.UseExternalEmojis, permissions.UseExternalStickers, permissions.SendVoiceMessages)
	return
}

func (c *PanelTicketPermissionsTable) Delete(ctx context.Context, panelId int) error {
	query := `DELETE FROM panel_ticket_permissions WHERE "panel_id"=$1;`
	_, err := c.Exec(ctx, query, panelId)
	return err
}
