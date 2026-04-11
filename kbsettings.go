package database

import (
	"context"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type KBSettings struct {
	GuildId        uint64  `json:"guild_id,string"`
	PrimaryBg      *int    `json:"primary_bg"`
	CardBg         *int    `json:"card_bg"`
	TextColour     *int    `json:"text_colour"`
	AccentColour   *int    `json:"accent_colour"`
	LogoUrl        *string `json:"logo_url"`
	HideBranding   bool    `json:"hide_branding"`
	CustomDomain   *string `json:"custom_domain"`
	DomainVerified bool    `json:"domain_verified"`
}

type KBSettingsTable struct {
	*pgxpool.Pool
}

func newKBSettings(db *pgxpool.Pool) *KBSettingsTable {
	return &KBSettingsTable{
		Pool: db,
	}
}

func (t KBSettingsTable) Schema() string {
	return `
CREATE TABLE IF NOT EXISTS kb_settings(
	"guild_id" int8 NOT NULL PRIMARY KEY,
	"primary_bg" int4 DEFAULT NULL,
	"card_bg" int4 DEFAULT NULL,
	"text_colour" int4 DEFAULT NULL,
	"accent_colour" int4 DEFAULT NULL,
	"logo_url" text DEFAULT NULL,
	"hide_branding" bool NOT NULL DEFAULT false,
	"custom_domain" text DEFAULT NULL,
	"domain_verified" bool NOT NULL DEFAULT false
);
CREATE INDEX IF NOT EXISTS kb_settings_custom_domain_idx ON kb_settings("custom_domain") WHERE "custom_domain" IS NOT NULL AND "domain_verified" = true;
`
}

func (t *KBSettingsTable) Get(ctx context.Context, guildId uint64) (KBSettings, bool, error) {
	query := `
SELECT "guild_id", "primary_bg", "card_bg", "text_colour", "accent_colour", "logo_url", "hide_branding", "custom_domain", "domain_verified"
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
		&settings.CustomDomain,
		&settings.DomainVerified,
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
INSERT INTO kb_settings("guild_id", "primary_bg", "card_bg", "text_colour", "accent_colour", "logo_url", "hide_branding", "custom_domain", "domain_verified")
VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT("guild_id") DO UPDATE SET
	"primary_bg" = $2,
	"card_bg" = $3,
	"text_colour" = $4,
	"accent_colour" = $5,
	"logo_url" = $6,
	"hide_branding" = $7,
	"custom_domain" = $8,
	"domain_verified" = $9;
`

	_, err := t.Exec(ctx, query,
		settings.GuildId,
		settings.PrimaryBg,
		settings.CardBg,
		settings.TextColour,
		settings.AccentColour,
		settings.LogoUrl,
		settings.HideBranding,
		settings.CustomDomain,
		settings.DomainVerified,
	)
	return err
}

// GetGuildByDomain returns the guild ID that has claimed a given custom domain (verified or not).
// Returns 0, false if no guild has this domain.
func (t *KBSettingsTable) GetGuildByDomain(ctx context.Context, domain string) (uint64, bool, error) {
	query := `SELECT "guild_id" FROM kb_settings WHERE "custom_domain" = $1 LIMIT 1;`
	var guildId uint64
	err := t.QueryRow(ctx, query, domain).Scan(&guildId)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, false, nil
		}
		return 0, false, err
	}
	return guildId, true, nil
}

// GetAllVerifiedDomains returns all verified custom KB domains.
func (t *KBSettingsTable) GetAllVerifiedDomains(ctx context.Context) ([]string, error) {
	query := `SELECT "custom_domain" FROM kb_settings WHERE "custom_domain" IS NOT NULL AND "domain_verified" = true;`
	rows, err := t.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var domains []string
	for rows.Next() {
		var domain string
		if err := rows.Scan(&domain); err != nil {
			return nil, err
		}
		domains = append(domains, domain)
	}
	return domains, nil
}
