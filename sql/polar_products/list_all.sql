SELECT "id", "polar_product_id", "sku_id", "name", "description", "interval",
       "price_gbp", "features", "highlighted", "sort_order", "tier", "servers_permitted"
FROM polar_products
ORDER BY "sort_order", "price_gbp";
