package database

import (
	"context"

	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Panel struct {
	PanelId                        int     `json:"panel_id"`
	MessageId                      uint64  `json:"message_id,string"`
	ChannelId                      uint64  `json:"channel_id,string"`
	GuildId                        uint64  `json:"guild_id,string"`
	Title                          string  `json:"title"`
	Content                        string  `json:"content"`
	Colour                         int32   `json:"colour"`
	TargetCategory                 uint64  `json:"category_id,string"`
	EmojiName                      *string `json:"emoji_name"`
	EmojiId                        *uint64 `json:"emoji_id,string"`
	WelcomeMessageEmbed            *int    `json:"welcome_message_embed"`
	WithDefaultTeam                bool    `json:"default_team"`
	CustomId                       string  `json:"custom_id"`
	ImageUrl                       *string `json:"image_url,omitempty"`
	ThumbnailUrl                   *string `json:"thumbnail_url,omitempty"`
	ButtonStyle                    int     `json:"button_style,string"`
	ButtonLabel                    string  `json:"button_label"`
	FormId                         *int    `json:"form_id"`
	NamingScheme                   *string `json:"naming_scheme"`
	ForceDisabled                  bool    `json:"force_disabled"`
	Disabled                       bool    `json:"disabled"`
	ExitSurveyFormId               *int    `json:"exit_survey_form_id"`
	PendingCategory                *uint64 `json:"pending_category,string"`
	MentionBehaviour               string  `json:"mention_behaviour"`
	TranscriptChannelId            *uint64 `json:"transcript_channel_id,string,omitempty"`
	UseThreads                     bool    `json:"use_threads"`
	TicketNotificationChannel      *uint64 `json:"ticket_notification_channel,string,omitempty"`
	CooldownSeconds                int     `json:"cooldown_seconds"`
	TicketLimit                    *uint8  `json:"ticket_limit,omitempty"`
	HideCloseButton                bool    `json:"hide_close_button"`
	HideCloseWithReasonButton      bool    `json:"hide_close_with_reason_button"`
	HideClaimButton                bool    `json:"hide_claim_button"`
	ShowInOpenCommand              bool    `json:"show_in_open_command"`
	StoreTranscripts               bool    `json:"store_transcripts"`
	OverflowEnabled                bool    `json:"overflow_enabled"`
	OverflowCategoryId             *uint64 `json:"overflow_category_id,string"`
	UsersCanClose                  bool    `json:"users_can_close"`
	CloseConfirmation              bool    `json:"close_confirmation"`
	FeedbackEnabled                bool    `json:"feedback_enabled"`
	SupportCanView                 bool    `json:"support_can_view"`
	SupportCanType                 bool    `json:"support_can_type"`
}

type PanelWithWelcomeMessage struct {
	Panel
	WelcomeMessage *CustomEmbed
}

type PanelTable struct {
	*pgxpool.Pool
}

func newPanelTable(db *pgxpool.Pool) *PanelTable {
	return &PanelTable{
		db,
	}
}

// TODO: Make custom_id unique
func (p PanelTable) Schema() string {
	return `
CREATE TABLE IF NOT EXISTS panels(
	"panel_id" SERIAL NOT NULL UNIQUE,
	"message_id" int8 NOT NULL UNIQUE,
	"channel_id" int8 NOT NULL,
	"guild_id" int8 NOT NULL,
	"title" varchar(255) NOT NULL,
	"content" text NOT NULL,
	"colour" int4 NOT NULL,
	"target_category" int8 NOT NULL,
	"emoji_name" varchar(32) DEFAULT NULL,
	"emoji_id" int8 DEFAULT NULL,
	"welcome_message" int NULL,
	"default_team" bool NOT NULL,
	"custom_id" varchar(100) NOT NULL,
	"image_url" varchar(255),
	"thumbnail_url" varchar(255),
	"button_style" int2 DEFAULT 1,
	"button_label" varchar(80) NOT NULL,
	"form_id" int DEFAULT NULL,
	"naming_scheme" varchar(100) DEFAULT NULL,
	"force_disabled" bool NOT NULL DEFAULT false,
	"disabled" bool NOT NULL DEFAULT false,
	"exit_survey_form_id" int DEFAULT NULL,
	"pending_category" int8 DEFAULT NULL,
	"mention_behaviour" varchar(20) NOT NULL DEFAULT 'none',
	"transcript_channel_id" int8 DEFAULT NULL,
	"use_threads" bool NOT NULL DEFAULT false,
	"ticket_notification_channel" int8 DEFAULT NULL,
	"cooldown_seconds" int NOT NULL DEFAULT 0,
	"ticket_limit" int2 DEFAULT NULL,
	"hide_close_button" bool NOT NULL DEFAULT false,
	"hide_close_with_reason_button" bool NOT NULL DEFAULT false,
	"hide_claim_button" bool NOT NULL DEFAULT false,
	"show_in_open_command" bool NOT NULL DEFAULT false,
	"store_transcripts" bool NOT NULL DEFAULT true,
	"overflow_enabled" bool NOT NULL DEFAULT false,
	"overflow_category_id" int8 DEFAULT NULL,
	"users_can_close" bool NOT NULL DEFAULT true,
	"close_confirmation" bool NOT NULL DEFAULT true,
	"feedback_enabled" bool NOT NULL DEFAULT false,
	"support_can_view" bool NOT NULL DEFAULT true,
	"support_can_type" bool NOT NULL DEFAULT false,
	FOREIGN KEY ("welcome_message") REFERENCES embeds("id") ON DELETE SET NULL,
	FOREIGN KEY ("form_id") REFERENCES forms("form_id"),
	FOREIGN KEY ("exit_survey_form_id") REFERENCES forms("form_id"),
	PRIMARY KEY("panel_id")
);
CREATE INDEX IF NOT EXISTS panels_guild_id ON panels("guild_id");
CREATE INDEX IF NOT EXISTS panels_message_id ON panels("message_id");
CREATE INDEX IF NOT EXISTS panels_form_id ON panels("form_id");
CREATE INDEX IF NOT EXISTS panels_guild_id_form_id ON panels("guild_id", "form_id");
CREATE INDEX IF NOT EXISTS panels_custom_id ON panels("custom_id");`
}

func (p *PanelTable) Get(ctx context.Context, messageId uint64) (panel Panel, e error) {
	query := `
SELECT
	panel_id,
	message_id,
	channel_id,
	guild_id,
	title,
	content,
	colour,
	target_category,
	emoji_name,
	emoji_id,
	welcome_message,
	default_team,
	custom_id,
	image_url,
	thumbnail_url,
	button_style,
	button_label,
	form_id,
	naming_scheme,
	force_disabled,
	disabled,
	exit_survey_form_id,
	pending_category,
	mention_behaviour,
	transcript_channel_id,
	use_threads,
	ticket_notification_channel,
	cooldown_seconds,
	ticket_limit,
	hide_close_button,
	hide_close_with_reason_button,
	hide_claim_button,
	show_in_open_command,
	store_transcripts,
	overflow_enabled,
	overflow_category_id,
	users_can_close,
	close_confirmation,
	feedback_enabled,
	support_can_view,
	support_can_type
FROM panels
WHERE "message_id" = $1;
`

	if err := p.QueryRow(ctx, query, messageId).
		Scan(panel.fieldPtrs()...); err != nil && err != pgx.ErrNoRows {
		e = err
	}

	return
}

func (p *PanelTable) GetById(ctx context.Context, panelId int) (panel Panel, e error) {
	query := `
SELECT
	panel_id,
	message_id,
	channel_id,
	guild_id,
	title,
	content,
	colour,
	target_category,
	emoji_name,
	emoji_id,
	welcome_message,
	default_team,
	custom_id,
	image_url,
	thumbnail_url,
	button_style,
	button_label,
	form_id,
	naming_scheme,
	force_disabled,
	disabled,
	exit_survey_form_id,
	pending_category,
	mention_behaviour,
	transcript_channel_id,
	use_threads,
	ticket_notification_channel,
	cooldown_seconds,
	ticket_limit,
	hide_close_button,
	hide_close_with_reason_button,
	hide_claim_button,
	show_in_open_command,
	store_transcripts,
	overflow_enabled,
	overflow_category_id,
	users_can_close,
	close_confirmation,
	feedback_enabled,
	support_can_view,
	support_can_type
FROM panels
WHERE "panel_id" = $1;
`

	if err := p.QueryRow(ctx, query, panelId).
		Scan(panel.fieldPtrs()...); err != nil && err != pgx.ErrNoRows {
		e = err
	}

	return
}

func (p *PanelTable) GetByIdWithWelcomeMessage(ctx context.Context, guildId uint64, panelId int) (panel *PanelWithWelcomeMessage, e error) {
	query := `
SELECT
	panels.panel_id,
	panels.message_id,
	panels.channel_id,
	panels.guild_id,
	panels.title,
	panels.content,
	panels.colour,
	panels.target_category,
	panels.emoji_name,
	panels.emoji_id,
	panels.welcome_message,
	panels.default_team,
	panels.custom_id,
	panels.image_url,
	panels.thumbnail_url,
	panels.button_style,
	panels.button_label,
	panels.form_id,
	panels.naming_scheme,
	panels.force_disabled,
	panels.disabled,
	panels.exit_survey_form_id,
	panels.pending_category,
	panels.mention_behaviour,
	panels.transcript_channel_id,
	panels.use_threads,
	panels.ticket_notification_channel,
	panels.cooldown_seconds,
	panels.ticket_limit,
	panels.hide_close_button,
	panels.hide_close_with_reason_button,
	panels.hide_claim_button,
	panels.show_in_open_command,
	panels.store_transcripts,
	panels.overflow_enabled,
	panels.overflow_category_id,
	panels.users_can_close,
	panels.close_confirmation,
	panels.feedback_enabled,
	panels.support_can_view,
	panels.support_can_type,
	embeds.id,
	embeds.guild_id,
	embeds.title,
	embeds.description,
	embeds.url,
	embeds.colour,
	embeds.author_name,
	embeds.author_icon_url,
	embeds.author_url,
	embeds.image_url,
	embeds.thumbnail_url,
	embeds.footer_text,
	embeds.footer_icon_url,
	embeds.timestamp
FROM panels
LEFT JOIN embeds
ON panels.welcome_message = embeds.id
WHERE panels.guild_id = $1
AND panels.panel_id = $2;`

	rows, err := p.Query(ctx, query, guildId, panelId)
	defer rows.Close()
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var pan Panel
		var embed CustomEmbed

		// Can't scan missing values into non-nullable fields
		var embedId *int
		var embedGuildId *uint64
		var embedColour *uint32

		err := rows.Scan(append(pan.fieldPtrs(),
			&embedId,
			&embedGuildId,
			&embed.Title,
			&embed.Description,
			&embed.Url,
			&embedColour,
			&embed.AuthorName,
			&embed.AuthorIconUrl,
			&embed.AuthorUrl,
			&embed.ImageUrl,
			&embed.ThumbnailUrl,
			&embed.FooterText,
			&embed.FooterIconUrl,
			&embed.Timestamp,
		)...)

		if err != nil {
			return nil, err
		}

		var embedPtr *CustomEmbed
		if embedId != nil {
			embed.Id = *embedId
			embed.GuildId = *embedGuildId
			embed.Colour = *embedColour

			embedPtr = &embed
		}

		panel = &PanelWithWelcomeMessage{
			Panel:          pan,
			WelcomeMessage: embedPtr,
		}
	}

	return
}

