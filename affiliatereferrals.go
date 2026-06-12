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

type AffiliateReferrals struct {
	*pgxpool.Pool
}

type AffiliateReferral struct {
	Id                  uuid.UUID  `json:"id"`
	AffiliateCodeId     uuid.UUID  `json:"affiliate_code_id"`
	AffiliateUserId     uint64     `json:"affiliate_user_id"`
	ReferredUserId      uint64     `json:"referred_user_id"`
	PolarSubscriptionId string     `json:"polar_subscription_id"`
	ReferredTier        string     `json:"referred_tier"`
	ReferredSkuId       uuid.UUID  `json:"referred_sku_id"`
	PurchasedDays       int        `json:"purchased_days"`
	CreditDays          int        `json:"credit_days"`
	Status              string     `json:"status"`
	CreatedAt           time.Time  `json:"created_at"`
	RedeemableAt        time.Time  `json:"redeemable_at"`
	RedeemedAt          *time.Time `json:"redeemed_at"`
	EntitlementId       *uuid.UUID `json:"entitlement_id"`
	VoidedAt            *time.Time `json:"voided_at"`
	VoidedReason        *string    `json:"voided_reason"`
}

var (
	//go:embed sql/affiliate_referrals/schema.sql
	affiliateReferralsSchema string

	//go:embed sql/affiliate_referrals/create.sql
	affiliateReferralsCreate string

	//go:embed sql/affiliate_referrals/get_by_id.sql
	affiliateReferralsGetById string

	//go:embed sql/affiliate_referrals/get_by_subscription_id.sql
	affiliateReferralsGetBySubscriptionId string

	//go:embed sql/affiliate_referrals/list_by_affiliate_user_id.sql
	affiliateReferralsListByAffiliateUserId string

	//go:embed sql/affiliate_referrals/count_by_affiliate_user_id.sql
	affiliateReferralsCountByAffiliateUserId string

	//go:embed sql/affiliate_referrals/list_redeemable.sql
	affiliateReferralsListRedeemable string

	//go:embed sql/affiliate_referrals/update_status.sql
	affiliateReferralsUpdateStatus string

	//go:embed sql/affiliate_referrals/set_entitlement_id.sql
	affiliateReferralsSetEntitlementId string

	//go:embed sql/affiliate_referrals/void_by_subscription_id.sql
	affiliateReferralsVoidBySubscriptionId string

	//go:embed sql/affiliate_referrals/flag_by_subscription_id.sql
	affiliateReferralsFlagBySubscriptionId string

	//go:embed sql/affiliate_referrals/sum_redeemable_days.sql
	affiliateReferralsSumRedeemableDays string

	//go:embed sql/affiliate_referrals/count_pending.sql
	affiliateReferralsCountPending string

	//go:embed sql/affiliate_referrals/sum_redeemed_days.sql
	affiliateReferralsSumRedeemedDays string

	//go:embed sql/affiliate_referrals/list_flagged.sql
	affiliateReferralsListFlagged string

	//go:embed sql/affiliate_referrals/count_flagged.sql
	affiliateReferralsCountFlagged string
)

func newAffiliateReferrals(db *pgxpool.Pool) *AffiliateReferrals {
	return &AffiliateReferrals{db}
}

func (AffiliateReferrals) Schema() string {
	return affiliateReferralsSchema
}

