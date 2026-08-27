package database

import (
	"context"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type KBSettings struct {
	GuildId      uint64  `json:"guild_id,string"`
	PrimaryBg    *int    `json:"primary_bg"`
	CardBg       *int    `json:"card_bg"`
	TextColour   *int    `json:"text_colour"`
	AccentColour *int    `json:"accent_colour"`
	LogoUrl      *string `json:"logo_url"`
	HideBranding bool    `json:"hide_branding"`
}

type KBSettingsTable struct {
	*pgxpool.Pool
}

func newKBSettings(db *pgxpool.Pool) *KBSettingsTable {
	return &KBSettingsTable{
		Pool: db,
	}
}

// Schema no longer creates the custom domain columns. Databases that already ran
// an earlier version still have custom_domain, domain_verified and the
// domain_status family, plus their indexes; nothing reads or writes them.
//
// They are deliberately not dropped here. DROP COLUMN is irreversible and this
// file runs on every service boot, so removing them is a decision to take
// explicitly rather than a side effect of deploying.
func (t KBSettingsTable) Schema() string {
	return `
CREATE TABLE IF NOT EXISTS kb_settings(
	"guild_id" int8 NOT NULL PRIMARY KEY,
	"primary_bg" int4 DEFAULT NULL,
	"card_bg" int4 DEFAULT NULL,
	"text_colour" int4 DEFAULT NULL,
	"accent_colour" int4 DEFAULT NULL,
	"logo_url" text DEFAULT NULL,
	"hide_branding" bool NOT NULL DEFAULT false
);
`
}

func (t *KBSettingsTable) Get(ctx context.Context, guildId uint64) (KBSettings, bool, error) {
	query := `
SELECT "guild_id", "primary_bg", "card_bg", "text_colour", "accent_colour", "logo_url", "hide_branding"
FROM kb_settings
WHERE "guild_id" = $1;
`

	var settings KBSettings
	err := t.QueryRow(ctx, query, guildId).Scan(
		&settings.GuildId,
		&settings.PrimaryBg,
		&settings.CardBg,
		&settings.TextColour,
		&settings.AccentColour,
		&settings.LogoUrl,
		&settings.HideBranding,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return KBSettings{}, false, nil
		}
		return KBSettings{}, false, err
	}

	return settings, true, nil
}

func (t *KBSettingsTable) Set(ctx context.Context, settings KBSettings) error {
	query := `
INSERT INTO kb_settings("guild_id", "primary_bg", "card_bg", "text_colour", "accent_colour", "logo_url", "hide_branding")
VALUES($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT("guild_id") DO UPDATE SET
	"primary_bg" = $2,
	"card_bg" = $3,
	"text_colour" = $4,
	"accent_colour" = $5,
	"logo_url" = $6,
	"hide_branding" = $7;
`

	_, err := t.Exec(ctx, query,
		settings.GuildId,
		settings.PrimaryBg,
		settings.CardBg,
		settings.TextColour,
		settings.AccentColour,
		settings.LogoUrl,
		settings.HideBranding,
	)
	return err
}
