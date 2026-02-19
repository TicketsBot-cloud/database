package database

import (
	"context"
	_ "embed"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type DmMessage struct {
	MessageId uint64 `json:"message_id,string"`
}

type DmMessages struct {
	*pgxpool.Pool
}

func newDmMessages(db *pgxpool.Pool) *DmMessages {
	return &DmMessages{
		db,
	}
}

var (
	//go:embed sql/archive_dm_messages/schema.sql
	dmMessagesSchema string

	//go:embed sql/archive_dm_messages/insert.sql
	dmMessagesInsert string

	//go:embed sql/archive_dm_messages/get.sql
	dmMessagesGet string
)

func (d *DmMessages) Schema() string {
	return dmMessagesSchema
}

func (d *DmMessages) Set(ctx context.Context, guildId uint64, ticketId int, messageId uint64) error {
	_, err := d.Exec(ctx, dmMessagesInsert, guildId, ticketId, messageId)
	return err
}

func (d *DmMessages) Get(ctx context.Context, guildId uint64, ticketId int) (DmMessage, bool, error) {
	var data DmMessage
	err := d.QueryRow(ctx, dmMessagesGet, guildId, ticketId).Scan(&data.MessageId)
	if err != nil {
		if err == pgx.ErrNoRows {
			return DmMessage{}, false, nil
		} else {
			return DmMessage{}, false, err
		}
	}

	return data, true, nil
}
