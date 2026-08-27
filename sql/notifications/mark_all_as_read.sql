UPDATE notifications
SET "read" = TRUE
WHERE "user_id" = $1
  AND "read" = FALSE;
