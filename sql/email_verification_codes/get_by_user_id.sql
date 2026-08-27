SELECT "user_id", "code", "expires_at", "created_at", "failed_attempts"
FROM email_verification_codes
WHERE "user_id" = $1;
