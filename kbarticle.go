package database

import (
	"context"
	"time"

	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type KBArticle struct {
	Id          int                    `json:"id"`
	GuildId     uint64                 `json:"guild_id,string"`
	Title       string                 `json:"title"`
	Slug        string                 `json:"slug"`
	Description *string                `json:"description"`
	Content     *string                `json:"content"`
	Embed       *CustomEmbedWithFields `json:"embed"`
	CategoryIds []int                  `json:"category_ids"`
	Keywords    []string               `json:"keywords"`
	Position    int                    `json:"position"`
	Published   bool                   `json:"published"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type KBArticlesTable struct {
	*pgxpool.Pool
}

func newKBArticles(db *pgxpool.Pool) *KBArticlesTable {
	return &KBArticlesTable{
		Pool: db,
	}
}

func (t KBArticlesTable) Schema() string {
	return `
CREATE TABLE IF NOT EXISTS kb_articles(
	"id" SERIAL,
	"guild_id" int8 NOT NULL,
	"title" varchar(100) NOT NULL,
	"slug" varchar(100) NOT NULL,
	"description" varchar(255) DEFAULT NULL,
	"content" text DEFAULT NULL CONSTRAINT kb_content_length CHECK (length(content) <= 4096),
	"embed" JSONB DEFAULT NULL,
	"category_ids" int4[] DEFAULT '{}',
	"keywords" text[] DEFAULT '{}',
	"position" int4 NOT NULL DEFAULT 0,
	"published" bool NOT NULL DEFAULT true,
	"search_vector" tsvector,
	"created_at" timestamptz NOT NULL DEFAULT NOW(),
	"updated_at" timestamptz NOT NULL DEFAULT NOW(),
	PRIMARY KEY("id"),
	UNIQUE("guild_id", "slug")
);
CREATE INDEX IF NOT EXISTS kb_articles_guild_id_idx ON kb_articles("guild_id");
CREATE INDEX IF NOT EXISTS kb_articles_keywords_idx ON kb_articles USING GIN("keywords");

CREATE INDEX IF NOT EXISTS kb_articles_search_idx ON kb_articles USING GIN("search_vector");

CREATE OR REPLACE FUNCTION kb_articles_search_vector_update() RETURNS trigger AS $$
BEGIN
	NEW.search_vector :=
		setweight(to_tsvector('english', coalesce(NEW.title, '')), 'A') ||
		setweight(to_tsvector('english', coalesce(NEW.content, '')), 'B') ||
		setweight(to_tsvector('english', coalesce(array_to_string(NEW.keywords, ' '), '')), 'C');
	RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS kb_articles_search_vector_trigger ON kb_articles;
CREATE TRIGGER kb_articles_search_vector_trigger
	BEFORE INSERT OR UPDATE ON kb_articles
	FOR EACH ROW EXECUTE FUNCTION kb_articles_search_vector_update();
`
}

var kbArticleColumns = `"id", "guild_id", "title", "slug", "description", "content", "embed", "category_ids", "keywords", "position", "published", "created_at", "updated_at"`

func scanKBArticle(row pgx.Row) (KBArticle, error) {
	var article KBArticle
	var embedRaw *string
	var categoryIds pgtype.Int4Array
	var keywords pgtype.TextArray

	err := row.Scan(
		&article.Id,
		&article.GuildId,
		&article.Title,
		&article.Slug,
		&article.Description,
		&article.Content,
		&embedRaw,
		&categoryIds,
		&keywords,
		&article.Position,
		&article.Published,
		&article.CreatedAt,
		&article.UpdatedAt,
	)
	if err != nil {
		return article, err
	}

	if embedRaw != nil {
		if err := json.UnmarshalFromString(*embedRaw, &article.Embed); err != nil {
			return article, err
		}
	}

	article.CategoryIds = make([]int, 0)
	if categoryIds.Status == pgtype.Present {
		for _, el := range categoryIds.Elements {
			if el.Status == pgtype.Present {
				article.CategoryIds = append(article.CategoryIds, int(el.Int))
			}
		}
	}

	article.Keywords = make([]string, 0)
	if keywords.Status == pgtype.Present {
		for _, el := range keywords.Elements {
			if el.Status == pgtype.Present {
				article.Keywords = append(article.Keywords, el.String)
			}
		}
	}

	return article, nil
}

func (t *KBArticlesTable) GetByGuild(ctx context.Context, guildId uint64) ([]KBArticle, error) {
	query := `
SELECT ` + kbArticleColumns + `
FROM kb_articles
WHERE "guild_id" = $1
ORDER BY "position" ASC, "id" ASC;
`

	rows, err := t.Query(ctx, query, guildId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var articles []KBArticle
	for rows.Next() {
		article, err := scanKBArticle(rows)
		if err != nil {
			return nil, err
		}
		articles = append(articles, article)
	}

	return articles, nil
}

func (t *KBArticlesTable) Get(ctx context.Context, id int) (KBArticle, bool, error) {
	query := `
SELECT ` + kbArticleColumns + `
FROM kb_articles
WHERE "id" = $1;
`

	article, err := scanKBArticle(t.QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return KBArticle{}, false, nil
		}
		return KBArticle{}, false, err
	}

	return article, true, nil
}

func (t *KBArticlesTable) GetBySlug(ctx context.Context, guildId uint64, slug string) (KBArticle, bool, error) {
	query := `
SELECT ` + kbArticleColumns + `
FROM kb_articles
WHERE "guild_id" = $1 AND "slug" = $2 AND "published" = true;
`

	article, err := scanKBArticle(t.QueryRow(ctx, query, guildId, slug))
	if err != nil {
		if err == pgx.ErrNoRows {
			return KBArticle{}, false, nil
		}
		return KBArticle{}, false, err
	}

	return article, true, nil
}

func (t *KBArticlesTable) GetByCategory(ctx context.Context, guildId uint64, categoryId int) ([]KBArticle, error) {
	query := `
SELECT ` + kbArticleColumns + `
FROM kb_articles
WHERE "guild_id" = $1 AND $2 = ANY("category_ids") AND "published" = true
ORDER BY "position" ASC, "id" ASC;
`

	rows, err := t.Query(ctx, query, guildId, categoryId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var articles []KBArticle
	for rows.Next() {
		article, err := scanKBArticle(rows)
		if err != nil {
			return nil, err
		}
		articles = append(articles, article)
	}

	return articles, nil
}

func (t *KBArticlesTable) Search(ctx context.Context, guildId uint64, query string, limit int) ([]KBArticle, error) {
	// Use full-text search with ts_rank for relevance, plus ILIKE fallback for partial matches.
	// Title matches (weight A) rank highest, then content (B), then keywords (C).
	q := `
SELECT ` + kbArticleColumns + `
FROM kb_articles
WHERE "guild_id" = $1 AND "published" = true
  AND (
    ("search_vector" IS NOT NULL AND "search_vector" @@ websearch_to_tsquery('english', $2))
    OR "title" ILIKE '%' || $2 || '%'
    OR "content" ILIKE '%' || $2 || '%'
    OR EXISTS (SELECT 1 FROM unnest("keywords") kw WHERE kw ILIKE '%' || $2 || '%')
  )
ORDER BY
  ts_rank(coalesce("search_vector", ''::tsvector), websearch_to_tsquery('english', $2)) DESC,
  CASE WHEN "title" ILIKE $2 || '%' THEN 0 ELSE 1 END,
  "position" ASC
LIMIT $3;
`

	rows, err := t.Query(ctx, q, guildId, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var articles []KBArticle
	for rows.Next() {
		article, err := scanKBArticle(rows)
		if err != nil {
			return nil, err
		}
		articles = append(articles, article)
	}

	return articles, nil
}

func (t *KBArticlesTable) SearchContaining(ctx context.Context, guildId uint64, substring string, limit int) ([]KBArticleSummary, error) {
	// Full-text search for autocomplete, with ILIKE fallback for partial/short queries
	query := `
SELECT "id", "title"
FROM kb_articles
WHERE "guild_id" = $1 AND "published" = true
  AND (
    ("search_vector" IS NOT NULL AND "search_vector" @@ websearch_to_tsquery('english', $2))
    OR "title" ILIKE '%' || $2 || '%'
  )
ORDER BY
  ts_rank(coalesce("search_vector", ''::tsvector), websearch_to_tsquery('english', $2)) DESC,
  CASE WHEN "title" ILIKE $2 || '%' THEN 0 ELSE 1 END
LIMIT $3;
`

	rows, err := t.Query(ctx, query, guildId, substring, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []KBArticleSummary
	for rows.Next() {
		var s KBArticleSummary
		if err := rows.Scan(&s.Id, &s.Title); err != nil {
			return nil, err
		}
		results = append(results, s)
	}

	return results, nil
}

// KBArticleSummary is a lightweight projection returned by autocomplete searches.
type KBArticleSummary struct {
	Id    int    `json:"id"`
	Title string `json:"title"`
}

func (t *KBArticlesTable) Create(ctx context.Context, article KBArticle) (int, error) {
	query := `
INSERT INTO kb_articles("guild_id", "title", "slug", "description", "content", "embed", "category_ids", "keywords", "position", "published")
VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING "id";
`

	var embedRaw *string
	if article.Embed != nil {
		tmp, err := json.MarshalToString(article.Embed)
		if err != nil {
			return 0, err
		}
		embedRaw = &tmp
	}

	categoryIds := toInt4Array(article.CategoryIds)
	keywords := toTextArray(article.Keywords)

	var id int
	err := t.QueryRow(ctx, query,
		article.GuildId,
		article.Title,
		article.Slug,
		article.Description,
		article.Content,
		embedRaw,
		categoryIds,
		keywords,
		article.Position,
		article.Published,
	).Scan(&id)
	return id, err
}

func (t *KBArticlesTable) Update(ctx context.Context, article KBArticle) error {
	query := `
UPDATE kb_articles
SET "title" = $2, "slug" = $3, "description" = $4, "content" = $5, "embed" = $6, "category_ids" = $7, "keywords" = $8, "position" = $9, "published" = $10, "updated_at" = NOW()
WHERE "id" = $1 AND "guild_id" = $11;
`

	var embedRaw *string
	if article.Embed != nil {
		tmp, err := json.MarshalToString(article.Embed)
		if err != nil {
			return err
		}
		embedRaw = &tmp
	}

	categoryIds := toInt4Array(article.CategoryIds)
	keywords := toTextArray(article.Keywords)

	_, err := t.Exec(ctx, query,
		article.Id,
		article.Title,
		article.Slug,
		article.Description,
		article.Content,
		embedRaw,
		categoryIds,
		keywords,
		article.Position,
		article.Published,
		article.GuildId,
	)
	return err
}

func (t *KBArticlesTable) Delete(ctx context.Context, guildId uint64, articleId int) error {
	query := `DELETE FROM kb_articles WHERE "guild_id" = $1 AND "id" = $2;`
	_, err := t.Exec(ctx, query, guildId, articleId)
	return err
}

func (t *KBArticlesTable) GetCountByGuild(ctx context.Context, guildId uint64) (int, error) {
	query := `SELECT COUNT(*) FROM kb_articles WHERE "guild_id" = $1;`
	var count int
	err := t.QueryRow(ctx, query, guildId).Scan(&count)
	return count, err
}

func toInt4Array(ints []int) pgtype.Int4Array {
	if len(ints) == 0 {
		return pgtype.Int4Array{
			Status: pgtype.Present,
		}
	}

	elements := make([]pgtype.Int4, len(ints))
	for i, v := range ints {
		elements[i] = pgtype.Int4{Int: int32(v), Status: pgtype.Present}
	}

	return pgtype.Int4Array{
		Elements:   elements,
		Dimensions: []pgtype.ArrayDimension{{Length: int32(len(ints)), LowerBound: 1}},
		Status:     pgtype.Present,
	}
}
