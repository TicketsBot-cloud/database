package database

import (
	"context"
	"math/rand/v2"
	"os"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

func TestGdprExportTypeCategories(t *testing.T) {
	expected := map[GdprExportType][]string{
		GdprExportTypeServer: {
			"settings", "panels", "support_hours", "multi_panels", "forms", "tags", "staff", "blacklist",
			"knowledge_base", "integrations", "tickets", "transcripts", "audit_log", "gallery",
		},
		GdprExportTypeUser: {
			"settings", "account", "notifications", "tickets", "staff_activity", "premium", "affiliate", "whitelabel",
			"integrations", "gallery", "feedback", "blacklists", "audit_log", "gdpr_requests", "discord_data", "transcripts",
		},
	}

	for exportType, want := range expected {
		got := exportType.Categories()
		if !slices.Equal(got, want) {
			t.Fatalf("%s: got %v, want %v", exportType, got, want)
		}

		got[0] = "changed"
		if exportType.Categories()[0] != want[0] {
			t.Errorf("%s: mutating the result changed the list", exportType)
		}
	}

	if got := GdprExportType("guild").Categories(); got != nil {
		t.Errorf("unknown type: got %v, want nil", got)
	}
}

func TestGdprExportTypeNormaliseCategories(t *testing.T) {
	tests := []struct {
		name       string
		exportType GdprExportType
		in         []string
		want       []string
		ok         bool
	}{
		{"canonical order", GdprExportTypeServer, []string{"gallery", "transcripts", "settings"}, []string{"settings", "transcripts", "gallery"}, true},
		{"deduped", GdprExportTypeUser, []string{"transcripts", "account", "transcripts", "account"}, []string{"account", "transcripts"}, true},
		{"unknown key", GdprExportTypeServer, []string{"settings", "passwords"}, nil, false},
		{"other type's key", GdprExportTypeServer, []string{"settings", "account"}, nil, false},
		{"empty", GdprExportTypeUser, []string{}, nil, false},
		{"nil", GdprExportTypeUser, nil, nil, false},
		{"unknown type", GdprExportType("guild"), []string{"settings"}, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := tt.exportType.NormaliseCategories(tt.in)
			if ok != tt.ok || !slices.Equal(got, tt.want) {
				t.Errorf("got %v, %v; want %v, %v", got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestHashGdprRequester(t *testing.T) {
	want := "a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3"
	if got := HashGdprRequester(123); got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestGdprExportsIntegration(t *testing.T) {
	dsn := os.Getenv("GDPR_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("GDPR_TEST_DATABASE_URL is not set")
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	// Ten open connections let all ten creates contend for the lock at once.
	cfg.MinConns, cfg.MaxConns = 10, 10

	ctx := context.Background()
	pool, err := pgxpool.ConnectConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	db := NewDatabase(pool)
	mustCreate(ctx, pool, db.GdprLogs, db.GdprExports)
	exports := db.GdprExports

	create := func(t *testing.T, requesterId uint64, exportType GdprExportType) (GdprExport, *time.Time) {
		t.Helper()
		row, next, err := exports.CreateQueued(ctx, GdprExport{Type: exportType, RequesterId: requesterId, Categories: []string{GdprExportCategoryTranscripts}}, "ExportUser")
		if err != nil {
			t.Fatal(err)
		}
		return row, next
	}

	mustQueue := func(t *testing.T, requesterId uint64, exportType GdprExportType) GdprExport {
		t.Helper()
		row, next := create(t, requesterId, exportType)
		if next != nil {
			t.Fatalf("blocked until %v", next)
		}
		return row
	}

	nextAllowed := func(t *testing.T, requesterId uint64, exportType GdprExportType) *time.Time {
		t.Helper()
		next, err := exports.NextAllowedAt(ctx, requesterId, exportType)
		if err != nil {
			t.Fatal(err)
		}
		return next
	}

	rowCounts := func(t *testing.T, requesterId uint64) (exportRows, logRows int) {
		t.Helper()
		query := `SELECT (SELECT COUNT(*) FROM gdpr_exports WHERE requester_id = $1), (SELECT COUNT(*) FROM gdpr_logs WHERE requester = $2)`
		if err := pool.QueryRow(ctx, query, requesterId, HashGdprRequester(requesterId)).Scan(&exportRows, &logRows); err != nil {
			t.Fatal(err)
		}
		return
	}

	backdate := func(t *testing.T, column string, id int64, by time.Duration) {
		t.Helper()
		if _, err := pool.Exec(ctx, `UPDATE gdpr_exports SET `+column+` = NOW() - $2::interval WHERE id = $1`, id, by); err != nil {
			t.Fatal(err)
		}
	}

	listedExpired := func(t *testing.T, id int64) bool {
		t.Helper()
		expired, err := exports.ListExpired(ctx, 500)
		if err != nil {
			t.Fatal(err)
		}
		return slices.ContainsFunc(expired, func(e GdprExport) bool { return e.Id == id })
	}

	t.Run("create and limit", func(t *testing.T) {
		requesterId := uint64(rand.Int64())
		if next := nextAllowed(t, requesterId, GdprExportTypeServer); next != nil {
			t.Fatalf("new requester blocked until %v", next)
		}

		in := GdprExport{Type: GdprExportTypeServer, RequesterId: requesterId, GuildIds: []uint64{1, 2}, Categories: []string{"settings", GdprExportCategoryTranscripts}, TicketIds: []int{3}}
		row, next, err := exports.CreateQueued(ctx, in, "ExportGuild")
		if err != nil || next != nil {
			t.Fatalf("got next %v, err %v", next, err)
		}
		if row.Id == 0 || row.Status != GdprExportStatusQueued || row.ExpiresAt.Sub(row.CreatedAt) != 30*24*time.Hour {
			t.Errorf("unexpected row %+v", row)
		}

		got, err := exports.Get(ctx, row.Id)
		if err != nil {
			t.Fatal(err)
		}
		if got.Type != in.Type || got.RequesterId != requesterId || !slices.Equal(got.GuildIds, in.GuildIds) ||
			!slices.Equal(got.Categories, in.Categories) || !slices.Equal(got.TicketIds, in.TicketIds) ||
			got.Status != GdprExportStatusQueued || got.ObjectKey != nil || got.GdprLogId != row.GdprLogId ||
			!got.CreatedAt.Equal(row.CreatedAt) || !got.ExpiresAt.Equal(row.ExpiresAt) {
			t.Errorf("Get: got %+v, want %+v", *got, row)
		}

		var requester, requestType, status string
		if err := pool.QueryRow(ctx, `SELECT requester, request_type, status FROM gdpr_logs WHERE id = $1`, row.GdprLogId).Scan(&requester, &requestType, &status); err != nil {
			t.Fatal(err)
		}
		if requester != HashGdprRequester(requesterId) || requestType != "ExportGuild" || status != GdprLogStatusQueued {
			t.Errorf("log: got %s, %s, %s", requester, requestType, status)
		}

		blocked, next, err := exports.CreateQueued(ctx, in, "ExportGuild")
		if err != nil {
			t.Fatal(err)
		}
		want := row.CreatedAt.Add(24 * time.Hour)
		if blocked.Id != 0 || next == nil || !next.Equal(want) {
			t.Errorf("second create: got %+v, next %v; want next %v", blocked, next, want)
		}
		if next := nextAllowed(t, requesterId, GdprExportTypeServer); next == nil || !next.Equal(want) {
			t.Errorf("NextAllowedAt: got %v, want %v", next, want)
		}
		if exportRows, logRows := rowCounts(t, requesterId); exportRows != 1 || logRows != 1 {
			t.Errorf("blocked create inserted rows: %d exports, %d logs", exportRows, logRows)
		}

		user := mustQueue(t, requesterId, GdprExportTypeUser)
		got, err = exports.Get(ctx, user.Id)
		if err != nil {
			t.Fatal(err)
		}
		if len(got.GuildIds) != 0 || got.TicketIds != nil {
			t.Errorf("user export: got guild_ids %v, ticket_ids %v", got.GuildIds, got.TicketIds)
		}
	})

	t.Run("statuses", func(t *testing.T) {
		requesterId := uint64(rand.Int64())
		for _, status := range []GdprExportStatus{GdprExportStatusFailed, GdprExportStatusNoData} {
			if err := exports.SetStatus(ctx, mustQueue(t, requesterId, GdprExportTypeUser).Id, status); err != nil {
				t.Fatal(err)
			}
		}

		row := mustQueue(t, requesterId, GdprExportTypeUser)
		for _, status := range []GdprExportStatus{GdprExportStatusProcessing, GdprExportStatusReady} {
			if err := exports.SetStatus(ctx, row.Id, status); err != nil {
				t.Fatal(err)
			}
			if nextAllowed(t, requesterId, GdprExportTypeUser) == nil {
				t.Errorf("%s does not block", status)
			}
		}
	})

	t.Run("window", func(t *testing.T) {
		requesterId := uint64(rand.Int64())
		row := mustQueue(t, requesterId, GdprExportTypeServer)

		backdate(t, "created_at", row.Id, 23*time.Hour)
		if _, next := create(t, requesterId, GdprExportTypeServer); next == nil {
			t.Error("23h old export does not block")
		}

		backdate(t, "created_at", row.Id, 25*time.Hour)
		mustQueue(t, requesterId, GdprExportTypeServer)
	})

	t.Run("concurrent", func(t *testing.T) {
		requesterId := uint64(rand.Int64())
		nexts := make([]*time.Time, 10)
		errs := make([]error, len(nexts))

		start := make(chan struct{})
		var wg sync.WaitGroup
		for i := range nexts {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				_, nexts[i], errs[i] = exports.CreateQueued(ctx, GdprExport{Type: GdprExportTypeUser, RequesterId: requesterId, Categories: []string{"settings"}}, "ExportUser")
			}()
		}
		close(start)
		wg.Wait()

		created := 0
		for i, next := range nexts {
			if errs[i] != nil {
				t.Fatal(errs[i])
			}
			if next == nil {
				created++
			}
		}

		exportRows, logRows := rowCounts(t, requesterId)
		if created != 1 || exportRows != 1 || logRows != 1 {
			t.Errorf("got %d created, %d exports, %d logs; want 1 each", created, exportRows, logRows)
		}
	})

	t.Run("ready and expiry", func(t *testing.T) {
		row := mustQueue(t, uint64(rand.Int64()), GdprExportTypeUser)
		key := "user/1.zip"
		if err := exports.SetReady(ctx, row.Id, key); err != nil {
			t.Fatal(err)
		}

		got, err := exports.Get(ctx, row.Id)
		if err != nil {
			t.Fatal(err)
		}
		if got.Status != GdprExportStatusReady || got.ObjectKey == nil || *got.ObjectKey != key {
			t.Errorf("SetReady: got %+v", *got)
		}

		if listedExpired(t, row.Id) {
			t.Error("listed before it expired")
		}
		backdate(t, "expires_at", row.Id, time.Minute)
		if !listedExpired(t, row.Id) {
			t.Error("expired export not listed")
		}

		if err := exports.Delete(ctx, row.Id); err != nil {
			t.Fatal(err)
		}
		if got, err := exports.Get(ctx, row.Id); err != nil || got != nil {
			t.Errorf("after Delete: got %v, %v", got, err)
		}
		if listedExpired(t, row.Id) {
			t.Error("deleted export still listed")
		}
	})

	t.Run("list by requester", func(t *testing.T) {
		requesterId := uint64(rand.Int64())
		hash := HashGdprRequester(requesterId)
		if _, err := pool.Exec(ctx, `INSERT INTO gdpr_logs (requester, request_type, status, request_date) SELECT $1, 'AllMessages', 'Completed', NOW() - n * INTERVAL '1 minute' FROM generate_series(1, 100) n`, hash); err != nil {
			t.Fatal(err)
		}

		server := mustQueue(t, requesterId, GdprExportTypeServer)
		user := mustQueue(t, requesterId, GdprExportTypeUser)
		mustQueue(t, uint64(rand.Int64()), GdprExportTypeUser)
		backdate(t, "created_at", user.Id, time.Hour)

		listed, err := exports.ListByRequester(ctx, requesterId)
		if err != nil {
			t.Fatal(err)
		}
		var ids []int64
		for _, e := range listed {
			ids = append(ids, e.Id)
		}
		if want := []int64{server.Id, user.Id}; !slices.Equal(ids, want) {
			t.Errorf("exports: got %v, want %v", ids, want)
		}

		logs, err := db.GdprLogs.ListByRequester(ctx, hash)
		if err != nil {
			t.Fatal(err)
		}
		if len(logs) != 100 {
			t.Fatalf("got %d logs, want 100", len(logs))
		}
		if newest := logs[0]; newest.Id != user.GdprLogId || newest.Requester != hash || newest.RequestType != "ExportUser" || newest.Status != GdprLogStatusQueued {
			t.Errorf("newest log: got %+v", newest)
		}
		if !slices.IsSortedFunc(logs, func(a, b GDPRLog) int { return b.RequestDate.Compare(a.RequestDate) }) {
			t.Error("logs are not newest first")
		}
	})
}

func TestTranscriptQueriesIntegration(t *testing.T) {
	dsn := os.Getenv("GDPR_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("GDPR_TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	db := NewDatabase(pool)
	db.CreateTables(ctx, pool)

	g1, g2, g3, userId, other := uint64(rand.Int64()), uint64(rand.Int64()), uint64(rand.Int64()), uint64(rand.Int64()), uint64(rand.Int64())
	if g1 > g2 {
		g1, g2 = g2, g1
	}

	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}

	for _, ticket := range []struct {
		id            int
		guildId       uint64
		openerId      uint64
		open          bool
		hasTranscript bool
	}{
		{1, g1, userId, false, true},
		{2, g1, userId, true, true},
		{3, g1, userId, false, false},
		{4, g1, other, false, true},
		{5, g1, other, false, true},
		{6, g2, other, false, true},
		{7, g2, other, false, true},
		{1, g2, userId, false, true},
		{1, g3, other, true, true},
		{2, g3, other, false, false},
	} {
		exec(`INSERT INTO tickets (id, guild_id, user_id, open, open_time, has_transcript, status) VALUES ($1, $2, $3, $4, NOW(), $5, CASE WHEN $4 THEN 'OPEN' ELSE 'CLOSED' END::ticket_status)`,
			ticket.id, ticket.guildId, ticket.openerId, ticket.open, ticket.hasTranscript)
	}
	exec(`INSERT INTO participant (guild_id, ticket_id, user_id) VALUES ($1, 4, $3), ($2, 6, $3), ($1, 1, $3)`, g1, g2, userId)
	exec(`INSERT INTO ticket_members (guild_id, ticket_id, user_id) VALUES ($1, 5, $3), ($2, 6, $3), ($1, 3, $3)`, g1, g2, userId)

	refs, err := db.Tickets.GetUserTranscriptRefs(ctx, userId)
	if err != nil {
		t.Fatal(err)
	}
	if want := []TranscriptRef{{g1, 1}, {g1, 4}, {g1, 5}, {g2, 1}, {g2, 6}}; !slices.Equal(refs, want) {
		t.Errorf("GetUserTranscriptRefs: got %v, want %v", refs, want)
	}

	slices.Reverse(refs)
	if got, want := TranscriptRefGuildIds(refs), []uint64{g2, g1}; !slices.Equal(got, want) {
		t.Errorf("TranscriptRefGuildIds: got %v, want %v", got, want)
	}

	if got, err := db.Tickets.GuildsWithClosedTranscripts(ctx, []uint64{g2, g3, other, g1}); err != nil || !slices.Equal(got, []uint64{g2, g1}) {
		t.Errorf("GuildsWithClosedTranscripts: got %v, %v; want [%d %d]", got, err, g2, g1)
	}

	closed, err := db.Tickets.GetClosedWithTranscript(ctx, g1)
	if err != nil {
		t.Fatal(err)
	}
	if want := []int{1, 4, 5}; !slices.Equal(closed, want) {
		t.Errorf("GetClosedWithTranscript: got %v, want %v", closed, want)
	}

	filtered, err := db.Tickets.FilterClosedWithTranscript(ctx, g1, []int{5, 2, 3, 1, 1, 99, 2147483647})
	if err != nil {
		t.Fatal(err)
	}
	if want := []int{1, 5}; !slices.Equal(filtered, want) {
		t.Errorf("FilterClosedWithTranscript: got %v, want %v", filtered, want)
	}

	if got, err := db.Tickets.FilterClosedWithTranscript(ctx, g1, nil); got != nil || err != nil {
		t.Errorf("FilterClosedWithTranscript empty: got %v, %v; want nil, nil", got, err)
	}

	if _, err := db.Tickets.FilterClosedWithTranscript(ctx, g1, []int{2147483648}); err == nil {
		t.Error("FilterClosedWithTranscript accepted an id outside int4")
	}
}
