package database

import (
	"context"
	_ "embed"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type ArchiveDmMessage struct {
	MessageId uint64 `json:"message_id,string"`
}

type ArchiveDmMessages struct {
	*pgxpool.Pool
}

func newArchiveDmMessages(db *pgxpool.Pool) *ArchiveDmMessages {
	return &ArchiveDmMessages{
		db,
	}
}

var (
	//go:embed sql/archive_dm_messages/schema.sql
	archiveDmMessagesSchema string

	//go:embed sql/archive_dm_messages/insert.sql
	archiveDmMessagesInsert string

	//go:embed sql/archive_dm_messages/get.sql
	archiveDmMessagesGet string
)

func (d *ArchiveDmMessages) Schema() string {
	return archiveDmMessagesSchema
}

func (d *ArchiveDmMessages) Set(ctx context.Context, guildId uint64, ticketId int, messageId uint64) error {
	_, err := d.Exec(ctx, archiveDmMessagesInsert, guildId, ticketId, messageId)
	return err
}

func (d *ArchiveDmMessages) Get(ctx context.Context, guildId uint64, ticketId int) (ArchiveDmMessage, bool, error) {
	var data ArchiveDmMessage
	err := d.QueryRow(ctx, archiveDmMessagesGet, guildId, ticketId).Scan(&data.MessageId)
	if err != nil {
		if err == pgx.ErrNoRows {
			return ArchiveDmMessage{}, false, nil
		} else {
			return ArchiveDmMessage{}, false, err
		}
	}

	return data, true, nil
}
