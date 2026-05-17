CREATE TABLE IF NOT EXISTS notification_preferences (
    "user_id"    INT8 NOT NULL,
    "category"   TEXT NOT NULL,
    "discord_dm" BOOLEAN NOT NULL DEFAULT TRUE,
    "email"      BOOLEAN NOT NULL DEFAULT FALSE,
    "in_app"     BOOLEAN NOT NULL DEFAULT TRUE,
    PRIMARY KEY ("user_id", "category")
);
