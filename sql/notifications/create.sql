INSERT INTO notifications ("user_id", "category", "title", "body", "link")
VALUES ($1, $2, $3, $4, $5)
RETURNING "id", "created_at";
