UPDATE affiliate_codes
SET "status" = $2,
    "approved_at" = CASE WHEN $2 = 'active' THEN NOW() ELSE "approved_at" END,
    "approved_by" = CASE WHEN $2 = 'active' THEN $3 ELSE "approved_by" END,
    "revoked_at" = CASE WHEN $2 = 'revoked' THEN NOW() ELSE "revoked_at" END
WHERE "id" = $1;
