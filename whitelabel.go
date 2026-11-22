package database

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

// WhitelabelBot holds the stored credentials of a whitelabel bot. The secret
// fields never serialise, so a struct that reaches a JSON response or export
// cannot leak them. Decryption belongs to callers holding a keyring.
type WhitelabelBot struct {
	UserId          uint64
	BotId           uint64
	PublicKey       string
	Token           *string `json:"-"`
	TokenCiphertext []byte  `json:"-"`
	TokenKeyVersion *int16  `json:"-"`
}

const whitelabelColumns = `"user_id", "bot_id", "public_key", "token", "token_ciphertext", "token_key_version"`

func (b *WhitelabelBot) scanTargets() []any {
	return []any{&b.UserId, &b.BotId, &b.PublicKey, &b.Token, &b.TokenCiphertext, &b.TokenKeyVersion}
}

type WhitelabelBotTable struct {
	*pgxpool.Pool
}

func newWhitelabelBotTable(db *pgxpool.Pool) *WhitelabelBotTable {
	return &WhitelabelBotTable{
		db,
	}
}

func (w WhitelabelBotTable) Schema() string {
	return `
CREATE TABLE IF NOT EXISTS whitelabel(
	"user_id" int8 UNIQUE NOT NULL,
	"bot_id" int8 UNIQUE NOT NULL,
	"public_key" CHAR(64) NOT NULL,
	"token" VARCHAR(84) UNIQUE,
	"token_ciphertext" BYTEA,
	"token_key_version" SMALLINT,
	PRIMARY KEY("user_id")
);
ALTER TABLE whitelabel ADD COLUMN IF NOT EXISTS "token_ciphertext" BYTEA;
ALTER TABLE whitelabel ADD COLUMN IF NOT EXISTS "token_key_version" SMALLINT;
ALTER TABLE whitelabel ALTER COLUMN "token" DROP NOT NULL;
CREATE INDEX IF NOT EXISTS whitelabel_bot_id ON whitelabel("bot_id");
`
}

func (w *WhitelabelBotTable) GetByUserId(ctx context.Context, userId uint64) (WhitelabelBot, error) {
	query := "SELECT " + whitelabelColumns + ` FROM whitelabel WHERE "user_id" = $1;`

	var bot WhitelabelBot
	if err := w.QueryRow(ctx, query, userId).Scan(bot.scanTargets()...); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return WhitelabelBot{}, err
	}

	return bot, nil
}

func (w *WhitelabelBotTable) GetByBotId(ctx context.Context, botId uint64) (WhitelabelBot, error) {
	query := "SELECT " + whitelabelColumns + ` FROM whitelabel WHERE "bot_id" = $1;`

	var bot WhitelabelBot
	if err := w.QueryRow(ctx, query, botId).Scan(bot.scanTargets()...); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return WhitelabelBot{}, err
	}

	return bot, nil
}

const insertIfAbsentQuery = `
INSERT INTO whitelabel("user_id", "bot_id", "public_key", "token", "token_ciphertext", "token_key_version")
VALUES($1, $2, $3, NULL, $4, $5)
ON CONFLICT DO NOTHING;`

const updateIfSnapshotQuery = `
UPDATE whitelabel
SET "bot_id" = $2, "public_key" = $3, "token" = NULL, "token_ciphertext" = $4, "token_key_version" = $5
WHERE "user_id" = $1
	AND "bot_id" = $6
	AND "token" IS NOT DISTINCT FROM $7::varchar
	AND "token_ciphertext" IS NOT DISTINCT FROM $8::bytea
	AND "token_key_version" IS NOT DISTINCT FROM $9::smallint;`

// uniqueViolation is the Postgres SQLSTATE for a unique constraint failure.
const uniqueViolation = "23505"