func (p *PanelTable) GetByCustomId(ctx context.Context, guildId uint64, customId string) (panel Panel, ok bool, e error) {
	query := `
SELECT
	panel_id,
	message_id,
	channel_id,
	guild_id,
	title,
	content,
	colour,
	target_category,
	emoji_name,
	emoji_id,
	welcome_message,
	default_team,
	custom_id,
	image_url,
	thumbnail_url,
	button_style,
	button_label,
	form_id,
	naming_scheme,
	force_disabled,
	disabled,
	exit_survey_form_id,
	pending_category,
	mention_behaviour,
	transcript_channel_id,
	use_threads,
	ticket_notification_channel,
	cooldown_seconds,
	ticket_limit,
	hide_close_button,
	hide_close_with_reason_button,
	hide_claim_button,
	show_in_open_command,
	store_transcripts,
	overflow_enabled,
	overflow_category_id,
	users_can_close,
	close_confirmation,
	feedback_enabled,
	support_can_view,
	support_can_type
FROM panels
WHERE "guild_id" = $1 AND "custom_id" = $2;
`

	switch err := p.QueryRow(ctx, query, guildId, customId).Scan(panel.fieldPtrs()...); err {
	case nil:
		ok = true
	case pgx.ErrNoRows:
	default:
		e = err
	}

	return
}

