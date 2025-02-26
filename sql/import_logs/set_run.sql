INSERT INTO import_logs 
    (guild_id, log_type, run_id, run_log_id, run_type)
VALUES
    ($1, $2, $3, 1, $4)