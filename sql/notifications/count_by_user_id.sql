SELECT COUNT(*)
FROM notifications
WHERE "user_id" = $1
  AND ($2::TEXT IS NULL OR "category" = $2);
