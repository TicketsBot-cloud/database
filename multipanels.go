package database

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type MultiPanel struct {
	Id                    int                    `json:"id"`
	MessageId             uint64                 `json:"message_id,string"`
	ChannelId             uint64                 `json:"channel_id,string"`
	GuildId               uint64                 `json:"guild_id,string"`
	SelectMenu            bool                   `json:"select_menu"`
	SelectMenuPlaceholder *string                `json:"select_menu_placeholder"`
	Embed                 *CustomEmbedWithFields `json:"embed"`
	UsesComponentsV2      bool                   `json:"uses_components_v2"`
	Components            *string                `json:"components"`
	ForceDisabled         bool                   `json:"force_disabled"`
	Name                  *string                `json:"name"`
}

type MultiPanelTable struct {
	*pgxpool.Pool
}

func newMultiMultiPanelTable(db *pgxpool.Pool) *MultiPanelTable {
	return &MultiPanelTable{
		db,
	}
}

func (MultiPanelTable) Schema() string {
	return `
CREATE TABLE IF NOT EXISTS multi_panels(
	"id" SERIAL NOT NULL,
	"message_id" int8 NOT NULL,
	"channel_id" int8 NOT NULL,
	"guild_id" int8 NOT NULL,
	"select_menu" bool DEFAULT 'f',
	"select_menu_placeholder" VARCHAR(150) DEFAULT NULL,
	"embed" JSONB DEFAULT NULL,
	PRIMARY KEY("id")
);
CREATE INDEX IF NOT EXISTS multi_panels_guild_id ON multi_panels("guild_id");
CREATE INDEX IF NOT EXISTS multi_panels_message_id ON multi_panels("message_id");
ALTER TABLE multi_panels ADD COLUMN IF NOT EXISTS "uses_components_v2" bool NOT NULL DEFAULT false;
ALTER TABLE multi_panels ADD COLUMN IF NOT EXISTS "components" JSONB DEFAULT NULL;
ALTER TABLE multi_panels ADD COLUMN IF NOT EXISTS "force_disabled" bool NOT NULL DEFAULT false;
ALTER TABLE multi_panels ADD COLUMN IF NOT EXISTS "name" text DEFAULT NULL;`
}

func (p *MultiPanelTable) Get(ctx context.Context, id int) (MultiPanel, bool, error) {
	query := `
SELECT
	"id", "message_id", "channel_id", "guild_id", "select_menu", "select_menu_placeholder", "embed", "uses_components_v2", "components", "force_disabled", "name"
FROM
	multi_panels
WHERE
	"id" = $1
;`

	var panel MultiPanel
	var embedRaw *string
	err := p.QueryRow(ctx, query, id).Scan(
		&panel.Id, &panel.MessageId, &panel.ChannelId, &panel.GuildId, &panel.SelectMenu, &panel.SelectMenuPlaceholder, &embedRaw, &panel.UsesComponentsV2, &panel.Components, &panel.ForceDisabled, &panel.Name,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MultiPanel{}, false, nil
		} else {
			return MultiPanel{}, false, err
		}
	}

	if embedRaw != nil {
		if err := json.Unmarshal([]byte(*embedRaw), &panel.Embed); err != nil {
			return MultiPanel{}, false, err
		}
	}

	return panel, true, nil
}

func (p *MultiPanelTable) GetByMessageId(ctx context.Context, messageId uint64) (MultiPanel, bool, error) {
	query := `
SELECT
	"id", "message_id", "channel_id", "guild_id", "select_menu", "select_menu_placeholder", "embed", "uses_components_v2", "components", "force_disabled", "name"
FROM
	multi_panels
WHERE
	"message_id" = $1
;`

	var panel MultiPanel
	var embedRaw *string
	err := p.QueryRow(ctx, query, messageId).Scan(
		&panel.Id, &panel.MessageId, &panel.ChannelId, &panel.GuildId, &panel.SelectMenu, &panel.SelectMenuPlaceholder, &embedRaw, &panel.UsesComponentsV2, &panel.Components, &panel.ForceDisabled, &panel.Name,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MultiPanel{}, false, nil
		} else {
			return MultiPanel{}, false, err
		}
	}

	if embedRaw != nil {
		if err := json.Unmarshal([]byte(*embedRaw), &panel.Embed); err != nil {
			return MultiPanel{}, false, err
		}
	}

	return panel, true, nil
}

