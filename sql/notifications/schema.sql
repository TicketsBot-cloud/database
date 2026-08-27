CREATE TABLE IF NOT EXISTS notifications (
    "id"         BIGSERIAL PRIMARY KEY,
    "user_id"    INT8 NOT NULL,
    "category"   TEXT NOT NULL,
    "title"      TEXT NOT NULL,
    "body"       TEXT NOT NULL,
    "link"       TEXT,
    "read"       BOOLEAN NOT NULL DEFAULT FALSE,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_notifications_user_id_read ON notifications("user_id", "read");
CREATE INDEX IF NOT EXISTS idx_notifications_user_id_created ON notifications("user_id", "created_at" DESC);
