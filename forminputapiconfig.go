package database

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type FormInputApiConfig struct {
	Id                   int       `json:"id"`
	FormInputId          int       `json:"form_input_id"`
	EndpointUrl          string    `json:"endpoint_url"`
	Method               string    `json:"method"`
	CacheDurationSeconds *int      `json:"cache_duration_seconds,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type FormInputApiConfigTable struct {
	*pgxpool.Pool
}

func newFormInputApiConfigTable(db *pgxpool.Pool) *FormInputApiConfigTable {
	return &FormInputApiConfigTable{
		db,
	}
}

func (f FormInputApiConfigTable) Schema() string {
	return `
	CREATE TABLE IF NOT EXISTS form_input_api_config(
		"id" SERIAL NOT NULL UNIQUE,
		"form_input_id" INT NOT NULL UNIQUE,
		"endpoint_url" VARCHAR(500) NOT NULL,
		"method" VARCHAR(10) NOT NULL DEFAULT 'GET',
		"cache_duration_seconds" INT DEFAULT 300,
		"created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		"updated_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY("form_input_id") REFERENCES form_input("id") ON DELETE CASCADE,
		CHECK(method IN ('GET', 'POST', 'PUT', 'PATCH', 'DELETE')),
		CHECK(cache_duration_seconds >= 0),
		PRIMARY KEY("id")
	);
	CREATE INDEX IF NOT EXISTS form_input_api_config_form_input_id ON form_input_api_config("form_input_id");
	`
}

func (f *FormInputApiConfigTable) Get(ctx context.Context, formInputId int) (config FormInputApiConfig, ok bool, e error) {
	query := `
	SELECT "id", "form_input_id", "endpoint_url", "method", "cache_duration_seconds", "created_at", "updated_at"
	FROM form_input_api_config
	WHERE "form_input_id" = $1;`

	err := f.QueryRow(ctx, query, formInputId).Scan(
		&config.Id,
		&config.FormInputId,
		&config.EndpointUrl,
		&config.Method,
		&config.CacheDurationSeconds,
		&config.CreatedAt,
		&config.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return FormInputApiConfig{}, false, nil
		} else {
			return FormInputApiConfig{}, false, err
		}
	}

	return config, true, nil
}

func (f *FormInputApiConfigTable) GetById(ctx context.Context, id int) (config FormInputApiConfig, ok bool, e error) {
	query := `
	SELECT "id", "form_input_id", "endpoint_url", "method", "cache_duration_seconds", "created_at", "updated_at"
	FROM form_input_api_config
	WHERE "id" = $1;`

	err := f.QueryRow(ctx, query, id).Scan(
		&config.Id,
		&config.FormInputId,
		&config.EndpointUrl,
		&config.Method,
		&config.CacheDurationSeconds,
		&config.CreatedAt,
		&config.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return FormInputApiConfig{}, false, nil
		} else {
			return FormInputApiConfig{}, false, err
		}
	}

	return config, true, nil
}

func (f *FormInputApiConfigTable) GetByFormId(ctx context.Context, formId int) ([]FormInputApiConfig, error) {
	query := `
	SELECT c."id", c."form_input_id", c."endpoint_url", c."method", c."cache_duration_seconds", c."created_at", c."updated_at"
	FROM form_input_api_config c
	INNER JOIN form_input i ON c."form_input_id" = i."id"
	WHERE i."form_id" = $1
	ORDER BY i."position" ASC;`

	rows, err := f.Query(ctx, query, formId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []FormInputApiConfig
	for rows.Next() {
		var config FormInputApiConfig
		if err := rows.Scan(
			&config.Id,
			&config.FormInputId,
			&config.EndpointUrl,
			&config.Method,
			&config.CacheDurationSeconds,
			&config.CreatedAt,
			&config.UpdatedAt,
		); err != nil {
			return nil, err
		}
		configs = append(configs, config)
	}

	return configs, rows.Err()
}

func (f *FormInputApiConfigTable) GetByFormInputId(ctx context.Context, formInputId int) (config FormInputApiConfig, ok bool, e error) {
	query := `
	SELECT "id", "form_input_id", "endpoint_url", "method", "cache_duration_seconds", "created_at", "updated_at"
	FROM form_input_api_config
	WHERE "form_input_id" = $1;`

	err := f.QueryRow(ctx, query, formInputId).Scan(
		&config.Id,
		&config.FormInputId,
		&config.EndpointUrl,
		&config.Method,
		&config.CacheDurationSeconds,
		&config.CreatedAt,
		&config.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return FormInputApiConfig{}, false, nil
		} else {
			return FormInputApiConfig{}, false, err
		}
	}

	return config, true, nil
}

func (f *FormInputApiConfigTable) GetAllByGuild(ctx context.Context, guildId uint64) (map[int]FormInputApiConfig, error) {

	query := `
		SELECT c."id", c."form_input_id", c."endpoint_url", c."method", c."cache_duration_seconds", c."created_at", c."updated_at"
		FROM form_input_api_config c
		INNER JOIN form_input i ON c."form_input_id" = i."id"
		INNER JOIN forms f ON i."form_id" = f."form_id"
		WHERE f."guild_id" = $1;`

	rows, err := f.Query(ctx, query, guildId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	configs := make(map[int]FormInputApiConfig)
	for rows.Next() {
		var config FormInputApiConfig
		if err := rows.Scan(
			&config.Id,
			&config.FormInputId,
			&config.EndpointUrl,
			&config.Method,
			&config.CacheDurationSeconds,
			&config.CreatedAt,
			&config.UpdatedAt,
		); err != nil {
			return nil, err
		}
		configs[config.FormInputId] = config
	}

	return configs, rows.Err()
}

func (f *FormInputApiConfigTable) Create(ctx context.Context, formInputId int, endpointUrl string, method string, cacheDurationSeconds *int) (int, error) {
	query := `
	INSERT INTO form_input_api_config("form_input_id", "endpoint_url", "method", "cache_duration_seconds")
	VALUES($1, $2, $3, $4)
	RETURNING "id";`

	var id int
	if err := f.QueryRow(ctx, query, formInputId, endpointUrl, method, cacheDurationSeconds).Scan(&id); err != nil {
		return 0, err
	}

	return id, nil
}

func (f *FormInputApiConfigTable) CreateTx(ctx context.Context, tx pgx.Tx, formInputId int, endpointUrl string, method string, cacheDurationSeconds *int) (int, error) {
	query := `
	INSERT INTO form_input_api_config("form_input_id", "endpoint_url", "method", "cache_duration_seconds")
	VALUES($1, $2, $3, $4)
	RETURNING "id";`

	var id int
	if err := tx.QueryRow(ctx, query, formInputId, endpointUrl, method, cacheDurationSeconds).Scan(&id); err != nil {
		return 0, err
	}

	return id, nil
}

func (f *FormInputApiConfigTable) Update(ctx context.Context, id int, endpointUrl string, method string, cacheDurationSeconds *int) error {
	query := `
	UPDATE form_input_api_config
	SET "endpoint_url" = $2,
		"method" = $3,
		"cache_duration_seconds" = $4,
		"updated_at" = CURRENT_TIMESTAMP
	WHERE "id" = $1;`

	_, err := f.Exec(ctx, query, id, endpointUrl, method, cacheDurationSeconds)
	return err
}

func (f *FormInputApiConfigTable) UpdateTx(ctx context.Context, tx pgx.Tx, id int, endpointUrl string, method string, cacheDurationSeconds *int) error {
	query := `
	UPDATE form_input_api_config
	SET "endpoint_url" = $2,
		"method" = $3,
		"cache_duration_seconds" = $4,
		"updated_at" = CURRENT_TIMESTAMP
	WHERE "id" = $1;`

	_, err := tx.Exec(ctx, query, id, endpointUrl, method, cacheDurationSeconds)
	return err
}

func (f *FormInputApiConfigTable) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM form_input_api_config WHERE "id" = $1;`
	_, err := f.Exec(ctx, query, id)
	return err
}

func (f *FormInputApiConfigTable) DeleteTx(ctx context.Context, tx pgx.Tx, id int) error {
	query := `DELETE FROM form_input_api_config WHERE "id" = $1;`
	_, err := tx.Exec(ctx, query, id)
	return err
}

func (f *FormInputApiConfigTable) DeleteByFormInput(ctx context.Context, formInputId int) error {
	query := `DELETE FROM form_input_api_config WHERE "form_input_id" = $1;`
	_, err := f.Exec(ctx, query, formInputId)
	return err
}

func (f *FormInputApiConfigTable) DeleteByFormInputTx(ctx context.Context, tx pgx.Tx, formInputId int) error {
	query := `DELETE FROM form_input_api_config WHERE "form_input_id" = $1;`
	_, err := tx.Exec(ctx, query, formInputId)
	return err
}
