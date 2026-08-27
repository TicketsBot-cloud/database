UPDATE affiliate_referrals
SET "status" = $2,
    "redeemed_at" = CASE WHEN $2 = 'redeemed' THEN NOW() ELSE "redeemed_at" END,
    "voided_at" = CASE WHEN $2 IN ('voided', 'flagged') THEN NOW() ELSE "voided_at" END
WHERE "id" = $1;
