UPDATE affiliate_referrals
SET "status" = 'voided',
    "voided_at" = NOW(),
    "voided_reason" = $2
WHERE "polar_subscription_id" = $1
  AND "status" IN ('pending', 'redeemable');
