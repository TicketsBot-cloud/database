INSERT INTO subscription_skus ("sku_id", "tier", "priority", "is_global")
VALUES ($1, $2, $3, $4)
ON CONFLICT ("sku_id") DO UPDATE SET "tier" = $2, "priority" = $3, "is_global" = $4;
