package database

import (
	"context"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type GalleryListingTagsTable struct {
	*pgxpool.Pool
}

func newGalleryListingTagsTable(pool *pgxpool.Pool) *GalleryListingTagsTable {
	return &GalleryListingTagsTable{
		pool,
	}
}

func (t GalleryListingTagsTable) Schema() string {
	return `
CREATE TABLE IF NOT EXISTS gallery_listing_tags (
	"listing_id"  INT NOT NULL REFERENCES gallery_listings("id") ON DELETE CASCADE,
	"tag"         VARCHAR(30) NOT NULL,
	PRIMARY KEY ("listing_id", "tag")
);
CREATE INDEX IF NOT EXISTS gallery_listing_tags_tag_idx ON gallery_listing_tags("tag");
`
}

// Set replaces all tags for a listing within a transaction.
func (t *GalleryListingTagsTable) Set(ctx context.Context, listingId int, tags []string) error {
	tx, err := t.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := t.deleteByListingWithTx(ctx, tx, listingId); err != nil {
		return err
	}

	for _, tag := range tags {
		if err := t.insertWithTx(ctx, tx, listingId, tag); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (t *GalleryListingTagsTable) deleteByListingWithTx(ctx context.Context, tx pgx.Tx, listingId int) error {
	query := `DELETE FROM gallery_listing_tags WHERE "listing_id" = $1;`
	_, err := tx.Exec(ctx, query, listingId)
	return err
}

func (t *GalleryListingTagsTable) insertWithTx(ctx context.Context, tx pgx.Tx, listingId int, tag string) error {
	query := `INSERT INTO gallery_listing_tags ("listing_id", "tag") VALUES ($1, $2) ON CONFLICT DO NOTHING;`
	_, err := tx.Exec(ctx, query, listingId, tag)
	return err
}

// GetByListing returns all tags for a single listing.
func (t *GalleryListingTagsTable) GetByListing(ctx context.Context, listingId int) ([]string, error) {
	query := `SELECT "tag" FROM gallery_listing_tags WHERE "listing_id" = $1 ORDER BY "tag" ASC;`

	rows, err := t.Query(ctx, query, listingId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}

	return tags, nil
}

// GetByListings returns tags for multiple listings in a single query, keyed by listing ID.
func (t *GalleryListingTagsTable) GetByListings(ctx context.Context, listingIds []int) (map[int][]string, error) {
	if len(listingIds) == 0 {
		return make(map[int][]string), nil
	}

	query := `SELECT "listing_id", "tag" FROM gallery_listing_tags WHERE "listing_id" = ANY($1) ORDER BY "listing_id", "tag" ASC;`

	rows, err := t.Query(ctx, query, listingIds)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int][]string)
	for rows.Next() {
		var listingId int
		var tag string
		if err := rows.Scan(&listingId, &tag); err != nil {
			return nil, err
		}
		result[listingId] = append(result[listingId], tag)
	}

	return result, nil
}
