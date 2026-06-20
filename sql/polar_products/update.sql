UPDATE polar_products
SET "polar_product_id" = $2,
    "sku_id" = $3,
    "name" = $4,
    "description" = $5,
    "interval" = $6,
    "price" = $7,
    "currency" = $8,
    "features" = $9,
    "highlighted" = $10,
    "sort_order" = $11,
    "tier" = $12,
    "servers_permitted" = $13
WHERE "id" = $1;