func (p *PanelTable) GetByFormId(ctx context.Context, guildId uint64, formId int) (panel Panel, ok bool, e error) {
	query := `
SELECT
	panel_id,
	message_id,
	channel_id,
	guild_id,
	title,
	content,
	colour,
	target_category,
	emoji_name,
	emoji_id,
	welcome_message,
	default_team,
	custom_id,
	image_url,
	thumbnail_url,
	button_style,
	button_label,
	form_id,
	naming_scheme,
	force_disabled,
	disabled,
	exit_survey_form_id,
	pending_category,
	mention_behaviour,
	transcript_channel_id,
	use_threads,
	ticket_notification_channel,
	cooldown_seconds,
	ticket_limit,
	hide_close_button,
	hide_close_with_reason_button,
	hide_claim_button,
	show_in_open_command,
	store_transcripts,
	overflow_enabled,
	overflow_category_id,
	users_can_close,
	close_confirmation,
	feedback_enabled,
	support_can_view,
	support_can_type
FROM panels
WHERE "guild_id" = $1 AND "form_id" = $2;
`

	switch err := p.QueryRow(ctx, query, guildId, formId).Scan(panel.fieldPtrs()...); err {
	case nil:
		ok = true
	case pgx.ErrNoRows:
	default:
		e = err
	}

	return
}

