INSERT INTO polar_entitlements ("polar_subscription_id", "entitlement_id", "user_id", "polar_product_id", "status")
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT ("polar_subscription_id") DO UPDATE SET
    "entitlement_id" = EXCLUDED."entitlement_id",
    "status" = EXCLUDED."status";
