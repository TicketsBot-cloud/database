INSERT INTO affiliate_codes ("user_id", "code", "status", "discount_basis_points", "credit_percentage")
VALUES ($1, $2, $3, $4, $5)
RETURNING "id", "created_at";
