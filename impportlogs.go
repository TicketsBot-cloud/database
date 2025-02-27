package database

import (
	"context"
	_ "embed"
	"strings"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

type ImportLogsTable struct {
	*pgxpool.Pool
}

type ImportRun struct {
	RunId   int          `json:"run_id"`
	RunType string       `json:"run_type"`
	Date    time.Time    `json:"date"`
	Logs    []ImportLogs `json:"logs"`
}

type ImportLogs struct {
	GuildId    uint64    `json:"guild_id"`
	LogType    string    `json:"log_type"`
	RunType    string    `json:"run_type"`
	RunId      int       `json:"run_id"`
	RunLogId   int       `json:"run_log_id"`
	EntityType *string   `json:"entity_type"`
	Message    *string   `json:"message"`
	Date       time.Time `json:"date"`
}

var (
	//go:embed sql/import_logs/schema.sql
	importLogsSchema string

	//go:embed sql/import_logs/set.sql
	importLogsSet string

	//go:embed sql/import_logs/set_run.sql
	importLogsSetRun string
)

func newImportLogs(db *pgxpool.Pool) *ImportLogsTable {
	return &ImportLogsTable{
		db,
	}
}

func (s ImportLogsTable) Schema() string {
	return importLogsSchema
}

func (s *ImportLogsTable) GetRuns(ctx context.Context, guildId uint64) ([]ImportRun, error) {
	query := `SELECT run_id, run_type, date FROM import_logs WHERE "guild_id" = $1 AND log_type = 'RUN_START';`

	var runs []ImportRun

	rows, err := s.Query(ctx, query, guildId)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var mappingEntry ImportRun
		if err := rows.Scan(&mappingEntry.RunId, &mappingEntry.RunType, &mappingEntry.Date); err != nil {
			return nil, err
		}

		runs = append(runs, mappingEntry)
	}

	for i := range runs {
		logs, err := s.GetRunLogs(ctx, guildId, runs[i].RunId)
		if err != nil {
			return nil, err
		}

		runs[i].Logs = logs
	}

	return runs, nil
}

func (s *ImportLogsTable) GetRunLogs(ctx context.Context, guildId uint64, runId int) ([]ImportLogs, error) {
	query := `SELECT guild_id, log_type, run_id, run_log_id, run_type, entity_type, message, date FROM import_logs WHERE guild_id = $1 AND run_id = $2 ORDER BY run_log_id ASC;`

	var logs []ImportLogs

	rows, err := s.Query(ctx, query, guildId, runId)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var logEntry ImportLogs
		if err := rows.Scan(&logEntry.GuildId, &logEntry.LogType, &logEntry.RunId, &logEntry.RunLogId, &logEntry.RunType, &logEntry.EntityType, &logEntry.Message, &logEntry.Date); err != nil {
			return nil, err
		}

		logs = append(logs, logEntry)
	}

	return logs, nil
}

func (s *ImportLogsTable) CreateRun(ctx context.Context, guildId uint64, runType string) (int, error) {
	runCount := 1
	currentRuns, _ := s.GetRuns(ctx, guildId)

	runCount += len(currentRuns)

	_, err := s.Exec(ctx, importLogsSetRun, guildId, "RUN_START", runCount, runType)
	return runCount, err
}

func (s *ImportLogsTable) AddLog(ctx context.Context, guildId uint64, runId int, runType string, logType string, entityType string, message string) error {
	return s.addLog(ctx, guildId, runId, runType, logType, entityType, message, 1)
}

func (s *ImportLogsTable) addLog(ctx context.Context, guildId uint64, runId int, runType string, logType string, entityType string, message string, try int) error {
	_, err := s.Exec(ctx, importLogsSet, guildId, logType, runId, runType, entityType, message)
	if err != nil && strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
		// Try again
		if try > 5 {
			return err
		}

		return s.addLog(ctx, guildId, runId, runType, logType, entityType, message, try+1)
	}
	return err
}