func (t *AffiliateReferrals) scanRow(row pgx.Row) (*AffiliateReferral, error) {
	var r AffiliateReferral
	if err := row.Scan(
		&r.Id, &r.AffiliateCodeId, &r.AffiliateUserId, &r.ReferredUserId,
		&r.PolarSubscriptionId, &r.ReferredTier, &r.ReferredSkuId,
		&r.PurchasedDays, &r.CreditDays, &r.Status,
		&r.CreatedAt, &r.RedeemableAt, &r.RedeemedAt,
		&r.EntitlementId, &r.VoidedAt, &r.VoidedReason,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &r, nil
}

func (t *AffiliateReferrals) scanRows(rows pgx.Rows) ([]AffiliateReferral, error) {
	defer rows.Close()

	var referrals []AffiliateReferral
	for rows.Next() {
		var r AffiliateReferral
		if err := rows.Scan(
			&r.Id, &r.AffiliateCodeId, &r.AffiliateUserId, &r.ReferredUserId,
			&r.PolarSubscriptionId, &r.ReferredTier, &r.ReferredSkuId,
			&r.PurchasedDays, &r.CreditDays, &r.Status,
			&r.CreatedAt, &r.RedeemableAt, &r.RedeemedAt,
			&r.EntitlementId, &r.VoidedAt, &r.VoidedReason,
		); err != nil {
			return nil, err
		}
		referrals = append(referrals, r)
	}
	return referrals, nil
}

func (t *AffiliateReferrals) Create(
	ctx context.Context,
	affiliateCodeId uuid.UUID,
	affiliateUserId, referredUserId uint64,
	polarSubscriptionId, referredTier string,
	referredSkuId uuid.UUID,
	purchasedDays, creditDays int,
	status string,
	redeemableAt time.Time,
) (AffiliateReferral, error) {
	var id uuid.UUID
	var createdAt time.Time
	if err := t.QueryRow(ctx, affiliateReferralsCreate,
		affiliateCodeId, affiliateUserId, referredUserId,
		polarSubscriptionId, referredTier, referredSkuId,
		purchasedDays, creditDays, status, redeemableAt,
	).Scan(&id, &createdAt); err != nil {
		return AffiliateReferral{}, err
	}

	return AffiliateReferral{
		Id:                  id,
		AffiliateCodeId:     affiliateCodeId,
		AffiliateUserId:     affiliateUserId,
		ReferredUserId:      referredUserId,
		PolarSubscriptionId: polarSubscriptionId,
		ReferredTier:        referredTier,
		ReferredSkuId:       referredSkuId,
		PurchasedDays:       purchasedDays,
		CreditDays:          creditDays,
		Status:              status,
		CreatedAt:           createdAt,
		RedeemableAt:        redeemableAt,
	}, nil
}

func (t *AffiliateReferrals) GetById(ctx context.Context, id uuid.UUID) (*AffiliateReferral, error) {
	return t.scanRow(t.QueryRow(ctx, affiliateReferralsGetById, id))
}

func (t *AffiliateReferrals) GetBySubscriptionId(ctx context.Context, polarSubscriptionId string) (*AffiliateReferral, error) {
	return t.scanRow(t.QueryRow(ctx, affiliateReferralsGetBySubscriptionId, polarSubscriptionId))
}

func (t *AffiliateReferrals) ListByAffiliateUserId(ctx context.Context, affiliateUserId uint64, limit, offset int) ([]AffiliateReferral, error) {
	rows, err := t.Query(ctx, affiliateReferralsListByAffiliateUserId, affiliateUserId, limit, offset)
	if err != nil {
		return nil, err
	}
	return t.scanRows(rows)
}

func (t *AffiliateReferrals) CountByAffiliateUserId(ctx context.Context, affiliateUserId uint64) (int, error) {
	var count int
	if err := t.QueryRow(ctx, affiliateReferralsCountByAffiliateUserId, affiliateUserId).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (t *AffiliateReferrals) ListRedeemable(ctx context.Context, tx pgx.Tx, affiliateUserId uint64) ([]AffiliateReferral, error) {
	rows, err := tx.Query(ctx, affiliateReferralsListRedeemable, affiliateUserId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var referrals []AffiliateReferral
	for rows.Next() {
		var r AffiliateReferral
		if err := rows.Scan(
			&r.Id, &r.AffiliateCodeId, &r.AffiliateUserId, &r.ReferredUserId,
			&r.PolarSubscriptionId, &r.ReferredTier, &r.ReferredSkuId,
			&r.PurchasedDays, &r.CreditDays, &r.Status,
			&r.CreatedAt, &r.RedeemableAt, &r.RedeemedAt,
			&r.EntitlementId, &r.VoidedAt, &r.VoidedReason,
		); err != nil {
			return nil, err
		}
		referrals = append(referrals, r)
	}
	return referrals, nil
}

func (t *AffiliateReferrals) UpdateStatus(ctx context.Context, tx pgx.Tx, id uuid.UUID, status string) error {
	_, err := tx.Exec(ctx, affiliateReferralsUpdateStatus, id, status)
	return err
}

func (t *AffiliateReferrals) SetEntitlementId(ctx context.Context, tx pgx.Tx, id uuid.UUID, entitlementId uuid.UUID) error {
	_, err := tx.Exec(ctx, affiliateReferralsSetEntitlementId, id, entitlementId)
	return err
}

func (t *AffiliateReferrals) VoidBySubscriptionId(ctx context.Context, polarSubscriptionId, reason string) error {
	_, err := t.Exec(ctx, affiliateReferralsVoidBySubscriptionId, polarSubscriptionId, reason)
	return err
}

func (t *AffiliateReferrals) FlagBySubscriptionId(ctx context.Context, polarSubscriptionId, reason string) error {
	_, err := t.Exec(ctx, affiliateReferralsFlagBySubscriptionId, polarSubscriptionId, reason)
	return err
}

func (t *AffiliateReferrals) SumRedeemedByUser(ctx context.Context, affiliateUserId uint64) (int, error) {
	var sum int
	if err := t.QueryRow(ctx, affiliateReferralsSumRedeemedDays, affiliateUserId).Scan(&sum); err != nil {
		return 0, err
	}
	return sum, nil
}

func (t *AffiliateReferrals) SumRedeemedByUserTx(ctx context.Context, tx pgx.Tx, affiliateUserId uint64) (int, error) {
	var sum int
	if err := tx.QueryRow(ctx, affiliateReferralsSumRedeemedDays, affiliateUserId).Scan(&sum); err != nil {
		return 0, err
	}
	return sum, nil
}

func (t *AffiliateReferrals) ListFlagged(ctx context.Context, limit, offset int) ([]AffiliateReferral, error) {
	rows, err := t.Query(ctx, affiliateReferralsListFlagged, limit, offset)
	if err != nil {
		return nil, err
	}
	return t.scanRows(rows)
}

func (t *AffiliateReferrals) CountFlagged(ctx context.Context) (int, error) {
	var count int
	if err := t.QueryRow(ctx, affiliateReferralsCountFlagged).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (t *AffiliateReferrals) SumRedeemableByUser(ctx context.Context, affiliateUserId uint64) (int, error) {
	var sum int
	if err := t.QueryRow(ctx, affiliateReferralsSumRedeemableDays, affiliateUserId).Scan(&sum); err != nil {
		return 0, err
	}
	return sum, nil
}

func (t *AffiliateReferrals) CountPending(ctx context.Context, affiliateUserId uint64) (int, error) {
	var count int
	if err := t.QueryRow(ctx, affiliateReferralsCountPending, affiliateUserId).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}