func (p *MultiPanelTable) GetByGuild(ctx context.Context, guildId uint64) ([]MultiPanel, error) {
	query := `
SELECT "id", "message_id", "channel_id", "guild_id", "select_menu", "select_menu_placeholder", "embed", "uses_components_v2", "components", "force_disabled", "name"
FROM multi_panels
WHERE "guild_id" = $1;
`

	rows, err := p.Query(ctx, query, guildId)
	defer rows.Close()
	if err != nil {
		return nil, err
	}

	var panels []MultiPanel
	for rows.Next() {
		var panel MultiPanel
		var embedRaw *string
		err := rows.Scan(
			&panel.Id, &panel.MessageId, &panel.ChannelId, &panel.GuildId, &panel.SelectMenu, &panel.SelectMenuPlaceholder, &embedRaw, &panel.UsesComponentsV2, &panel.Components, &panel.ForceDisabled, &panel.Name,
		)

		if err != nil {
			return nil, err
		}

		if embedRaw != nil {
			if err := json.Unmarshal([]byte(*embedRaw), &panel.Embed); err != nil {
				return nil, err
			}
		}

		panels = append(panels, panel)
	}

	return panels, nil
}

func (p *MultiPanelTable) Create(ctx context.Context, panel MultiPanel) (int, error) {
	query := `
INSERT INTO
	multi_panels("message_id", "channel_id", "guild_id", "select_menu", "select_menu_placeholder", "embed", "uses_components_v2", "components", "force_disabled", "name")
VALUES
	($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING
	"id"
;
`

	var embedRaw *string
	if panel.Embed != nil {
		embedRawBytes, err := json.Marshal(panel.Embed)
		if err != nil {
			return 0, err
		}

		embedRaw = ptr(string(embedRawBytes))
	}

	var multiPanelId int
	if err := p.QueryRow(ctx, query,
		panel.MessageId, panel.ChannelId, panel.GuildId, panel.SelectMenu, panel.SelectMenuPlaceholder, embedRaw, panel.UsesComponentsV2, panel.Components, panel.ForceDisabled, panel.Name,
	).Scan(&multiPanelId); err != nil {
		return 0, err
	}

	return multiPanelId, nil
}

func (p *MultiPanelTable) Update(ctx context.Context, multiPanelId int, multiPanel MultiPanel) (err error) {
	query := `
UPDATE multi_panels
	SET "message_id" = $2,
		"channel_id" = $3,
		"select_menu" = $4,
		"select_menu_placeholder" = $5,
		"embed" = $6,
		"uses_components_v2" = $7,
		"components" = $8,
		"force_disabled" = $9,
		"name" = $10
	WHERE
		"id" = $1
;`

	var embedRaw *string
	if multiPanel.Embed != nil {
		embedRawBytes, err := json.Marshal(multiPanel.Embed)
		if err != nil {
			return err
		}

		embedRaw = ptr(string(embedRawBytes))
	}

	_, err = p.Exec(ctx, query,
		multiPanelId, multiPanel.MessageId, multiPanel.ChannelId, multiPanel.SelectMenu, multiPanel.SelectMenuPlaceholder, embedRaw, multiPanel.UsesComponentsV2, multiPanel.Components, multiPanel.ForceDisabled, multiPanel.Name,
	)

	return
}

func (p *MultiPanelTable) UpdateMessageId(ctx context.Context, multiPanelId int, messageId uint64) (err error) {
	query := `
UPDATE multi_panels
SET "message_id" = $1
WHERE "id" = $2;
`

	_, err = p.Exec(ctx, query, messageId, multiPanelId)
	return
}

// SetForceDisabled sets force_disabled for all of a guild's multi-panels that use Components V2.
// Used by the premium downgrade sweep.
func (p *MultiPanelTable) SetForceDisabled(ctx context.Context, guildId uint64, forceDisabled bool) error {
	query := `UPDATE multi_panels SET "force_disabled" = $2 WHERE "guild_id" = $1 AND "uses_components_v2" = true;`
	_, err := p.Exec(ctx, query, guildId, forceDisabled)
	return err
}

func (p *MultiPanelTable) Delete(ctx context.Context, guildId uint64, multiPanelId int) (success bool, err error) {
	query := `
WITH deleted AS (
	DELETE FROM
		multi_panels
	WHERE
		"guild_id" = $1
		AND
		"id" = $2
	RETURNING *
)

SELECT
	COUNT(*)
FROM
	deleted
;
`

	var count int
	err = p.QueryRow(ctx, query, guildId, multiPanelId).Scan(&count)
	success = count > 0

	return
}
