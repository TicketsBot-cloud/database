package database

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

type GDPRLogsTable struct {
	*pgxpool.Pool
}

type GDPRLog struct {
	Id          int       `json:"id"`
	Requester   string    `json:"requester"` // Sha256 hash of the requester identifier
	RequestType string    `json:"request_type"`
	RequestDate time.Time `json:"request_date"`
	Status      string    `json:"status"`
}

const (
	GdprLogStatusQueued    = "Queued"
	GdprLogStatusCompleted = "Completed"
	GdprLogStatusFailed    = "Failed"
	GdprLogStatusExpired   = "Expired"
)

func HashGdprRequester(userId uint64) string {
	sum := sha256.Sum256([]byte(strconv.FormatUint(userId, 10)))
	return hex.EncodeToString(sum[:])
}

func newGDPRLogs(db *pgxpool.Pool) *GDPRLogsTable {
	return &GDPRLogsTable{
		db,
	}
}

func (s GDPRLogsTable) Schema() string {
	return `
CREATE TABLE IF NOT EXISTS gdpr_logs (
	id SERIAL PRIMARY KEY,
	requester VARCHAR(256) NOT NULL,
	request_type VARCHAR(256) NOT NULL,
	request_date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	status TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS gdpr_logs_requester_request_date ON gdpr_logs(requester, request_date DESC);
`
}

func (s *GDPRLogsTable) InsertLog(requester string, requestType string, status string) (int, error) {
	query := `INSERT INTO gdpr_logs (requester, request_type, status) VALUES ($1, $2, $3) RETURNING id;`

	var id int
	err := s.QueryRow(context.Background(), query, requester, requestType, status).Scan(&id)
	return id, err
}

func (s *GDPRLogsTable) UpdateLogStatus(id int, status string) error {
	query := `UPDATE gdpr_logs SET status = $1 WHERE id = $2;`

	_, err := s.Exec(context.Background(), query, status, id)
	return err
}

func (s *GDPRLogsTable) ListByRequester(ctx context.Context, requester string) ([]GDPRLog, error) {
	// Capped at the newest 100 requests, add pagination if users need older ones.
	query := `SELECT id, requester, request_type, request_date, status FROM gdpr_logs WHERE requester = $1 ORDER BY request_date DESC LIMIT 100;`

	rows, err := s.Query(ctx, query, requester)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []GDPRLog
	for rows.Next() {
		var entry GDPRLog
		if err := rows.Scan(&entry.Id, &entry.Requester, &entry.RequestType, &entry.RequestDate, &entry.Status); err != nil {
			return nil, err
		}

		logs = append(logs, entry)
	}

	return logs, rows.Err()
}
