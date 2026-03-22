UPDATE polar_products
SET "polar_product_id" = $2,
    "sku_id" = $3,
    "name" = $4,
    "description" = $5,
    "interval" = $6,
    "price_gbp" = $7,
    "features" = $8,
    "highlighted" = $9,
    "sort_order" = $10,
    "tier" = $11,
    "servers_permitted" = $12
WHERE "id" = $1;
