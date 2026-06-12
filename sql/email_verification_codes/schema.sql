CREATE TABLE IF NOT EXISTS email_verification_codes (
    "user_id"         INT8 NOT NULL PRIMARY KEY,
    "code"            TEXT NOT NULL,
    "expires_at"      TIMESTAMPTZ NOT NULL,
    "created_at"      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "failed_attempts" INT NOT NULL DEFAULT 0
);
