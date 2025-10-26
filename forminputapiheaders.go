package database

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type FormInputApiHeader struct {
	Id          int    `json:"id"`
	ApiConfigId int    `json:"api_config_id"`
	HeaderName  string `json:"header_name"`
	HeaderValue string `json:"header_value"`
	IsSecret    bool   `json:"is_secret"`
}

type FormInputApiHeaderTable struct {
	*pgxpool.Pool
}

func newFormInputApiHeaderTable(db *pgxpool.Pool) *FormInputApiHeaderTable {
	return &FormInputApiHeaderTable{
		db,
	}
}

func (f FormInputApiHeaderTable) Schema() string {
	return `
	CREATE TABLE IF NOT EXISTS form_input_api_headers(
		"id" SERIAL NOT NULL UNIQUE,
		"api_config_id" INT NOT NULL,
		"header_name" VARCHAR(255) NOT NULL,
		"header_value" TEXT NOT NULL,
		"is_secret" BOOLEAN DEFAULT FALSE,
		FOREIGN KEY("api_config_id") REFERENCES form_input_api_config("id") ON DELETE CASCADE,
		UNIQUE("api_config_id", "header_name"),
		PRIMARY KEY("id")
	);
	CREATE INDEX IF NOT EXISTS form_input_api_headers_api_config_id ON form_input_api_headers("api_config_id");
	`
}

func (f *FormInputApiHeaderTable) Get(ctx context.Context, id int) (header FormInputApiHeader, ok bool, e error) {
	query := `
	SELECT "id", "api_config_id", "header_name", "header_value", "is_secret"
	FROM form_input_api_headers
	WHERE "id" = $1;`

	err := f.QueryRow(ctx, query, id).Scan(
		&header.Id,
		&header.ApiConfigId,
		&header.HeaderName,
		&header.HeaderValue,
		&header.IsSecret,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return FormInputApiHeader{}, false, nil
		} else {
			return FormInputApiHeader{}, false, err
		}
	}

	return header, true, nil
}

