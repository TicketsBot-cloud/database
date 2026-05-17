CREATE TABLE IF NOT EXISTS user_emails (
    "user_id"    INT8 NOT NULL PRIMARY KEY,
    "email"      TEXT NOT NULL,
    "verified"   BOOLEAN NOT NULL DEFAULT FALSE,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
