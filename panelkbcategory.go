package database

import (
	"context"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type PanelKBCategoriesTable struct {
	*pgxpool.Pool
}

func newPanelKBCategories(db *pgxpool.Pool) *PanelKBCategoriesTable {
	return &PanelKBCategoriesTable{
		Pool: db,
	}
}

func (t PanelKBCategoriesTable) Schema() string {
	return `
CREATE TABLE IF NOT EXISTS panel_kb_categories(
	"panel_id" int4 NOT NULL,
	"category_id" int4 NOT NULL,
	PRIMARY KEY("panel_id", "category_id")
);
`
}

func (t *PanelKBCategoriesTable) GetByPanel(ctx context.Context, panelId int) ([]int, error) {
	query := `SELECT "category_id" FROM panel_kb_categories WHERE "panel_id" = $1;`

	rows, err := t.Query(ctx, query, panelId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categoryIds []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		categoryIds = append(categoryIds, id)
	}

	return categoryIds, nil
}

func (t *PanelKBCategoriesTable) Set(ctx context.Context, panelId int, categoryIds []int) error {
	tx, err := t.Begin(ctx)
	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)

	if err := t.SetWithTx(ctx, tx, panelId, categoryIds); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (t *PanelKBCategoriesTable) SetWithTx(ctx context.Context, tx pgx.Tx, panelId int, categoryIds []int) error {
	// Remove existing category associations for this panel
	if _, err := tx.Exec(ctx, `DELETE FROM panel_kb_categories WHERE "panel_id" = $1;`, panelId); err != nil {
		return err
	}

	// Insert each provided category association
	for _, categoryId := range categoryIds {
		query := `INSERT INTO panel_kb_categories("panel_id", "category_id") VALUES($1, $2) ON CONFLICT("panel_id", "category_id") DO NOTHING;`
		if _, err := tx.Exec(ctx, query, panelId, categoryId); err != nil {
			return err
		}
	}

	return nil
}
