UPDATE notifications
SET "read" = TRUE
WHERE "id" = $1
  AND "user_id" = $2;
