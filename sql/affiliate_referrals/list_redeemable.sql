SELECT "id", "affiliate_code_id", "affiliate_user_id", "referred_user_id",
       "polar_subscription_id", "referred_tier", "referred_sku_id",
       "purchased_days", "credit_days", "status",
       "created_at", "redeemable_at", "redeemed_at",
       "entitlement_id", "voided_at", "voided_reason"
FROM affiliate_referrals
WHERE "affiliate_user_id" = $1
  AND "status" IN ('pending', 'redeemable')
  AND "redeemable_at" <= NOW()
ORDER BY "created_at" ASC
FOR UPDATE;
