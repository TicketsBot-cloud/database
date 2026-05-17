SELECT "id", "user_id", "category", "title", "body", "link", "read", "created_at"
FROM notifications
WHERE "user_id" = $1
  AND ($2::TEXT IS NULL OR "category" = $2)
ORDER BY "created_at" DESC
LIMIT $3 OFFSET $4;
