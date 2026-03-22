CREATE TABLE IF NOT EXISTS polar_products (
    "id" UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    "polar_product_id" TEXT NOT NULL UNIQUE,
    "sku_id" UUID NOT NULL REFERENCES skus("id"),
    "name" TEXT NOT NULL,
    "description" TEXT NOT NULL DEFAULT '',
    "interval" TEXT NOT NULL DEFAULT 'month',
    "price_gbp" INT NOT NULL,
    "features" TEXT[] NOT NULL DEFAULT '{}',
    "highlighted" BOOLEAN NOT NULL DEFAULT false,
    "sort_order" INT NOT NULL DEFAULT 0,
    "tier" TEXT NOT NULL DEFAULT 'premium',
    "servers_permitted" INT DEFAULT NULL
);
