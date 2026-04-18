package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type GalleryListingStatus string

const (
	GalleryListingStatusPending  GalleryListingStatus = "pending"
	GalleryListingStatusApproved GalleryListingStatus = "approved"
	GalleryListingStatusRejected GalleryListingStatus = "rejected"
)

type GalleryListing struct {
	Id              int                  `json:"id"`
	SubmitterUserId uint64               `json:"submitter_user_id,string"`
	SourceGuildId   uint64               `json:"source_guild_id,string"`
	Name            string               `json:"name"`
	Description     string               `json:"description"`
	Category        string               `json:"category"`
	Status          GalleryListingStatus `json:"status"`
	ReviewNote      *string              `json:"review_note"`
	ReviewedBy      *uint64              `json:"reviewed_by,string"`
	ReviewedAt      *time.Time           `json:"reviewed_at"`
	ImportCount     int                  `json:"import_count"`
	Featured        bool                 `json:"featured"`
	// Visual design fields only — no operational settings
	Title          string  `json:"title"`
	Content        string  `json:"content"`
	Colour         int32   `json:"colour"`
	ImageUrl       *string `json:"image_url"`
	ThumbnailUrl   *string `json:"thumbnail_url"`
	ButtonStyle    *int16  `json:"button_style"`
	ButtonLabel    string  `json:"button_label"`
	EmojiName      *string `json:"emoji_name"`
	WelcomeMessage []byte  `json:"welcome_message"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type GalleryListingsTable struct {
	*pgxpool.Pool
}

func newGalleryListingsTable(pool *pgxpool.Pool) *GalleryListingsTable {
	return &GalleryListingsTable{
		pool,
	}
}

func (t GalleryListingsTable) Schema() string {
	return `
CREATE TABLE IF NOT EXISTS gallery_listings (
	"id"                 SERIAL PRIMARY KEY,
	"submitter_user_id"  INT8 NOT NULL,
	"source_guild_id"    INT8 NOT NULL,
	"name"               VARCHAR(100) NOT NULL,
	"description"        TEXT NOT NULL,
	"category"           VARCHAR(50) NOT NULL,
	"status"             VARCHAR(20) NOT NULL DEFAULT 'pending',
	"review_note"        TEXT,
	"reviewed_by"        INT8,
	"reviewed_at"        TIMESTAMPTZ,
	"import_count"       INT NOT NULL DEFAULT 0,
	"featured"           BOOL NOT NULL DEFAULT false,
	"title"              VARCHAR(255) NOT NULL,
	"content"            TEXT NOT NULL,
	"colour"             INT4 NOT NULL,
	"image_url"          TEXT,
	"thumbnail_url"      TEXT,
	"button_style"       INT2 DEFAULT 1,
	"button_label"       VARCHAR(80) NOT NULL,
	"emoji_name"         VARCHAR(32),
	"welcome_message"    JSONB,
	"created_at"         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	"updated_at"         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS gallery_listings_status_idx ON gallery_listings("status");
CREATE INDEX IF NOT EXISTS gallery_listings_category_idx ON gallery_listings("category");
CREATE INDEX IF NOT EXISTS gallery_listings_submitter_user_id_idx ON gallery_listings("submitter_user_id");
CREATE INDEX IF NOT EXISTS gallery_listings_source_guild_id_idx ON gallery_listings("source_guild_id");
CREATE INDEX IF NOT EXISTS gallery_listings_featured_idx ON gallery_listings("featured") WHERE "featured" = true;
CREATE INDEX IF NOT EXISTS gallery_listings_import_count_idx ON gallery_listings("import_count" DESC);
CREATE INDEX IF NOT EXISTS gallery_listings_created_at_idx ON gallery_listings("created_at" DESC);
`
}

func (l *GalleryListing) fieldPtrs() []interface{} {
	return []interface{}{
		&l.Id,
		&l.SubmitterUserId,
		&l.SourceGuildId,
		&l.Name,
		&l.Description,
		&l.Category,
		&l.Status,
		&l.ReviewNote,
		&l.ReviewedBy,
		&l.ReviewedAt,
		&l.ImportCount,
		&l.Featured,
		&l.Title,
		&l.Content,
		&l.Colour,
		&l.ImageUrl,
		&l.ThumbnailUrl,
		&l.ButtonStyle,
		&l.ButtonLabel,
		&l.EmojiName,
		&l.WelcomeMessage,
		&l.CreatedAt,
		&l.UpdatedAt,
	}
}

const galleryListingSelectColumns = `
	"id",
	"submitter_user_id",
	"source_guild_id",
	"name",
	"description",
	"category",
	"status",
	"review_note",
	"reviewed_by",
	"reviewed_at",
	"import_count",
	"featured",
	"title",
	"content",
	"colour",
	"image_url",
	"thumbnail_url",
	"button_style",
	"button_label",
	"emoji_name",
	"welcome_message",
	"created_at",
	"updated_at"
`

func (t *GalleryListingsTable) Create(ctx context.Context, listing GalleryListing) (int, error) {
	query := `
INSERT INTO gallery_listings (
	"submitter_user_id",
	"source_guild_id",
	"name",
	"description",
	"category",
	"status",
	"title",
	"content",
	"colour",
	"image_url",
	"thumbnail_url",
	"button_style",
	"button_label",
	"emoji_name",
	"welcome_message"
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
RETURNING "id";`

	var id int
	err := t.QueryRow(ctx, query,
		listing.SubmitterUserId,
		listing.SourceGuildId,
		listing.Name,
		listing.Description,
		listing.Category,
		listing.Status,
		listing.Title,
		listing.Content,
		listing.Colour,
		listing.ImageUrl,
		listing.ThumbnailUrl,
		listing.ButtonStyle,
		listing.ButtonLabel,
		listing.EmojiName,
		listing.WelcomeMessage,
	).Scan(&id)

	return id, err
}

func (t *GalleryListingsTable) GetById(ctx context.Context, id int) (listing GalleryListing, ok bool, e error) {
	query := `SELECT ` + galleryListingSelectColumns + ` FROM gallery_listings WHERE "id" = $1;`

	switch err := t.QueryRow(ctx, query, id).Scan(listing.fieldPtrs()...); err {
	case nil:
		ok = true
	case pgx.ErrNoRows:
	default:
		e = err
	}

	return
}

// GetApproved returns paginated approved listings with optional filters.
// Sort must be "popular" (import_count DESC) or "recent" (created_at DESC).
// Search matches name or description using case-insensitive ILIKE.
func (t *GalleryListingsTable) GetApproved(ctx context.Context, category string, tag string, search string, sort string, page int, pageSize int) ([]GalleryListing, int, error) {
	var conditions []string
	var args []interface{}

	conditions = append(conditions, `"status" = 'approved'`)

	if category != "" {
		args = append(args, category)
		conditions = append(conditions, fmt.Sprintf(`"category" = $%d`, len(args)))
	}

	if tag != "" {
		args = append(args, tag)
		conditions = append(conditions, fmt.Sprintf(`"id" IN (SELECT "listing_id" FROM gallery_listing_tags WHERE "tag" = $%d)`, len(args)))
	}

	if search != "" {
		args = append(args, "%"+search+"%")
		argIdx := len(args)
		conditions = append(conditions, fmt.Sprintf(`("name" ILIKE $%d OR "description" ILIKE $%d)`, argIdx, argIdx))
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total matching records
	countQuery := "SELECT COUNT(*) FROM gallery_listings" + whereClause
	var total int
	if err := t.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Determine sort order
	orderClause := ` ORDER BY "created_at" DESC`
	if sort == "popular" {
		orderClause = ` ORDER BY "import_count" DESC, "created_at" DESC`
	}

	// Pagination
	offset := (page - 1) * pageSize
	args = append(args, pageSize)
	limitClause := fmt.Sprintf(` LIMIT $%d`, len(args))
	args = append(args, offset)
	offsetClause := fmt.Sprintf(` OFFSET $%d`, len(args))

	query := "SELECT " + galleryListingSelectColumns + " FROM gallery_listings" + whereClause + orderClause + limitClause + offsetClause

	rows, err := t.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var listings []GalleryListing
	for rows.Next() {
		var listing GalleryListing
		if err := rows.Scan(listing.fieldPtrs()...); err != nil {
			return nil, 0, err
		}
		listings = append(listings, listing)
	}

	return listings, total, nil
}

// GetFeatured returns all approved featured listings.
func (t *GalleryListingsTable) GetFeatured(ctx context.Context) ([]GalleryListing, error) {
	query := `SELECT ` + galleryListingSelectColumns + ` FROM gallery_listings WHERE "status" = 'approved' AND "featured" = true ORDER BY "import_count" DESC;`

	rows, err := t.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var listings []GalleryListing
	for rows.Next() {
		var listing GalleryListing
		if err := rows.Scan(listing.fieldPtrs()...); err != nil {
			return nil, err
		}
		listings = append(listings, listing)
	}

	return listings, nil
}

// GetBySubmitter returns all listings submitted by a given user.
func (t *GalleryListingsTable) GetBySubmitter(ctx context.Context, userId uint64) ([]GalleryListing, error) {
	query := `SELECT ` + galleryListingSelectColumns + ` FROM gallery_listings WHERE "submitter_user_id" = $1 ORDER BY "created_at" DESC;`

	rows, err := t.Query(ctx, query, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var listings []GalleryListing
	for rows.Next() {
		var listing GalleryListing
		if err := rows.Scan(listing.fieldPtrs()...); err != nil {
			return nil, err
		}
		listings = append(listings, listing)
	}

	return listings, nil
}

// GetByGuild returns all listings from a given guild.
func (t *GalleryListingsTable) GetByGuild(ctx context.Context, guildId uint64) ([]GalleryListing, error) {
	query := `SELECT ` + galleryListingSelectColumns + ` FROM gallery_listings WHERE "source_guild_id" = $1 ORDER BY "created_at" DESC;`

	rows, err := t.Query(ctx, query, guildId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var listings []GalleryListing
	for rows.Next() {
		var listing GalleryListing
		if err := rows.Scan(listing.fieldPtrs()...); err != nil {
			return nil, err
		}
		listings = append(listings, listing)
	}

	return listings, nil
}

// GetPending returns all listings with pending status for admin review.
func (t *GalleryListingsTable) GetPending(ctx context.Context) ([]GalleryListing, error) {
	return t.GetByStatus(ctx, GalleryListingStatusPending)
}

// GetByStatus returns all listings filtered by status.
func (t *GalleryListingsTable) GetByStatus(ctx context.Context, status GalleryListingStatus) ([]GalleryListing, error) {
	query := `SELECT ` + galleryListingSelectColumns + ` FROM gallery_listings WHERE "status" = $1 ORDER BY "created_at" DESC;`

	rows, err := t.Query(ctx, query, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var listings []GalleryListing
	for rows.Next() {
		var listing GalleryListing
		if err := rows.Scan(listing.fieldPtrs()...); err != nil {
			return nil, err
		}
		listings = append(listings, listing)
	}

	return listings, nil
}

// UpdateStatus updates the listing status along with reviewer information.
func (t *GalleryListingsTable) UpdateStatus(ctx context.Context, id int, status GalleryListingStatus, reviewedBy uint64, reviewNote *string) error {
	query := `
UPDATE gallery_listings
SET "status" = $2, "reviewed_by" = $3, "reviewed_at" = NOW(), "review_note" = $4, "updated_at" = NOW()
WHERE "id" = $1;`

	_, err := t.Exec(ctx, query, id, status, reviewedBy, reviewNote)
	return err
}

// Update updates mutable fields on a gallery listing (category, featured).
func (t *GalleryListingsTable) Update(ctx context.Context, id int, category *string, featured *bool) error {
	var setClauses []string
	var args []interface{}
	args = append(args, id) // $1 is always the id

	if category != nil {
		args = append(args, *category)
		setClauses = append(setClauses, fmt.Sprintf(`"category" = $%d`, len(args)))
	}

	if featured != nil {
		args = append(args, *featured)
		setClauses = append(setClauses, fmt.Sprintf(`"featured" = $%d`, len(args)))
	}

	if len(setClauses) == 0 {
		return nil
	}

	setClauses = append(setClauses, `"updated_at" = NOW()`)

	query := `UPDATE gallery_listings SET ` + strings.Join(setClauses, ", ") + ` WHERE "id" = $1;`
	_, err := t.Exec(ctx, query, args...)
	return err
}

// IncrementImportCount atomically increments the import count for a listing.
func (t *GalleryListingsTable) IncrementImportCount(ctx context.Context, id int) error {
	query := `UPDATE gallery_listings SET "import_count" = "import_count" + 1, "updated_at" = NOW() WHERE "id" = $1;`
	_, err := t.Exec(ctx, query, id)
	return err
}

// UpdateMetadata updates the name, description, and category of a listing.
func (t *GalleryListingsTable) UpdateMetadata(ctx context.Context, id int, name string, description string, category string) error {
	query := `UPDATE gallery_listings SET "name" = $2, "description" = $3, "category" = $4, "updated_at" = NOW() WHERE "id" = $1;`
	_, err := t.Exec(ctx, query, id, name, description, category)
	return err
}

// Resubmit replaces a listing's panel snapshot and metadata, resetting it for review.
func (t *GalleryListingsTable) Resubmit(ctx context.Context, listing GalleryListing) error {
	query := `
UPDATE gallery_listings SET
	"name" = $2,
	"description" = $3,
	"category" = $4,
	"status" = $5,
	"review_note" = NULL,
	"reviewed_by" = NULL,
	"reviewed_at" = NULL,
	"title" = $6,
	"content" = $7,
	"colour" = $8,
	"image_url" = $9,
	"thumbnail_url" = $10,
	"button_style" = $11,
	"button_label" = $12,
	"emoji_name" = $13,
	"welcome_message" = $14,
	"updated_at" = NOW()
WHERE "id" = $1;`

	_, err := t.Exec(ctx, query,
		listing.Id,
		listing.Name,
		listing.Description,
		listing.Category,
		listing.Status,
		listing.Title,
		listing.Content,
		listing.Colour,
		listing.ImageUrl,
		listing.ThumbnailUrl,
		listing.ButtonStyle,
		listing.ButtonLabel,
		listing.EmojiName,
		listing.WelcomeMessage,
	)
	return err
}

// Delete removes a gallery listing by ID.
func (t *GalleryListingsTable) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM gallery_listings WHERE "id" = $1;`
	_, err := t.Exec(ctx, query, id)
	return err
}
