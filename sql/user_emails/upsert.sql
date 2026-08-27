INSERT INTO user_emails ("user_id", "email")
VALUES ($1, $2)
ON CONFLICT ("user_id")
DO UPDATE SET "email" = $2, "verified" = FALSE, "updated_at" = NOW();
