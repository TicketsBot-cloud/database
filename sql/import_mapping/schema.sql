CREATE TYPE mapping_area AS ENUM ('ticket', 'form', 'form_input', 'panel');

CREATE TABLE IF NOT EXISTS import_mapping
(
    guild_id   int8 NOT NULL,
    area mapping_area NOT NULL,
    source_id int4 NOT NULL,
    target_id int4 NOT NULL,
    UNIQUE NULLS NOT DISTINCT (guild_id, area, source_id, target_id)
);