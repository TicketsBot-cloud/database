package database

import (
	stdjson "encoding/json"
	"strings"
	"testing"
)

func TestWhitelabelBotJSONOmitsSecrets(t *testing.T) {
	token := "plaintext-token-value"
	version := int16(3)

	tests := []struct {
		name string
		bot  WhitelabelBot
	}{
		{"plaintext only", WhitelabelBot{UserId: 1, BotId: 2, PublicKey: "pk", Token: &token}},
		{"ciphertext only", WhitelabelBot{UserId: 1, BotId: 2, PublicKey: "pk", TokenCiphertext: []byte("sealed-bytes"), TokenKeyVersion: &version}},
		{"both", WhitelabelBot{UserId: 1, BotId: 2, PublicKey: "pk", Token: &token, TokenCiphertext: []byte("sealed-bytes"), TokenKeyVersion: &version}},
		{"empty", WhitelabelBot{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, err := stdjson.Marshal(tt.bot)
			if err != nil {
				t.Fatal(err)
			}

			var fields map[string]any
			if err := stdjson.Unmarshal(raw, &fields); err != nil {
				t.Fatal(err)
			}

			for _, forbidden := range []string{"Token", "TokenCiphertext", "TokenKeyVersion", "token", "token_ciphertext", "token_key_version"} {
				if _, ok := fields[forbidden]; ok {
					t.Errorf("field %q must not serialise", forbidden)
				}
			}
			if strings.Contains(string(raw), "plaintext-token-value") || strings.Contains(string(raw), "sealed-bytes") {
				t.Error("secret leaked into JSON")
			}
		})
	}
}

// The conditional statements are what make a rollback safe under concurrency,
// so their shape is pinned here because a live database is not available to
// unit tests.
func TestWhitelabelConditionalStatementsAreCompareAndSwap(t *testing.T) {
	for _, tt := range []struct {
		name  string
		query string
	}{
		{"restore", restoreIfCurrentQuery},
		{"delete", deleteIfCurrentQuery},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for _, clause := range []string{`"user_id" = $1`, `"bot_id" = $2`, `"token_ciphertext" = $3`} {
				if !strings.Contains(tt.query, clause) {
					t.Errorf("statement is missing %s", clause)
				}
			}
		})
	}
}

func TestWhitelabelWriteStatementsAreConditional(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		want      []string
		forbidden []string
	}{
		{
			name:  "update only applies to the snapshot",
			query: updateIfSnapshotQuery,
			want: []string{
				`UPDATE whitelabel`,
				`WHERE "user_id" = $1`,
				`"bot_id" = $6`,
				`"token" IS NOT DISTINCT FROM $7::varchar`,
				`"token_ciphertext" IS NOT DISTINCT FROM $8::bytea`,
				`"token_key_version" IS NOT DISTINCT FROM $9::smallint`,
				`"token" = NULL`,
			},
			forbidden: []string{"ON CONFLICT"},
		},
		{
			name:      "insert never overwrites",
			query:     insertIfAbsentQuery,
			want:      []string{"INSERT INTO whitelabel", "ON CONFLICT DO NOTHING"},
			forbidden: []string{"DO UPDATE"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, clause := range tt.want {
				if !strings.Contains(tt.query, clause) {
					t.Errorf("statement is missing %s", clause)
				}
			}
			for _, clause := range tt.forbidden {
				if strings.Contains(tt.query, clause) {
					t.Errorf("statement must not contain %s", clause)
				}
			}
		})
	}
}
