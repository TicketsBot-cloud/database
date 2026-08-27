UPDATE email_verification_codes SET "failed_attempts" = "failed_attempts" + 1 WHERE "user_id" = $1 RETURNING "failed_attempts";
