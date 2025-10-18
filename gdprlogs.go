package database

import (
	"context"
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
