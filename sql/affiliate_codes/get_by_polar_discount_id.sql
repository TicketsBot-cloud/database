SELECT "id", "user_id", "code", "polar_discount_id", "status",
       "discount_basis_points", "credit_percentage",
       "created_at", "approved_at", "approved_by", "revoked_at"
FROM affiliate_codes
WHERE "polar_discount_id" = $1;
