package database

import (
	"context"
	_ "embed"
	"errors"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type EmailVerificationCodes struct {
	*pgxpool.Pool
}

type EmailVerificationCode struct {
	UserId         uint64    `json:"user_id"`
	Code           string    `json:"code"`
	ExpiresAt      time.Time `json:"expires_at"`
	CreatedAt      time.Time `json:"created_at"`
	FailedAttempts int       `json:"failed_attempts"`
}

var (
	//go:embed sql/email_verification_codes/schema.sql
	emailVerificationCodesSchema string

	//go:embed sql/email_verification_codes/upsert.sql
	emailVerificationCodesUpsert string

	//go:embed sql/email_verification_codes/get_by_user_id.sql
	emailVerificationCodesGetByUserId string

	//go:embed sql/email_verification_codes/delete.sql
	emailVerificationCodesDelete string

	//go:embed sql/email_verification_codes/increment_attempts.sql
	emailVerificationCodesIncrementAttempts string
)

func newEmailVerificationCodes(db *pgxpool.Pool) *EmailVerificationCodes {
	return &EmailVerificationCodes{db}
}

func (EmailVerificationCodes) Schema() string {
	return emailVerificationCodesSchema
}

func (t *EmailVerificationCodes) Upsert(ctx context.Context, userId uint64, code string, expiresAt time.Time) error {
	_, err := t.Exec(ctx, emailVerificationCodesUpsert, userId, code, expiresAt)
	return err
}

func (t *EmailVerificationCodes) GetByUserId(ctx context.Context, userId uint64) (*EmailVerificationCode, error) {
	var vc EmailVerificationCode
	if err := t.QueryRow(ctx, emailVerificationCodesGetByUserId, userId).Scan(
		&vc.UserId, &vc.Code, &vc.ExpiresAt, &vc.CreatedAt, &vc.FailedAttempts,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &vc, nil
}

func (t *EmailVerificationCodes) Delete(ctx context.Context, userId uint64) error {
	_, err := t.Exec(ctx, emailVerificationCodesDelete, userId)
	return err
}

func (t *EmailVerificationCodes) IncrementAttempts(ctx context.Context, userId uint64) (int, error) {
	var count int
	if err := t.QueryRow(ctx, emailVerificationCodesIncrementAttempts, userId).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}
