UPDATE polar_entitlements
SET "status" = $2
WHERE "polar_subscription_id" = $1;
