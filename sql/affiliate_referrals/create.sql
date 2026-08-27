INSERT INTO affiliate_referrals (
    "affiliate_code_id", "affiliate_user_id", "referred_user_id",
    "polar_subscription_id", "referred_tier", "referred_sku_id",
    "purchased_days", "credit_days", "status", "redeemable_at"
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING "id", "created_at";
