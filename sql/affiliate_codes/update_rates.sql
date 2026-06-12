UPDATE affiliate_codes
SET "discount_basis_points" = $2,
    "credit_percentage" = $3
WHERE "id" = $1;
