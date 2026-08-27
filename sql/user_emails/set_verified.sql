UPDATE user_emails
SET "verified" = $2, "updated_at" = NOW()
WHERE "user_id" = $1;
