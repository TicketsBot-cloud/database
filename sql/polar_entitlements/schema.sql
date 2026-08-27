CREATE TABLE IF NOT EXISTS polar_entitlements (
    "polar_subscription_id" TEXT NOT NULL,
    "entitlement_id" UUID NOT NULL,
    "user_id" INT8 NOT NULL,
    "polar_product_id" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'active',
    PRIMARY KEY ("polar_subscription_id"),
    FOREIGN KEY ("entitlement_id") REFERENCES entitlements ("id") ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS polar_entitlements_user_id_idx ON polar_entitlements("user_id");
CREATE INDEX IF NOT EXISTS polar_entitlements_entitlement_id_idx ON polar_entitlements("entitlement_id");
