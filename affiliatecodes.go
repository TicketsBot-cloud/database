package database

import (
	"context"
	_ "embed"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type AffiliateCodes struct {
	*pgxpool.Pool
}

type AffiliateCode struct {
	Id                  uuid.UUID  `json:"id"`
	UserId              uint64     `json:"user_id"`
	Code                string     `json:"code"`
	PolarDiscountId     *string    `json:"polar_discount_id"`
	Status              string     `json:"status"`
	DiscountBasisPoints int        `json:"discount_basis_points"`
	CreditPercentage    *int       `json:"credit_percentage"`
	CreatedAt           time.Time  `json:"created_at"`
	ApprovedAt          *time.Time `json:"approved_at"`
	ApprovedBy          *uint64    `json:"approved_by"`
	RevokedAt           *time.Time `json:"revoked_at"`
}

var (
	//go:embed sql/affiliate_codes/schema.sql
	affiliateCodesSchema string

	//go:embed sql/affiliate_codes/create.sql
	affiliateCodesCreate string

	//go:embed sql/affiliate_codes/get_by_id.sql
	affiliateCodesGetById string

	//go:embed sql/affiliate_codes/get_by_code.sql
	affiliateCodesGetByCode string

	//go:embed sql/affiliate_codes/get_by_user_id.sql
	affiliateCodesGetByUserId string

	//go:embed sql/affiliate_codes/get_by_polar_discount_id.sql
	affiliateCodesGetByPolarDiscountId string

	//go:embed sql/affiliate_codes/list_all.sql
	affiliateCodesListAll string

	//go:embed sql/affiliate_codes/count_all.sql
	affiliateCodesCountAll string

	//go:embed sql/affiliate_codes/update_status.sql
	affiliateCodesUpdateStatus string

	//go:embed sql/affiliate_codes/set_polar_discount_id.sql
	affiliateCodesSetPolarDiscountId string

	//go:embed sql/affiliate_codes/update_rates.sql
	affiliateCodesUpdateRates string

	//go:embed sql/affiliate_codes/update_code.sql
	affiliateCodesUpdateCode string
)

func newAffiliateCodes(db *pgxpool.Pool) *AffiliateCodes {
	return &AffiliateCodes{db}
}

func (AffiliateCodes) Schema() string {
	return affiliateCodesSchema
}

func (t *AffiliateCodes) Create(ctx context.Context, userId uint64, code, status string, discountBasisPoints int, creditPercentage *int) (AffiliateCode, error) {
	var id uuid.UUID
	var createdAt time.Time
	if err := t.QueryRow(ctx, affiliateCodesCreate, userId, code, status, discountBasisPoints, creditPercentage).Scan(&id, &createdAt); err != nil {
		return AffiliateCode{}, err
	}

	return AffiliateCode{
		Id:                  id,
		UserId:              userId,
		Code:                code,
		Status:              status,
		DiscountBasisPoints: discountBasisPoints,
		CreditPercentage:    creditPercentage,
		CreatedAt:           createdAt,
	}, nil
}

func (t *AffiliateCodes) scanRow(row pgx.Row) (*AffiliateCode, error) {
	var ac AffiliateCode
	if err := row.Scan(
		&ac.Id, &ac.UserId, &ac.Code, &ac.PolarDiscountId, &ac.Status,
		&ac.DiscountBasisPoints, &ac.CreditPercentage,
		&ac.CreatedAt, &ac.ApprovedAt, &ac.ApprovedBy, &ac.RevokedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &ac, nil
}

func (t *AffiliateCodes) GetById(ctx context.Context, id uuid.UUID) (*AffiliateCode, error) {
	return t.scanRow(t.QueryRow(ctx, affiliateCodesGetById, id))
}

func (t *AffiliateCodes) GetByCode(ctx context.Context, code string) (*AffiliateCode, error) {
	return t.scanRow(t.QueryRow(ctx, affiliateCodesGetByCode, code))
}

func (t *AffiliateCodes) GetByUserId(ctx context.Context, userId uint64) (*AffiliateCode, error) {
	return t.scanRow(t.QueryRow(ctx, affiliateCodesGetByUserId, userId))
}

func (t *AffiliateCodes) GetByPolarDiscountId(ctx context.Context, polarDiscountId string) (*AffiliateCode, error) {
	return t.scanRow(t.QueryRow(ctx, affiliateCodesGetByPolarDiscountId, polarDiscountId))
}

func (t *AffiliateCodes) ListAll(ctx context.Context, status *string, limit, offset int) ([]AffiliateCode, error) {
	rows, err := t.Query(ctx, affiliateCodesListAll, status, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var codes []AffiliateCode
	for rows.Next() {
		var ac AffiliateCode
		if err := rows.Scan(
			&ac.Id, &ac.UserId, &ac.Code, &ac.PolarDiscountId, &ac.Status,
			&ac.DiscountBasisPoints, &ac.CreditPercentage,
			&ac.CreatedAt, &ac.ApprovedAt, &ac.ApprovedBy, &ac.RevokedAt,
		); err != nil {
			return nil, err
		}
		codes = append(codes, ac)
	}
	return codes, nil
}

func (t *AffiliateCodes) CountAll(ctx context.Context, status *string) (int, error) {
	var count int
	if err := t.QueryRow(ctx, affiliateCodesCountAll, status).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (t *AffiliateCodes) UpdateStatus(ctx context.Context, id uuid.UUID, status string, approvedBy *uint64) error {
	_, err := t.Exec(ctx, affiliateCodesUpdateStatus, id, status, approvedBy)
	return err
}

func (t *AffiliateCodes) SetPolarDiscountId(ctx context.Context, id uuid.UUID, polarDiscountId string) error {
	_, err := t.Exec(ctx, affiliateCodesSetPolarDiscountId, id, polarDiscountId)
	return err
}

func (t *AffiliateCodes) UpdateRates(ctx context.Context, id uuid.UUID, discountBasisPoints int, creditPercentage *int) error {
	_, err := t.Exec(ctx, affiliateCodesUpdateRates, id, discountBasisPoints, creditPercentage)
	return err
}

func (t *AffiliateCodes) UpdateCode(ctx context.Context, id uuid.UUID, code string) error {
	_, err := t.Exec(ctx, affiliateCodesUpdateCode, id, code)
	return err
}
