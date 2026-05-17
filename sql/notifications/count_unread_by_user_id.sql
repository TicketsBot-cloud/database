SELECT COUNT(*)
FROM notifications
WHERE "user_id" = $1
  AND "read" = FALSE;
