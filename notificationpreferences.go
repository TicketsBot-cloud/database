package database

import (
	"context"
	_ "embed"
	"errors"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type NotificationPreferencesTable struct {
	*pgxpool.Pool
}

type NotificationPreference struct {
	UserId    uint64 `json:"user_id"`
	Category  string `json:"category"`
	DiscordDm bool   `json:"discord_dm"`
	Email     bool   `json:"email"`
	InApp     bool   `json:"in_app"`
}

var (
	//go:embed sql/notification_preferences/schema.sql
	notificationPreferencesSchema string

	//go:embed sql/notification_preferences/get_by_user_id.sql
	notificationPreferencesGetByUserId string

	//go:embed sql/notification_preferences/get_by_user_id_and_category.sql
	notificationPreferencesGetByUserIdAndCategory string

	//go:embed sql/notification_preferences/upsert.sql
	notificationPreferencesUpsert string
)

func newNotificationPreferencesTable(pool *pgxpool.Pool) *NotificationPreferencesTable {
	return &NotificationPreferencesTable{pool}
}

func (NotificationPreferencesTable) Schema() string {
	return notificationPreferencesSchema
}

func (t *NotificationPreferencesTable) GetByUserId(ctx context.Context, userId uint64) ([]NotificationPreference, error) {
	rows, err := t.Query(ctx, notificationPreferencesGetByUserId, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prefs []NotificationPreference
	for rows.Next() {
		var p NotificationPreference
		if err := rows.Scan(&p.UserId, &p.Category, &p.DiscordDm, &p.Email, &p.InApp); err != nil {
			return nil, err
		}
		prefs = append(prefs, p)
	}
	return prefs, nil
}

func (t *NotificationPreferencesTable) GetByUserIdAndCategory(ctx context.Context, userId uint64, category string) (*NotificationPreference, error) {
	var p NotificationPreference
	if err := t.QueryRow(ctx, notificationPreferencesGetByUserIdAndCategory, userId, category).Scan(
		&p.UserId, &p.Category, &p.DiscordDm, &p.Email, &p.InApp,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (t *NotificationPreferencesTable) Upsert(ctx context.Context, userId uint64, category string, discordDm, email, inApp bool) error {
	_, err := t.Exec(ctx, notificationPreferencesUpsert, userId, category, discordDm, email, inApp)
	return err
}
