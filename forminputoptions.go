package database

import (
	"context"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type FormInputOption struct {
	Id          int     `json:"id"`
	FormInputId int     `json:"form_input_id"`
	Position    int     `json:"position"`
	Label       string  `json:"label"`
	Description *string `json:"description,omitempty"`
	Value       string  `json:"value"`
}

type FormInputOptionTable struct {
	*pgxpool.Pool
}

func newFormInputOptionTable(db *pgxpool.Pool) *FormInputOptionTable {
	return &FormInputOptionTable{
		db,
	}
}

func (f FormInputOptionTable) Schema() string {
	return `
	CREATE TABLE IF NOT EXISTS form_input_option(
	"id" SERIAL NOT NULL UNIQUE,
	"form_input_id" int NOT NULL,
	"position" int NOT NULL,
	"label" VARCHAR(100) NOT NULL,
	"description" VARCHAR(255) NULL,
	"value" VARCHAR(100) NOT NULL,
	FOREIGN KEY("form_input_id") REFERENCES form_input("id") ON DELETE CASCADE,
	UNIQUE("form_input_id", "position") DEFERRABLE INITIALLY DEFERRED,
	CHECK(position >= 1),
	CHECK(position <= 25),
	PRIMARY KEY("id")
	);
	CREATE INDEX IF NOT EXISTS form_input_option_form_input_id ON form_input_options("form_input_id");
	`
}

func (f *FormInputOptionTable) GetOptions(ctx context.Context, formInputId int) (options []FormInputOption, e error) {
	rows, err := f.Query(ctx, `SELECT id, form_input_id, position, label, description, value FROM form_input_option WHERE form_input_id=$1 ORDER BY position ASC`, formInputId)
	if err != nil {
		return options, err
	}
	defer rows.Close()

	for rows.Next() {
		var option FormInputOption
		err := rows.Scan(&option.Id, &option.FormInputId, &option.Position, &option.Label, &option.Description, &option.Value)
		if err != nil {
			return options, err
		}

		options = append(options, option)
	}

	if rows.Err() != nil {
		return options, rows.Err()
	}

	return options, nil
}

func (f *FormInputOptionTable) GetOptionsByForm(ctx context.Context, formId int) (options map[int][]FormInputOption, e error) {
	query := `SELECT o.id, o.form_input_id, o.position, o.label, o.description, o.value
	FROM form_input_option o
	JOIN form_input i ON o.form_input_id = i.id
	WHERE i.form_id = $1
	ORDER BY o.form_input_id, o.position ASC;`

	rows, err := f.Query(ctx, query, formId)
	if err != nil {
		return options, err
	}
	defer rows.Close()
	options = make(map[int][]FormInputOption)

	for rows.Next() {
		var option FormInputOption
		err := rows.Scan(&option.Id, &option.FormInputId, &option.Position, &option.Label, &option.Description, &option.Value)
		if err != nil {
			return options, err
		}

		options[option.FormInputId] = append(options[option.FormInputId], option)
	}

	if rows.Err() != nil {
		return options, rows.Err()
	}

	return options, nil
}

func (f *FormInputOptionTable) GetAllOptionsByGuild(ctx context.Context, guildId uint64) (map[int][]FormInputOption, error) {
	query := `SELECT o.id, o.form_input_id, o.position, o.label, o.description, o.value
	FROM form_input_option o
	JOIN form_input i ON o.form_input_id = i.id
	JOIN forms f ON i.form_id = f.form_id
	WHERE f.guild_id = $1
	ORDER BY o.form_input_id, o.position ASC;`

	rows, err := f.Query(ctx, query, guildId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	optionsMap := make(map[int][]FormInputOption)

	for rows.Next() {
		var option FormInputOption
		err := rows.Scan(&option.Id, &option.FormInputId, &option.Position, &option.Label, &option.Description, &option.Value)
		if err != nil {
			return nil, err
		}

		optionsMap[option.FormInputId] = append(optionsMap[option.FormInputId], option)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return optionsMap, nil
}

func (f *FormInputOptionTable) Create(ctx context.Context, formInputOption FormInputOption) (id int, e error) {
	err := f.QueryRow(ctx, `INSERT INTO form_input_option (form_input_id, position, label, description, value) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		formInputOption.FormInputId,
		formInputOption.Position,
		formInputOption.Label,
		formInputOption.Description,
		formInputOption.Value,
	).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (f *FormInputOptionTable) CreateTx(ctx context.Context, tx pgx.Tx, formInputOption FormInputOption) (id int, e error) {
	err := tx.QueryRow(ctx, `INSERT INTO form_input_option (form_input_id, position, label, description, value) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		formInputOption.FormInputId,
		formInputOption.Position,
		formInputOption.Label,
		formInputOption.Description,
		formInputOption.Value,
	).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (f *FormInputOptionTable) Delete(ctx context.Context, id int) (e error) {
	q := `WITH deleted_positions AS (
		DELETE FROM form_input_option
		WHERE id = $1
		RETURNING form_input_id, position
	)
	UPDATE form_input_option
	SET position = position - 1
	WHERE form_input_id = (SELECT form_input_id FROM deleted_positions)
	AND position > (SELECT position FROM deleted_positions);`

	_, err := f.Exec(ctx, q, id)
	return err
}

func (f *FormInputOptionTable) DeleteTx(ctx context.Context, tx pgx.Tx, id int) (e error) {
	q := `WITH deleted_positions AS (
		DELETE FROM form_input_option
		WHERE id = $1
		RETURNING form_input_id, position
	)
	UPDATE form_input_option
	SET position = position - 1
	WHERE form_input_id = (SELECT form_input_id FROM deleted_positions)
	AND position > (SELECT position FROM deleted_positions);`

	_, err := tx.Exec(ctx, q, id)
	return err
}
