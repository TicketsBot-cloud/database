package database

import (
	"context"
	_ "embed"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type ImportMappingTable struct {
	*pgxpool.Pool
}

type ImportMapping struct {
	GuildId  uint64 `json:"guild_id"`
	Area     string `json:"area"`
	SourceId int    `json:"source_id"`
	TargetId int    `json:"target_id"`
}

var (
	//go:embed sql/import_mapping/schema.sql
	importMappingSchema string

	//go:embed sql/import_mapping/set.sql
	importMappingSet string
)

func newImportMapping(db *pgxpool.Pool) *ImportMappingTable {
	return &ImportMappingTable{
		db,
	}
}

func (s ImportMappingTable) Schema() string {
	return importMappingSchema
}

func (s *ImportMappingTable) GetMapping(ctx context.Context, guildId uint64) (map[string]map[int]int, error) {
	query := `SELECT * FROM import_mapping WHERE "guild_id" = $1;`

	rows, err := s.Query(ctx, query, guildId)
	if err != nil {
		return nil, err
	}

	mapping := make(map[string]map[int]int)

	for rows.Next() {
		var mappingEntry ImportMapping
		if err := rows.Scan(&mappingEntry.GuildId, &mappingEntry.Area, &mappingEntry.SourceId, &mappingEntry.TargetId); err != nil {
			return nil, err
		}

		if _, ok := mapping[mappingEntry.Area]; !ok {
			mapping[mappingEntry.Area] = make(map[int]int)
		}

		mapping[mappingEntry.Area][mappingEntry.SourceId] = mappingEntry.TargetId
	}

	return mapping, nil
}

func (s *ImportMappingTable) Set(ctx context.Context, guildId uint64, area string, sourceId, targetId int) error {
	_, err := s.Exec(ctx, importMappingSet, guildId, area, sourceId, targetId)
	return err
}

func (s *ImportMappingTable) SetBulk(ctx context.Context, guildId uint64, area string, mappings map[int]int) error {
	rows := make([][]interface{}, len(mappings))

	i := 0
	for sourceId, targetId := range mappings {
		rows[i] = []interface{}{guildId, area, sourceId, targetId}
		i++
	}

	_, err := s.CopyFrom(ctx, pgx.Identifier{"import_mapping"}, []string{"guild_id", "area", "source_id", "target_id"}, pgx.CopyFromRows(rows))

	return err
}
