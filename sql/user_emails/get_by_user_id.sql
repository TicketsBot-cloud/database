SELECT "user_id", "email", "verified", "created_at", "updated_at"
FROM user_emails
WHERE "user_id" = $1;
