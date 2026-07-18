package database

import (
	"context"

	"github.com/jackc/pgx/v4/pgxpool"
)

type KBArticleFeedbackTable struct {
	*pgxpool.Pool
}

func newKBArticleFeedback(db *pgxpool.Pool) *KBArticleFeedbackTable {
	return &KBArticleFeedbackTable{
		Pool: db,
	}
}

func (t KBArticleFeedbackTable) Schema() string {
	return `
CREATE TABLE IF NOT EXISTS kb_article_feedback(
	"id" serial NOT NULL,
	"guild_id" int8 NOT NULL,
	"article_id" int4 NOT NULL,
	"panel_id" int4 NULL,
	"user_id" int8 NOT NULL,
	"helpful" bool NOT NULL,
	"created_at" timestamptz NOT NULL DEFAULT now(),
	FOREIGN KEY("article_id") REFERENCES kb_articles("id") ON DELETE CASCADE,
	PRIMARY KEY("id")
);
CREATE UNIQUE INDEX IF NOT EXISTS kb_article_feedback_article_user_uindex ON kb_article_feedback("article_id", "user_id");
CREATE INDEX IF NOT EXISTS kb_article_feedback_guild_id_idx ON kb_article_feedback("guild_id");
`
}

// Set records a single feedback vote for an article. Feedback is deduplicated per
// (article, user): a user's first vote is kept and later clicks are ignored, so the
// count reflects distinct users rather than clicks.
func (t *KBArticleFeedbackTable) Set(ctx context.Context, guildId uint64, articleId int, panelId *int, userId uint64, helpful bool) error {
	query := `
INSERT INTO kb_article_feedback("guild_id", "article_id", "panel_id", "user_id", "helpful")
VALUES($1, $2, $3, $4, $5)
ON CONFLICT("article_id", "user_id") DO NOTHING;`

	_, err := t.Exec(ctx, query, guildId, articleId, panelId, userId, helpful)
	return err
}

// CountHelpful returns the number of distinct users who marked an article as helpful.
func (t *KBArticleFeedbackTable) CountHelpful(ctx context.Context, articleId int) (int, error) {
	query := `SELECT COUNT(*) FROM kb_article_feedback WHERE "article_id" = $1 AND "helpful" = true;`

	var count int
	if err := t.QueryRow(ctx, query, articleId).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}
