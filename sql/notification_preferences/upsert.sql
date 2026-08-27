INSERT INTO notification_preferences ("user_id", "category", "discord_dm", "email", "in_app")
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT ("user_id", "category") DO UPDATE
SET "discord_dm" = $3, "email" = $4, "in_app" = $5;
