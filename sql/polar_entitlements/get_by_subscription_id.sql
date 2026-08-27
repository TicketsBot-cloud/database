SELECT "polar_subscription_id", "entitlement_id", "user_id", "polar_product_id", "status"
FROM polar_entitlements
WHERE "polar_subscription_id" = $1;
