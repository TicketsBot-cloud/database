package database

import (
	"context"

	"github.com/jackc/pgx/v4/pgxpool"
)

type Experiment struct {
	Id                int    `json:"id"`
	Name              string `json:"name"`
	RolloutPercentage int    `json:"rollout_percentage"`
}

type ExperimentTable struct {
	*pgxpool.Pool
}

func newExperimentTable(db *pgxpool.Pool) *ExperimentTable {
	return &ExperimentTable{
		db,
	}
}

func (e ExperimentTable) Schema() string {
	return `
	CREATE TABLE IF NOT EXISTS experiments(
		"id" SERIAL NOT NULL UNIQUE,
		"name" VARCHAR(255) NOT NULL,
		"rollout_percentage" INT NOT NULL,
		PRIMARY KEY("id")
	);
	CREATE UNIQUE INDEX IF NOT EXISTS experiments_name ON experiments("name");
	`
}

func (e *ExperimentTable) GetAll(ctx context.Context) (experiments []Experiment, err error) {
	query := `SELECT "id", "name", "rollout_percentage" FROM experiments;`

	rows, err := e.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var experiment Experiment
		err := rows.Scan(&experiment.Id, &experiment.Name, &experiment.RolloutPercentage)
		if err != nil {
			return nil, err
		}
		experiments = append(experiments, experiment)
	}

	return experiments, nil
}

func (e *ExperimentTable) GetByName(ctx context.Context, name string) (*Experiment, error) {
	query := `SELECT "id", "name", "rollout_percentage" FROM experiments WHERE "name" = $1;`

	row := e.QueryRow(ctx, query, name)
	var experiment Experiment
	err := row.Scan(&experiment.Id, &experiment.Name, &experiment.RolloutPercentage)
	if err != nil {
		return nil, err
	}

	return &experiment, nil
}

func (e *ExperimentTable) SetRolloutPercentage(ctx context.Context, name string, percentage int) error {
	query := `
	INSERT INTO experiments ("name", "rollout_percentage")
	VALUES ($1, $2)
	ON CONFLICT ("name") DO UPDATE SET "rollout_percentage" = EXCLUDED."rollout_percentage";
	`

	_, err := e.Exec(ctx, query, name, percentage)
	return err
}

func (e *ExperimentTable) Delete(ctx context.Context, name string) error {
	query := `DELETE FROM experiments WHERE "name" = $1;`

	_, err := e.Exec(ctx, query, name)
	return err
}