func (p *PanelTable) GetByFormCustomId(ctx context.Context, guildId uint64, customId string) (panel Panel, ok bool, e error) {
	query := `
SELECT
	panels.panel_id,
	panels.message_id,
	panels.channel_id,
	panels.guild_id,
	panels.title,
	panels.content,
	panels.colour,
	panels.target_category,
	panels.emoji_name,
	panels.emoji_id,
	panels.welcome_message,
	panels.default_team,
	panels.custom_id,
	panels.image_url,
	panels.thumbnail_url,
	panels.button_style,
	panels.button_label,
	panels.form_id,
	panels.naming_scheme,
	panels.force_disabled,
	panels.disabled,
	panels.exit_survey_form_id,
	panels.pending_category,
	panels.mention_behaviour,
	panels.transcript_channel_id,
	panels.use_threads,
	panels.ticket_notification_channel,
	panels.cooldown_seconds,
	panels.ticket_limit,
	panels.hide_close_button,
	panels.hide_close_with_reason_button,
	panels.hide_claim_button,
	panels.show_in_open_command,
	panels.store_transcripts,
	panels.overflow_enabled,
	panels.overflow_category_id,
	panels.users_can_close,
	panels.close_confirmation,
	panels.feedback_enabled,
	panels.support_can_view,
	panels.support_can_type
FROM panels
INNER JOIN forms
ON forms.form_id = panels.form_id
WHERE forms.guild_id = $1 AND forms.custom_id = $2;
`

	switch err := p.QueryRow(ctx, query, guildId, customId).Scan(panel.fieldPtrs()...); err {
	case nil:
		ok = true
	case pgx.ErrNoRows:
	default:
		e = err
	}

	return
}

func (p *PanelTable) GetByGuild(ctx context.Context, guildId uint64) (panels []Panel, e error) {
	query := `
SELECT
	panel_id,
	message_id,
	channel_id,
	guild_id,
	title,
	content,
	colour,
	target_category,
	emoji_name,
	emoji_id,
	welcome_message,
	default_team,
	custom_id,
	image_url,
	thumbnail_url,
	button_style,
	button_label,
	form_id,
	naming_scheme,
	force_disabled,
	disabled,
	exit_survey_form_id,
	pending_category,
	mention_behaviour,
	transcript_channel_id,
	use_threads,
	ticket_notification_channel,
	cooldown_seconds,
	ticket_limit,
	hide_close_button,
	hide_close_with_reason_button,
	hide_claim_button,
	show_in_open_command,
	store_transcripts,
	overflow_enabled,
	overflow_category_id,
	users_can_close,
	close_confirmation,
	feedback_enabled,
	support_can_view,
	support_can_type
FROM panels
WHERE "guild_id" = $1
ORDER BY "panel_id" ASC;`

	rows, err := p.Query(ctx, query, guildId)
	defer rows.Close()
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var panel Panel
		if err := rows.Scan(panel.fieldPtrs()...); err != nil {
			return nil, err
		}

		panels = append(panels, panel)
	}

	return
}

