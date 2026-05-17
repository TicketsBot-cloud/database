package database

import (
	"context"
	_ "embed"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

type NotificationsTable struct {
	*pgxpool.Pool
}

type Notification struct {
	Id        int64     `json:"id"`
	UserId    uint64    `json:"user_id"`
	Category  string    `json:"category"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Link      *string   `json:"link"`
	Read      bool      `json:"read"`
	CreatedAt time.Time `json:"created_at"`
}

var (
	//go:embed sql/notifications/schema.sql
	notificationsSchema string

	//go:embed sql/notifications/create.sql
	notificationsCreate string

	//go:embed sql/notifications/list_by_user_id.sql
	notificationsListByUserId string

	//go:embed sql/notifications/count_unread_by_user_id.sql
	notificationsCountUnreadByUserId string

	//go:embed sql/notifications/count_by_user_id.sql
	notificationsCountByUserId string

	//go:embed sql/notifications/mark_as_read.sql
	notificationsMarkAsRead string

	//go:embed sql/notifications/mark_all_as_read.sql
	notificationsMarkAllAsRead string
)

func newNotificationsTable(pool *pgxpool.Pool) *NotificationsTable {
	return &NotificationsTable{pool}
}

func (NotificationsTable) Schema() string {
	return notificationsSchema
}

func (t *NotificationsTable) Create(ctx context.Context, userId uint64, category, title, body string, link *string) (Notification, error) {
	if len(title) > 200 {
		title = title[:200]
	}
	if len(body) > 2000 {
		body = body[:2000]
	}

	var id int64
	var createdAt time.Time
	if err := t.QueryRow(ctx, notificationsCreate, userId, category, title, body, link).Scan(&id, &createdAt); err != nil {
		return Notification{}, err
	}

	return Notification{
		Id:        id,
		UserId:    userId,
		Category:  category,
		Title:     title,
		Body:      body,
		Link:      link,
		Read:      false,
		CreatedAt: createdAt,
	}, nil
}

func (t *NotificationsTable) ListByUserId(ctx context.Context, userId uint64, category *string, limit, offset int) ([]Notification, error) {
	rows, err := t.Query(ctx, notificationsListByUserId, userId, category, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []Notification
	for rows.Next() {
		var n Notification
		if err := rows.Scan(
			&n.Id, &n.UserId, &n.Category, &n.Title, &n.Body, &n.Link, &n.Read, &n.CreatedAt,
		); err != nil {
			return nil, err
		}
		notifications = append(notifications, n)
	}
	return notifications, nil
}

func (t *NotificationsTable) CountUnreadByUserId(ctx context.Context, userId uint64) (int, error) {
	var count int
	if err := t.QueryRow(ctx, notificationsCountUnreadByUserId, userId).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (t *NotificationsTable) CountByUserId(ctx context.Context, userId uint64, category *string) (int, error) {
	var count int
	if err := t.QueryRow(ctx, notificationsCountByUserId, userId, category).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (t *NotificationsTable) MarkAsRead(ctx context.Context, id int64, userId uint64) error {
	_, err := t.Exec(ctx, notificationsMarkAsRead, id, userId)
	return err
}

func (t *NotificationsTable) MarkAllAsRead(ctx context.Context, userId uint64) error {
	_, err := t.Exec(ctx, notificationsMarkAllAsRead, userId)
	return err
}
