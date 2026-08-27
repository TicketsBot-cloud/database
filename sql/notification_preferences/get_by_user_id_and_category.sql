SELECT "user_id", "category", "discord_dm", "email", "in_app"
FROM notification_preferences
WHERE "user_id" = $1
  AND "category" = $2;
