package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

// ExperimentExposure records that one unit was enrolled in one experiment. It is
// the assignment data GrowthBook reads when computing experiment results.
type ExperimentExposure struct {
	// ExperimentKey is GrowthBook's string experiment key. Deliberately not named
	// experiment_id: the unrelated legacy experiments table has an integer id
	// column, and GrowthBook's assignment query aliases this to experiment_id.
	ExperimentKey  string    `json:"experiment_key"`
	VariationId    int       `json:"variation_id"`
	IdentifierType string    `json:"identifier_type"`
	Identifier     string    `json:"identifier"`
	FeatureKey     string    `json:"feature_key"`
	ExposedAt      time.Time `json:"exposed_at"`
}

type ExperimentExposuresTable struct {
	*pgxpool.Pool
}

func newExperimentExposuresTable(db *pgxpool.Pool) *ExperimentExposuresTable {
	return &ExperimentExposuresTable{
		db,
	}
}

// Schema deliberately declares no primary key. The table is append-only and
// written in batches from every worker pod, so a BIGSERIAL would add a contended
// sequence and an index write for a column nothing reads.
//
// Writers deduplicate to roughly one row per unit per experiment per day, but
// duplicates are still possible across a deduplication window boundary or if the
// Redis deduplication layer is unavailable. Any assignment query must therefore
// take the first exposure per unit rather than assume uniqueness, and alias
// experiment_key to the experiment_id name GrowthBook expects:
//
//	SELECT identifier      AS user_id,
//	       MIN(exposed_at) AS timestamp,
//	       experiment_key  AS experiment_id,
//	       variation_id
//	FROM experiment_exposures
//	WHERE identifier_type = 'guild_id'
//	GROUP BY identifier, experiment_key, variation_id
func (e ExperimentExposuresTable) Schema() string {
	return `
	CREATE TABLE IF NOT EXISTS experiment_exposures(
		"experiment_key" VARCHAR(255) NOT NULL,
		"variation_id" INT NOT NULL,
		"identifier_type" VARCHAR(64) NOT NULL,
		"identifier" VARCHAR(64) NOT NULL,
		"feature_key" VARCHAR(255) NOT NULL DEFAULT '',
		"exposed_at" TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS experiment_exposures_experiment_exposed_at ON experiment_exposures("experiment_key", "exposed_at");
	CREATE INDEX IF NOT EXISTS experiment_exposures_exposed_at ON experiment_exposures("exposed_at");
	`
}

// InsertBatch appends exposures via COPY. Callers batch to keep the write cost
// off the event path.
func (e *ExperimentExposuresTable) InsertBatch(ctx context.Context, exposures []ExperimentExposure) error {
	if len(exposures) == 0 {
		return nil
	}

	rows := make([][]interface{}, 0, len(exposures))
	for _, exposure := range exposures {
		rows = append(rows, []interface{}{
			exposure.ExperimentKey,
			exposure.VariationId,
			exposure.IdentifierType,
			exposure.Identifier,
			exposure.FeatureKey,
			exposure.ExposedAt,
		})
	}

	_, err := e.CopyFrom(
		ctx,
		pgx.Identifier{"experiment_exposures"},
		[]string{"experiment_key", "variation_id", "identifier_type", "identifier", "feature_key", "exposed_at"},
		pgx.CopyFromRows(rows),
	)

	return err
}

// DeleteOlderThan enforces retention. Nothing calls this yet: it needs a caller
// in a daemon, otherwise the table grows without bound.
func (e *ExperimentExposuresTable) DeleteOlderThan(ctx context.Context, age time.Duration) (int64, error) {
	query := `DELETE FROM experiment_exposures WHERE "exposed_at" < NOW() - $1::interval;`

	tag, err := e.Exec(ctx, query, age)
	if err != nil {
		return 0, err
	}

	return tag.RowsAffected(), nil
}

// CountByExperiment reports rows per variation for one experiment, for the admin
// UI to sanity-check that an experiment is actually collecting data.
func (e *ExperimentExposuresTable) CountByExperiment(ctx context.Context, experimentKey string) (map[int]int, error) {
	query := `
	SELECT "variation_id", COUNT(DISTINCT "identifier")
	FROM experiment_exposures
	WHERE "experiment_key" = $1
	GROUP BY "variation_id";
	`

	rows, err := e.Query(ctx, query, experimentKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[int]int)
	for rows.Next() {
		var variationId, count int
		if err := rows.Scan(&variationId, &count); err != nil {
			return nil, err
		}

		counts[variationId] = count
	}

	return counts, rows.Err()
}
