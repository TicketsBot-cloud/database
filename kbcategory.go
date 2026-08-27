package database

import (
	"context"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type KBCategory struct {
	Id       int     `json:"id"`
	GuildId  uint64  `json:"guild_id,string"`
	Name     string  `json:"name"`
	Emoji    *string `json:"emoji"`
	Position int     `json:"position"`
}

type KBCategoriesTable struct {
	*pgxpool.Pool
}

func newKBCategories(db *pgxpool.Pool) *KBCategoriesTable {
	return &KBCategoriesTable{
		Pool: db,
	}
}

func (t KBCategoriesTable) Schema() string {
	return `
CREATE TABLE IF NOT EXISTS kb_categories(
	"id" SERIAL,
	"guild_id" int8 NOT NULL,
	"name" varchar(50) NOT NULL,
	"emoji" varchar(64) DEFAULT NULL,
	"position" int4 NOT NULL DEFAULT 0,
	PRIMARY KEY("id"),
	UNIQUE("guild_id", "name")
);
CREATE INDEX IF NOT EXISTS kb_categories_guild_id_idx ON kb_categories("guild_id");
`
}

func (t *KBCategoriesTable) GetByGuild(ctx context.Context, guildId uint64) ([]KBCategory, error) {
	query := `
SELECT "id", "guild_id", "name", "emoji", "position"
FROM kb_categories
WHERE "guild_id" = $1
ORDER BY "position" ASC, "id" ASC;
`

	rows, err := t.Query(ctx, query, guildId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []KBCategory
	for rows.Next() {
		var cat KBCategory
		if err := rows.Scan(&cat.Id, &cat.GuildId, &cat.Name, &cat.Emoji, &cat.Position); err != nil {
			return nil, err
		}

		categories = append(categories, cat)
	}

	return categories, nil
}

func (t *KBCategoriesTable) Get(ctx context.Context, id int) (KBCategory, bool, error) {
	query := `
SELECT "id", "guild_id", "name", "emoji", "position"
FROM kb_categories
WHERE "id" = $1;
`

	var cat KBCategory
	err := t.QueryRow(ctx, query, id).Scan(&cat.Id, &cat.GuildId, &cat.Name, &cat.Emoji, &cat.Position)
	if err != nil {
		if err == pgx.ErrNoRows {
			return KBCategory{}, false, nil
		}
		return KBCategory{}, false, err
	}

	return cat, true, nil
}

func (t *KBCategoriesTable) Create(ctx context.Context, cat KBCategory) (int, error) {
	query := `
INSERT INTO kb_categories("guild_id", "name", "emoji", "position")
VALUES($1, $2, $3, $4)
RETURNING "id";
`

	var id int
	err := t.QueryRow(ctx, query, cat.GuildId, cat.Name, cat.Emoji, cat.Position).Scan(&id)
	return id, err
}

func (t *KBCategoriesTable) Update(ctx context.Context, cat KBCategory) error {
	query := `
UPDATE kb_categories
SET "name" = $2, "emoji" = $3, "position" = $4
WHERE "id" = $1 AND "guild_id" = $5;
`

	_, err := t.Exec(ctx, query, cat.Id, cat.Name, cat.Emoji, cat.Position, cat.GuildId)
	return err
}

func (t *KBCategoriesTable) Delete(ctx context.Context, guildId uint64, id int) error {
	query := `DELETE FROM kb_categories WHERE "id" = $1 AND "guild_id" = $2;`
	_, err := t.Exec(ctx, query, id, guildId)
	return err
}
