INSERT INTO import_logs 
    (guild_id, log_type, run_id, run_type, entity_type, message)
VALUES
    ($1, $2, $3, $4, $5, $6)