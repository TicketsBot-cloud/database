SELECT COALESCE(SUM("credit_days"), 0)
FROM affiliate_referrals
WHERE "affiliate_user_id" = $1
  AND "status" = 'redeemed';
