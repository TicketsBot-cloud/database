INSERT INTO import_mapping (guild_id, area, source_id, target_id)
VALUES ($1, $2, $3, $4)
ON CONFLICT DO NOTHING;