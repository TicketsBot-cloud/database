INSERT INTO polar_products ("polar_product_id", "sku_id", "name", "description", "interval",
                           "price", "currency", "features", "highlighted", "sort_order", "tier", "servers_permitted")
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING "id", "polar_product_id", "sku_id", "name", "description", "interval",
          "price", "currency", "features", "highlighted", "sort_order", "tier", "servers_permitted";