func (p *PanelTable) GetByGuildWithWelcomeMessage(ctx context.Context, guildId uint64) (panels []PanelWithWelcomeMessage, e error) {
	query := `
SELECT
	panels.panel_id,
	panels.message_id,
	panels.channel_id,
	panels.guild_id,
	panels.title,
	panels.content,
	panels.colour,
	panels.target_category,
	panels.emoji_name,
	panels.emoji_id,
	panels.welcome_message,
	panels.default_team,
	panels.custom_id,
	panels.image_url,
	panels.thumbnail_url,
	panels.button_style,
	panels.button_label,
	panels.form_id,
	panels.naming_scheme,
	panels.force_disabled,
	panels.disabled,
	panels.exit_survey_form_id,
	panels.pending_category,
	panels.mention_behaviour,
	panels.transcript_channel_id,
	panels.use_threads,
	panels.ticket_notification_channel,
	panels.cooldown_seconds,
	panels.ticket_limit,
	panels.hide_close_button,
	panels.hide_close_with_reason_button,
	panels.hide_claim_button,
	panels.show_in_open_command,
	panels.store_transcripts,
	panels.overflow_enabled,
	panels.overflow_category_id,
	panels.users_can_close,
	panels.close_confirmation,
	panels.feedback_enabled,
	panels.support_can_view,
	panels.support_can_type,
	embeds.id,
	embeds.guild_id,
	embeds.title,
	embeds.description,
	embeds.url,
	embeds.colour,
	embeds.author_name,
	embeds.author_icon_url,
	embeds.author_url,
	embeds.image_url,
	embeds.thumbnail_url,
	embeds.footer_text,
	embeds.footer_icon_url,
	embeds.timestamp
FROM panels
LEFT JOIN embeds
ON panels.welcome_message = embeds.id
WHERE panels.guild_id = $1
ORDER BY panels.panel_id ASC;`

	rows, err := p.Query(ctx, query, guildId)
	defer rows.Close()
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var panel Panel
		var embed CustomEmbed

		// Can't scan missing values into non-nullable fields
		var embedId *int
		var embedGuildId *uint64
		var embedColour *uint32

		err := rows.Scan(append(panel.fieldPtrs(),
			&embedId,
			&embedGuildId,
			&embed.Title,
			&embed.Description,
			&embed.Url,
			&embedColour,
			&embed.AuthorName,
			&embed.AuthorIconUrl,
			&embed.AuthorUrl,
			&embed.ImageUrl,
			&embed.ThumbnailUrl,
			&embed.FooterText,
			&embed.FooterIconUrl,
			&embed.Timestamp,
		)...)

		if err != nil {
			return nil, err
		}

		var embedPtr *CustomEmbed
		if embedId != nil {
			embed.Id = *embedId
			embed.GuildId = *embedGuildId
			embed.Colour = *embedColour

			embedPtr = &embed
		}

		panels = append(panels, PanelWithWelcomeMessage{
			Panel:          panel,
			WelcomeMessage: embedPtr,
		})
	}

	return
}

func (p *PanelTable) GetPanelCount(ctx context.Context, guildId uint64) (count int, err error) {
	query := `SELECT COUNT(*) FROM panels WHERE "guild_id" = $1;`

	err = p.QueryRow(ctx, query, guildId).Scan(&count)
	return
}

func (p *PanelTable) Create(ctx context.Context, panel Panel) (int, error) {
	tx, err := p.Begin(ctx)
	if err != nil {
		return 0, err
	}

	defer tx.Rollback(ctx)

	return p.CreateWithTx(ctx, tx, panel)
}

