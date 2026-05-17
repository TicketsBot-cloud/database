SELECT COUNT(*)
FROM affiliate_referrals
WHERE "affiliate_user_id" = $1
  AND "status" = 'pending'
  AND "redeemable_at" > NOW();
