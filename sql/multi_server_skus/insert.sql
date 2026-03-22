INSERT INTO multi_server_skus ("sku_id", "servers_permitted")
VALUES ($1, $2)
ON CONFLICT ("sku_id") DO UPDATE SET "servers_permitted" = $2;
