INSERT INTO email_verification_codes ("user_id", "code", "expires_at", "failed_attempts")
VALUES ($1, $2, $3, 0)
ON CONFLICT ("user_id")
DO UPDATE SET "code" = $2, "expires_at" = $3, "created_at" = NOW(), "failed_attempts" = 0;
