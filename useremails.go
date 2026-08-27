package database

import (
	"context"
	_ "embed"
	"errors"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type UserEmails struct {
	*pgxpool.Pool
}

type UserEmail struct {
	UserId    uint64    `json:"user_id"`
	Email     string    `json:"email"`
	Verified  bool      `json:"verified"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

var (
	//go:embed sql/user_emails/schema.sql
	userEmailsSchema string

	//go:embed sql/user_emails/upsert.sql
	userEmailsUpsert string

	//go:embed sql/user_emails/get_by_user_id.sql
	userEmailsGetByUserId string

	//go:embed sql/user_emails/set_verified.sql
	userEmailsSetVerified string

	//go:embed sql/user_emails/delete.sql
	userEmailsDelete string
)

func newUserEmails(db *pgxpool.Pool) *UserEmails {
	return &UserEmails{db}
}

func (UserEmails) Schema() string {
	return userEmailsSchema
}

func (t *UserEmails) Upsert(ctx context.Context, userId uint64, email string) error {
	_, err := t.Exec(ctx, userEmailsUpsert, userId, email)
	return err
}

func (t *UserEmails) GetByUserId(ctx context.Context, userId uint64) (*UserEmail, error) {
	var ue UserEmail
	if err := t.QueryRow(ctx, userEmailsGetByUserId, userId).Scan(
		&ue.UserId, &ue.Email, &ue.Verified, &ue.CreatedAt, &ue.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &ue, nil
}

func (t *UserEmails) SetVerified(ctx context.Context, userId uint64, verified bool) error {
	_, err := t.Exec(ctx, userEmailsSetVerified, userId, verified)
	return err
}

func (t *UserEmails) Delete(ctx context.Context, userId uint64) error {
	_, err := t.Exec(ctx, userEmailsDelete, userId)
	return err
}