// WriteEncryptedIfSnapshot stores a sealed token (clearing the plaintext column
// in the same statement) only while the user's row is exactly what the caller
// read earlier. A zero snapshot.BotId means the caller saw no row, so the write
// is an insert that never overwrites anything, including another user's bot.
// It reports false when the row changed in the meantime or the bot id belongs
// to another user, in which case nothing is modified.
func (w *WhitelabelBotTable) WriteEncryptedIfSnapshot(ctx context.Context, snapshot WhitelabelBot, userId, botId uint64, publicKey string, sealed []byte, version int16) (bool, error) {
	if snapshot.BotId == 0 {
		tag, err := w.Exec(ctx, insertIfAbsentQuery, userId, botId, publicKey, sealed, version)
		if err != nil {
			return false, err
		}

		return tag.RowsAffected() > 0, nil
	}

	tag, err := w.Exec(ctx, updateIfSnapshotQuery, userId, botId, publicKey, sealed, version,
		snapshot.BotId, snapshot.Token, snapshot.TokenCiphertext, snapshot.TokenKeyVersion)
	if err != nil {
		var sqlState interface{ SQLState() string }
		if errors.As(err, &sqlState) && sqlState.SQLState() == uniqueViolation {
			return false, nil
		}

		return false, err
	}

	return tag.RowsAffected() > 0, nil
}

const restoreIfCurrentQuery = `
UPDATE whitelabel
SET "bot_id" = $4, "public_key" = $5, "token" = NULL, "token_ciphertext" = $6, "token_key_version" = $7
WHERE "user_id" = $1 AND "bot_id" = $2 AND "token_ciphertext" = $3;`

const deleteIfCurrentQuery = `DELETE FROM whitelabel WHERE "user_id" = $1 AND "bot_id" = $2 AND "token_ciphertext" = $3;`

// RestoreEncryptedIfCurrent replaces the row only while it still holds the
// exact sealed value the caller wrote for rejectedBotId. Sealed values carry a
// random nonce, so the comparison identifies one specific write. It reports
// false when another request has changed the row since, in which case nothing
// is modified.
func (w *WhitelabelBotTable) RestoreEncryptedIfCurrent(ctx context.Context, userId, rejectedBotId uint64, written []byte, botId uint64, publicKey string, sealed []byte, version int16) (bool, error) {
	tag, err := w.Exec(ctx, restoreIfCurrentQuery, userId, rejectedBotId, written, botId, publicKey, sealed, version)
	if err != nil {
		return false, err
	}

	return tag.RowsAffected() > 0, nil
}

// DeleteIfCurrent removes the row only while it still holds the exact sealed
// value the caller wrote for botId. It reports false when another request has
// changed the row since, in which case nothing is deleted.
func (w *WhitelabelBotTable) DeleteIfCurrent(ctx context.Context, userId, botId uint64, written []byte) (bool, error) {
	tag, err := w.Exec(ctx, deleteIfCurrentQuery, userId, botId, written)
	if err != nil {
		return false, err
	}

	return tag.RowsAffected() > 0, nil
}

func (w *WhitelabelBotTable) Delete(ctx context.Context, userId uint64) (*uint64, error) {
	query := `DELETE FROM whitelabel WHERE "user_id"=$1 RETURNING "bot_id";`

	var botId uint64
	if err := w.QueryRow(ctx, query, userId).Scan(&botId); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return &botId, nil
}

func (w *WhitelabelBotTable) GetAll(ctx context.Context) ([]WhitelabelBot, error) {
	query := "SELECT " + whitelabelColumns + ` FROM whitelabel;`

	rows, err := w.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bots []WhitelabelBot
	for rows.Next() {
		var bot WhitelabelBot
		if err := rows.Scan(bot.scanTargets()...); err != nil {
			return nil, err
		}
		bots = append(bots, bot)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return bots, nil
}

func (w *WhitelabelBotTable) GetBotsBySharder(ctx context.Context, sharderTotal, sharderID int) ([]WhitelabelBot, error) {
	query := "SELECT " + whitelabelColumns + ` FROM whitelabel WHERE "bot_id" % $1 = $2;`

	rows, err := w.Query(ctx, query, sharderTotal, sharderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bots []WhitelabelBot
	for rows.Next() {
		var bot WhitelabelBot
		if err := rows.Scan(bot.scanTargets()...); err != nil {
			return nil, err
		}
		bots = append(bots, bot)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return bots, nil
}