func (p *PanelTable) CreateWithTx(ctx context.Context, tx pgx.Tx, panel Panel) (panelId int, err error) {
	query := `
INSERT INTO panels(
	"message_id",
	"channel_id",
	"guild_id",
	"title",
	"content",
	"colour",
	"target_category",
	"emoji_name",
	"emoji_id",
	"welcome_message",
	"default_team",
	"custom_id",
	"image_url",
	"thumbnail_url",
	"button_style",
	"button_label",
	"form_id",
	"naming_scheme",
    "force_disabled",
	"disabled",
    "exit_survey_form_id",
	"pending_category",
	"mention_behaviour",
	"transcript_channel_id",
	"use_threads",
	"ticket_notification_channel",
	"cooldown_seconds",
	"ticket_limit",
	"hide_close_button",
	"hide_close_with_reason_button",
	"hide_claim_button",
	"show_in_open_command",
	"store_transcripts",
	"overflow_enabled",
	"overflow_category_id",
	"users_can_close",
	"close_confirmation",
	"feedback_enabled",
	"support_can_view",
	"support_can_type"
)
VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34, $35, $36, $37, $38, $39, $40)
ON CONFLICT("message_id") DO NOTHING
RETURNING "panel_id";`

	err = tx.QueryRow(ctx, query,
		panel.MessageId,
		panel.ChannelId,
		panel.GuildId,
		panel.Title,
		panel.Content,
		panel.Colour,
		panel.TargetCategory,
		panel.EmojiName,
		panel.EmojiId,
		panel.WelcomeMessageEmbed,
		panel.WithDefaultTeam,
		panel.CustomId,
		panel.ImageUrl,
		panel.ThumbnailUrl,
		panel.ButtonStyle,
		panel.ButtonLabel,
		panel.FormId,
		panel.NamingScheme,
		panel.ForceDisabled,
		panel.Disabled,
		panel.ExitSurveyFormId,
		panel.PendingCategory,
		panel.MentionBehaviour,
		panel.TranscriptChannelId,
		panel.UseThreads,
		panel.TicketNotificationChannel,
		panel.CooldownSeconds,
		panel.TicketLimit,
		panel.HideCloseButton,
		panel.HideCloseWithReasonButton,
		panel.HideClaimButton,
		panel.ShowInOpenCommand,
		panel.StoreTranscripts,
		panel.OverflowEnabled,
		panel.OverflowCategoryId,
		panel.UsersCanClose,
		panel.CloseConfirmation,
		panel.FeedbackEnabled,
		panel.SupportCanView,
		panel.SupportCanType,
	).Scan(&panelId)

	return
}

func (p *PanelTable) Update(ctx context.Context, panel Panel) (err error) {
	tx, err := p.Begin(ctx)
	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)

	if err := p.UpdateWithTx(ctx, tx, panel); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (p *PanelTable) UpdateWithTx(ctx context.Context, tx pgx.Tx, panel Panel) error {
	query := `
UPDATE panels
	SET "message_id" = $2,
		"channel_id" = $3,
		"title" = $4,
		"content" = $5,
		"colour" = $6,
		"target_category" = $7,
		"emoji_name" = $8,
		"emoji_id" = $9,
		"welcome_message" = $10,
		"default_team" = $11,
		"custom_id" = $12,
		"image_url" = $13,
		"thumbnail_url" = $14,
		"button_style" = $15,
		"button_label" = $16,
		"form_id" = $17,
		"naming_scheme" = $18,
	    "force_disabled" = $19,
	    "disabled" = $20,
	    "exit_survey_form_id" = $21,
	    "pending_category" = $22,
		"mention_behaviour" = $23,
		"transcript_channel_id" = $24,
		"use_threads" = $25,
		"ticket_notification_channel" = $26,
		"cooldown_seconds" = $27,
		"ticket_limit" = $28,
		"hide_close_button" = $29,
		"hide_close_with_reason_button" = $30,
		"hide_claim_button" = $31,
		"show_in_open_command" = $32,
		"store_transcripts" = $33,
		"overflow_enabled" = $34,
		"overflow_category_id" = $35,
		"users_can_close" = $36,
		"close_confirmation" = $37,
		"feedback_enabled" = $38,
		"support_can_view" = $39,
		"support_can_type" = $40
	WHERE
		"panel_id" = $1
;`

	_, err := tx.Exec(ctx, query,
		panel.PanelId,
		panel.MessageId,
		panel.ChannelId,
		panel.Title,
		panel.Content,
		panel.Colour,
		panel.TargetCategory,
		panel.EmojiName,
		panel.EmojiId,
		panel.WelcomeMessageEmbed,
		panel.WithDefaultTeam,
		panel.CustomId,
		panel.ImageUrl,
		panel.ThumbnailUrl,
		panel.ButtonStyle,
		panel.ButtonLabel,
		panel.FormId,
		panel.NamingScheme,
		panel.ForceDisabled,
		panel.Disabled,
		panel.ExitSurveyFormId,
		panel.PendingCategory,
		panel.MentionBehaviour,
		panel.TranscriptChannelId,
		panel.UseThreads,
		panel.TicketNotificationChannel,
		panel.CooldownSeconds,
		panel.TicketLimit,
		panel.HideCloseButton,
		panel.HideCloseWithReasonButton,
		panel.HideClaimButton,
		panel.ShowInOpenCommand,
		panel.StoreTranscripts,
		panel.OverflowEnabled,
		panel.OverflowCategoryId,
		panel.UsersCanClose,
		panel.CloseConfirmation,
		panel.FeedbackEnabled,
		panel.SupportCanView,
		panel.SupportCanType,
	)

	return err
}

