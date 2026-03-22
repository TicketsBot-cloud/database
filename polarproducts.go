package database

import (
	"context"
	_ "embed"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type PolarProducts struct {
	*pgxpool.Pool
}

func newPolarProducts(db *pgxpool.Pool) *PolarProducts {
	return &PolarProducts{db}
}

type PolarProduct struct {
	Id               uuid.UUID `json:"id"`
	PolarProductId   string    `json:"polar_product_id"`
	SkuId            uuid.UUID `json:"sku_id"`
	Name             string    `json:"name"`
	Description      string    `json:"description"`
	Interval         string    `json:"interval"`
	PriceGbp         int       `json:"price_gbp"`
	Features         []string  `json:"features"`
	Highlighted      bool      `json:"highlighted"`
	SortOrder        int       `json:"sort_order"`
	Tier             string    `json:"tier"`
	ServersPermitted *int      `json:"servers_permitted,omitempty"`
}

var (
	//go:embed sql/polar_products/schema.sql
	polarProductsSchema string

	//go:embed sql/polar_products/list_all.sql
	polarProductsListAll string

	//go:embed sql/polar_products/get_by_polar_product_id.sql
	polarProductsGetByPolarProductId string

	//go:embed sql/polar_products/insert.sql
	polarProductsInsert string

	//go:embed sql/polar_products/update.sql
	polarProductsUpdate string

	//go:embed sql/polar_products/delete.sql
	polarProductsDelete string
)

func (t PolarProducts) Schema() string {
	return polarProductsSchema
}

func scanPolarProduct(row pgx.Row) (PolarProduct, error) {
	var p PolarProduct
	var features pgtype.TextArray

	err := row.Scan(
		&p.Id,
		&p.PolarProductId,
		&p.SkuId,
		&p.Name,
		&p.Description,
		&p.Interval,
		&p.PriceGbp,
		&features,
		&p.Highlighted,
		&p.SortOrder,
		&p.Tier,
		&p.ServersPermitted,
	)
	if err != nil {
		return p, err
	}

	p.Features = make([]string, 0)
	if features.Status == pgtype.Present {
		for _, el := range features.Elements {
			if el.Status == pgtype.Present {
				p.Features = append(p.Features, el.String)
			}
		}
	}

	return p, nil
}

func (t *PolarProducts) ListAll(ctx context.Context) ([]PolarProduct, error) {
	rows, err := t.Pool.Query(ctx, polarProductsListAll)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []PolarProduct
	for rows.Next() {
		p, err := scanPolarProduct(rows)
		if err != nil {
			return nil, err
		}
		products = append(products, p)
	}

	if products == nil {
		products = make([]PolarProduct, 0)
	}

	return products, nil
}

func (t *PolarProducts) GetByPolarProductId(ctx context.Context, polarProductId string) (*PolarProduct, error) {
	p, err := scanPolarProduct(t.Pool.QueryRow(ctx, polarProductsGetByPolarProductId, polarProductId))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (t *PolarProducts) Create(ctx context.Context, tx pgx.Tx, product PolarProduct) (PolarProduct, error) {
	featuresArray := toTextArray(product.Features)
	return scanPolarProduct(tx.QueryRow(ctx, polarProductsInsert,
		product.PolarProductId,
		product.SkuId,
		product.Name,
		product.Description,
		product.Interval,
		product.PriceGbp,
		featuresArray,
		product.Highlighted,
		product.SortOrder,
		product.Tier,
		product.ServersPermitted,
	))
}

func (t *PolarProducts) Update(ctx context.Context, tx pgx.Tx, product PolarProduct) error {
	featuresArray := toTextArray(product.Features)
	_, err := tx.Exec(ctx, polarProductsUpdate,
		product.Id,
		product.PolarProductId,
		product.SkuId,
		product.Name,
		product.Description,
		product.Interval,
		product.PriceGbp,
		featuresArray,
		product.Highlighted,
		product.SortOrder,
		product.Tier,
		product.ServersPermitted,
	)
	return err
}

func (t *PolarProducts) Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	_, err := tx.Exec(ctx, polarProductsDelete, id)
	return err
}

func toTextArray(strs []string) pgtype.TextArray {
	if len(strs) == 0 {
		return pgtype.TextArray{
			Status: pgtype.Present,
		}
	}

	elements := make([]pgtype.Text, len(strs))
	for i, s := range strs {
		elements[i] = pgtype.Text{String: s, Status: pgtype.Present}
	}

	return pgtype.TextArray{
		Elements:   elements,
		Dimensions: []pgtype.ArrayDimension{{Length: int32(len(strs)), LowerBound: 1}},
		Status:     pgtype.Present,
	}
}
