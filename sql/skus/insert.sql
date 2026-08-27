INSERT INTO skus ("label", "type")
VALUES ($1, $2)
RETURNING "id", "label", "type";