func (p *PanelTable) SetOverflow(ctx context.Context, panelId int, enabled bool, categoryId *uint64) (err error) {
	query := `UPDATE panels SET "overflow_enabled" = $2, "overflow_category_id" = $3 WHERE "panel_id" = $1;`
	_, err = p.Exec(ctx, query, panelId, enabled, categoryId)
	return
}

func (p *PanelTable) UpdateMessageId(ctx context.Context, panelId int, messageId uint64) (err error) {
	query := `
UPDATE panels
SET "message_id" = $1
WHERE "panel_id" = $2;
`

	_, err = p.Exec(ctx, query, messageId, panelId)
	return
}

func (p *PanelTable) EnableAll(ctx context.Context, guildId uint64) (err error) {
	query := `
UPDATE panels
SET "force_disabled" = false
WHERE "guild_id" = $1;
`

	_, err = p.Exec(ctx, query, guildId)
	return
}

func (p *PanelTable) DisableSome(ctx context.Context, guildId uint64, freeLimit int) error {
	txOpts := pgx.TxOptions{
		IsoLevel:       pgx.Serializable,
		AccessMode:     pgx.ReadWrite,
		DeferrableMode: pgx.NotDeferrable,
	}

	tx, err := p.BeginTx(ctx, txOpts)
	if err != nil {
		return err
	}

	var panelCount int
	{
		query := `SELECT COUNT(*) FROM panels WHERE guild_id = $1 and "force_disabled" = false;`
		if err := tx.QueryRow(ctx, query, guildId).Scan(&panelCount); err != nil {
			return err
		}
	}

	if panelCount > freeLimit {
		// Find panels to disable
		query := `SELECT "panel_id" FROM panels WHERE guild_id = $1 and "force_disabled" = false ORDER BY "panel_id" DESC LIMIT $2;`
		rows, err := tx.Query(ctx, query, guildId, panelCount-freeLimit)
		if err != nil {
			return err
		}

		var toDisable []int
		for rows.Next() {
			var panelId int
			if err := rows.Scan(&panelId); err != nil {
				return err
			}

			toDisable = append(toDisable, panelId)
		}

		// Disable panels
		if len(toDisable) > 0 {
			query := `UPDATE panels SET "force_disabled" = true WHERE "panel_id" = ANY($1) AND "guild_id" = $2;`

			idArray := &pgtype.Int4Array{}
			if err := idArray.Set(toDisable); err != nil {
				return err
			}

			if _, err := tx.Exec(ctx, query, idArray, guildId); err != nil {
				return err
			}
		}
	}

	return tx.Commit(ctx)
}

func (p *PanelTable) Delete(ctx context.Context, panelId int) (err error) {
	query := `DELETE FROM panels WHERE "panel_id"=$1;`
	_, err = p.Exec(ctx, query, panelId)
	return
}

func (p *Panel) fieldPtrs() []interface{} {
	return []interface{}{
		&p.PanelId,
		&p.MessageId,
		&p.ChannelId,
		&p.GuildId,
		&p.Title,
		&p.Content,
		&p.Colour,
		&p.TargetCategory,
		&p.EmojiName,
		&p.EmojiId,
		&p.WelcomeMessageEmbed,
		&p.WithDefaultTeam,
		&p.CustomId,
		&p.ImageUrl,
		&p.ThumbnailUrl,
		&p.ButtonStyle,
		&p.ButtonLabel,
		&p.FormId,
		&p.NamingScheme,
		&p.ForceDisabled,
		&p.Disabled,
		&p.ExitSurveyFormId,
		&p.PendingCategory,
		&p.MentionBehaviour,
		&p.TranscriptChannelId,
		&p.UseThreads,
		&p.TicketNotificationChannel,
		&p.CooldownSeconds,
		&p.TicketLimit,
		&p.HideCloseButton,
		&p.HideCloseWithReasonButton,
		&p.HideClaimButton,
		&p.ShowInOpenCommand,
		&p.StoreTranscripts,
		&p.OverflowEnabled,
		&p.OverflowCategoryId,
		&p.UsersCanClose,
		&p.CloseConfirmation,
		&p.FeedbackEnabled,
		&p.SupportCanView,
		&p.SupportCanType,
	}
}
