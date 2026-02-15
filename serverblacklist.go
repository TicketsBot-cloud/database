package database

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type ServerBlacklist struct {
	*pgxpool.Pool
}

type ServerBlacklistEntry struct {
	GuildId     uint64
	Reason      *string
	OwnerId     *uint64
	RealOwnerId *uint64
}

func newServerBlacklist(db *pgxpool.Pool) *ServerBlacklist {
	return &ServerBlacklist{
		db,
	}
}

func (b ServerBlacklist) Schema() string {
	return `CREATE TABLE IF NOT EXISTS server_blacklist("guild_id" int8 NOT NULL UNIQUE, PRIMARY KEY("guild_id"));`
}

func (b *ServerBlacklist) IsBlacklisted(ctx context.Context, guildId uint64) (bool, *string, error) {
	query := `SELECT "reason" FROM server_blacklist WHERE "guild_id" = $1;`

	var reason *string
	if err := b.QueryRow(ctx, query, guildId).Scan(&reason); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil, nil
		} else {
			return false, nil, err
		}
	}

	return true, reason, nil
}

func (b *ServerBlacklist) Get(ctx context.Context, guildId uint64) (*ServerBlacklistEntry, error) {
	query := `SELECT "guild_id", "reason", "owner_id", "real_owner_id" FROM server_blacklist WHERE "guild_id" = $1;`

	var entry ServerBlacklistEntry
	if err := b.QueryRow(ctx, query, guildId).Scan(&entry.GuildId, &entry.Reason, &entry.OwnerId, &entry.RealOwnerId); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &entry, nil
}

// GetUserBlacklistedOwnerCounts returns how many servers this user was owner of
func (b *ServerBlacklist) GetUserBlacklistedOwnerCounts(ctx context.Context, userId uint64) (int, int, error) {
	query := `SELECT
		(SELECT COUNT(*) FROM server_blacklist WHERE "owner_id" = $1),
		(SELECT COUNT(*) FROM server_blacklist WHERE "real_owner_id" = $1);`

	var serverOwnerCount, realOwnerCount int
	if err := b.QueryRow(ctx, query, userId).Scan(&serverOwnerCount, &realOwnerCount); err != nil {
		return 0, 0, err
	}

	return serverOwnerCount, realOwnerCount, nil
}

func (b *ServerBlacklist) ListAll(ctx context.Context) ([]uint64, error) {
	query := `SELECT "guild_id" FROM server_blacklist;`

	rows, err := b.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	var guilds []uint64
	for rows.Next() {
		var guildId uint64
		if err := rows.Scan(&guildId); err != nil {
			return nil, err
		}

		guilds = append(guilds, guildId)
	}

	return guilds, nil
}

func (b *ServerBlacklist) Add(ctx context.Context, guildId uint64, reason *string, ownerId *uint64, realOwnerId *uint64) (err error) {
	query := `INSERT INTO server_blacklist("guild_id", "reason", "owner_id", "real_owner_id") VALUES($1, $2, $3, $4) ON CONFLICT("guild_id") DO UPDATE SET "reason" = $2, "owner_id" = $3, "real_owner_id" = $4`
	_, err = b.Exec(ctx, query, guildId, reason, ownerId, realOwnerId)
	return
}

func (b *ServerBlacklist) Delete(ctx context.Context, guildId uint64) (err error) {
	_, err = b.Exec(ctx, `DELETE FROM server_blacklist WHERE "guild_id" = $1;`, guildId)
	return
}

