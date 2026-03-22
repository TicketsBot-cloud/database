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

type PolarEntitlements struct {
	*pgxpool.Pool
}

func newPolarEntitlements(db *pgxpool.Pool) *PolarEntitlements {
	return &PolarEntitlements{
		db,
	}
}

type PolarEntitlement struct {
	PolarSubscriptionId string    `json:"polar_subscription_id"`
	EntitlementId       uuid.UUID `json:"entitlement_id"`
	UserId              uint64    `json:"user_id"`
	PolarProductId      string    `json:"polar_product_id"`
	Status              string    `json:"status"`
}

type PolarEntitlementWithDetails struct {
	PolarEntitlement
	SkuId                uuid.UUID  `json:"sku_id"`
	SkuLabel             string     `json:"sku_label"`
	Tier                 string     `json:"tier"`
	ExpiresAt            *time.Time `json:"expires_at"`
	GuildId              *uint64    `json:"guild_id"`
	PermittedServerCount *int       `json:"permitted_server_count,omitempty"`
}

var (
	//go:embed sql/polar_entitlements/schema.sql
	polarEntitlementsSchema string

	//go:embed sql/polar_entitlements/insert.sql
	polarEntitlementsInsert string

	//go:embed sql/polar_entitlements/get_by_subscription_id.sql
	polarEntitlementsGetBySubscriptionId string

	//go:embed sql/polar_entitlements/list_by_user.sql
	polarEntitlementsListByUser string

	//go:embed sql/polar_entitlements/delete_by_subscription_id.sql
	polarEntitlementsDeleteBySubscriptionId string

	//go:embed sql/polar_entitlements/update_status.sql
	polarEntitlementsUpdateStatus string
)

func (e PolarEntitlements) Schema() string {
	return polarEntitlementsSchema
}

func (e *PolarEntitlements) Insert(ctx context.Context, tx pgx.Tx, polarSubscriptionId string, entitlementId uuid.UUID, userId uint64, polarProductId string, status string) error {
	_, err := tx.Exec(ctx, polarEntitlementsInsert, polarSubscriptionId, entitlementId, userId, polarProductId, status)
	return err
}

func (e *PolarEntitlements) GetBySubscriptionId(ctx context.Context, tx pgx.Tx, polarSubscriptionId string) (*PolarEntitlement, error) {
	var ent PolarEntitlement
	err := tx.QueryRow(ctx, polarEntitlementsGetBySubscriptionId, polarSubscriptionId).Scan(
		&ent.PolarSubscriptionId,
		&ent.EntitlementId,
		&ent.UserId,
		&ent.PolarProductId,
		&ent.Status,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &ent, nil
}

func (e *PolarEntitlements) ListByUser(ctx context.Context, tx pgx.Tx, userId uint64) ([]PolarEntitlementWithDetails, error) {
	rows, err := tx.Query(ctx, polarEntitlementsListByUser, userId)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var entitlements []PolarEntitlementWithDetails
	for rows.Next() {
		var ent PolarEntitlementWithDetails
		if err := rows.Scan(
			&ent.PolarSubscriptionId,
			&ent.EntitlementId,
			&ent.UserId,
			&ent.PolarProductId,
			&ent.Status,
			&ent.SkuId,
			&ent.SkuLabel,
			&ent.Tier,
			&ent.ExpiresAt,
			&ent.GuildId,
			&ent.PermittedServerCount,
		); err != nil {
			return nil, err
		}

		entitlements = append(entitlements, ent)
	}

	return entitlements, nil
}

func (e *PolarEntitlements) DeleteBySubscriptionId(ctx context.Context, tx pgx.Tx, polarSubscriptionId string) error {
	_, err := tx.Exec(ctx, polarEntitlementsDeleteBySubscriptionId, polarSubscriptionId)
	return err
}

func (e *PolarEntitlements) UpdateStatus(ctx context.Context, tx pgx.Tx, polarSubscriptionId string, status string) error {
	_, err := tx.Exec(ctx, polarEntitlementsUpdateStatus, polarSubscriptionId, status)
	return err
}