func (f *FormInputApiHeaderTable) GetByApiConfig(ctx context.Context, apiConfigId int) ([]FormInputApiHeader, error) {
	query := `
	SELECT "id", "api_config_id", "header_name", "header_value", "is_secret"
	FROM form_input_api_headers
	WHERE "api_config_id" = $1
	ORDER BY "header_name" ASC;`

	rows, err := f.Query(ctx, query, apiConfigId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var headers []FormInputApiHeader
	for rows.Next() {
		var header FormInputApiHeader
		if err := rows.Scan(
			&header.Id,
			&header.ApiConfigId,
			&header.HeaderName,
			&header.HeaderValue,
			&header.IsSecret,
		); err != nil {
			return nil, err
		}
		headers = append(headers, header)
	}

	return headers, rows.Err()
}

func (f *FormInputApiHeaderTable) GetByFormInput(ctx context.Context, formInputId int) ([]FormInputApiHeader, error) {
	query := `
	SELECT h."id", h."api_config_id", h."header_name", h."header_value", h."is_secret"
	FROM form_input_api_headers h
	INNER JOIN form_input_api_config c ON h."api_config_id" = c."id"
	WHERE c."form_input_id" = $1
	ORDER BY h."header_name" ASC;`

	rows, err := f.Query(ctx, query, formInputId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var headers []FormInputApiHeader
	for rows.Next() {
		var header FormInputApiHeader
		if err := rows.Scan(
			&header.Id,
			&header.ApiConfigId,
			&header.HeaderName,
			&header.HeaderValue,
			&header.IsSecret,
		); err != nil {
			return nil, err
		}
		headers = append(headers, header)
	}

	return headers, rows.Err()
}

func (f *FormInputApiHeaderTable) GetHeadersMap(ctx context.Context, apiConfigId int) (map[string]string, error) {
	headers, err := f.GetByApiConfig(ctx, apiConfigId)
	if err != nil {
		return nil, err
	}

	headerMap := make(map[string]string)
	for _, header := range headers {
		headerMap[header.HeaderName] = header.HeaderValue
	}

	return headerMap, nil
}

func (f *FormInputApiHeaderTable) GetAllByGuild(ctx context.Context, guildId uint64) (map[int][]FormInputApiHeader, error) {
	query := `
		SELECT h."id", h."api_config_id", h."header_name", h."header_value", h."is_secret"
		FROM form_input_api_headers h
		INNER JOIN form_input_api_config c ON h."api_config_id" = c."id"
		INNER JOIN form_input i ON c."form_input_id" = i."id"
		INNER JOIN forms f ON i."form_id" = f."form_id"
		WHERE f."guild_id" = $1
		ORDER BY h."api_config_id", h."header_name" ASC;`

	rows, err := f.Query(ctx, query, guildId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	headers := make(map[int][]FormInputApiHeader)
	for rows.Next() {
		var header FormInputApiHeader
		if err := rows.Scan(
			&header.Id,
			&header.ApiConfigId,
			&header.HeaderName,
			&header.HeaderValue,
			&header.IsSecret,
		); err != nil {
			return nil, err
		}

		if _, ok := headers[header.ApiConfigId]; !ok {
			headers[header.ApiConfigId] = make([]FormInputApiHeader, 0)
		}
		headers[header.ApiConfigId] = append(headers[header.ApiConfigId], header)
	}

	return headers, rows.Err()
}

func (f *FormInputApiHeaderTable) Create(ctx context.Context, apiConfigId int, headerName string, headerValue string, isSecret bool) (int, error) {
	query := `
	INSERT INTO form_input_api_headers("api_config_id", "header_name", "header_value", "is_secret")
	VALUES($1, $2, $3, $4)
	RETURNING "id";`

	var id int
	if err := f.QueryRow(ctx, query, apiConfigId, headerName, headerValue, isSecret).Scan(&id); err != nil {
		return 0, err
	}

	return id, nil
}

func (f *FormInputApiHeaderTable) CreateTx(ctx context.Context, tx pgx.Tx, apiConfigId int, headerName string, headerValue string, isSecret bool) (int, error) {
	query := `
	INSERT INTO form_input_api_headers("api_config_id", "header_name", "header_value", "is_secret")
	VALUES($1, $2, $3, $4)
	RETURNING "id";`

	var id int
	if err := tx.QueryRow(ctx, query, apiConfigId, headerName, headerValue, isSecret).Scan(&id); err != nil {
		return 0, err
	}

	return id, nil
}

func (f *FormInputApiHeaderTable) BulkCreate(ctx context.Context, apiConfigId int, headers map[string]string, secretHeaders map[string]bool) error {
	tx, err := f.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for name, value := range headers {
		isSecret := false
		if secretHeaders != nil {
			isSecret = secretHeaders[name]
		}

		if _, err := f.CreateTx(ctx, tx, apiConfigId, name, value, isSecret); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (f *FormInputApiHeaderTable) Update(ctx context.Context, id int, headerValue string, isSecret bool) error {
	query := `
	UPDATE form_input_api_headers
	SET "header_value" = $2,
		"is_secret" = $3
	WHERE "id" = $1;`

	_, err := f.Exec(ctx, query, id, headerValue, isSecret)
	return err
}

func (f *FormInputApiHeaderTable) UpdateTx(ctx context.Context, tx pgx.Tx, id int, headerValue string, isSecret bool) error {
	query := `
	UPDATE form_input_api_headers
	SET "header_value" = $2,
		"is_secret" = $3
	WHERE "id" = $1;`

	_, err := tx.Exec(ctx, query, id, headerValue, isSecret)
	return err
}

func (f *FormInputApiHeaderTable) Upsert(ctx context.Context, apiConfigId int, headerName string, headerValue string, isSecret bool) error {
	query := `
	INSERT INTO form_input_api_headers("api_config_id", "header_name", "header_value", "is_secret")
	VALUES($1, $2, $3, $4)
	ON CONFLICT("api_config_id", "header_name")
	DO UPDATE SET
		"header_value" = EXCLUDED.header_value,
		"is_secret" = EXCLUDED.is_secret;`

	_, err := f.Exec(ctx, query, apiConfigId, headerName, headerValue, isSecret)
	return err
}

func (f *FormInputApiHeaderTable) UpsertTx(ctx context.Context, tx pgx.Tx, apiConfigId int, headerName string, headerValue string, isSecret bool) error {
	query := `
	INSERT INTO form_input_api_headers("api_config_id", "header_name", "header_value", "is_secret")
	VALUES($1, $2, $3, $4)
	ON CONFLICT("api_config_id", "header_name")
	DO UPDATE SET
		"header_value" = EXCLUDED.header_value,
		"is_secret" = EXCLUDED.is_secret;`

	_, err := tx.Exec(ctx, query, apiConfigId, headerName, headerValue, isSecret)
	return err
}

func (f *FormInputApiHeaderTable) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM form_input_api_headers WHERE "id" = $1;`
	_, err := f.Exec(ctx, query, id)
	return err
}

func (f *FormInputApiHeaderTable) DeleteTx(ctx context.Context, tx pgx.Tx, id int) error {
	query := `DELETE FROM form_input_api_headers WHERE "id" = $1;`
	_, err := tx.Exec(ctx, query, id)
	return err
}

func (f *FormInputApiHeaderTable) DeleteByApiConfig(ctx context.Context, apiConfigId int) error {
	query := `DELETE FROM form_input_api_headers WHERE "api_config_id" = $1;`
	_, err := f.Exec(ctx, query, apiConfigId)
	return err
}

func (f *FormInputApiHeaderTable) DeleteByApiConfigTx(ctx context.Context, tx pgx.Tx, apiConfigId int) error {
	query := `DELETE FROM form_input_api_headers WHERE "api_config_id" = $1;`
	_, err := tx.Exec(ctx, query, apiConfigId)
	return err
}

func (f *FormInputApiHeaderTable) DeleteByName(ctx context.Context, apiConfigId int, headerName string) error {
	query := `DELETE FROM form_input_api_headers WHERE "api_config_id" = $1 AND "header_name" = $2;`
	_, err := f.Exec(ctx, query, apiConfigId, headerName)
	return err
}

func (f *FormInputApiHeaderTable) DeleteByNameTx(ctx context.Context, tx pgx.Tx, apiConfigId int, headerName string) error {
	query := `DELETE FROM form_input_api_headers WHERE "api_config_id" = $1 AND "header_name" = $2;`
	_, err := tx.Exec(ctx, query, apiConfigId, headerName)
	return err
}
