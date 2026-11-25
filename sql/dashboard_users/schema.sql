CREATE TABLE IF NOT EXISTS dashboard_users (
    user_id int8 NOT NULL PRIMARY KEY,
    last_seen timestamptz NOT NULL DEFAULT now()
);
