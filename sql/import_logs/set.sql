INSERT INTO import_logs 
    (guild_id, log_type, run_id, entity_type, message)
VALUES
    ($1, $2, $3, $4, $5)