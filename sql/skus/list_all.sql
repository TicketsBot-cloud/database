SELECT s."id", s."label", s."type",
       ss."tier", ss."priority", ss."is_global",
       ms."servers_permitted"
FROM skus s
LEFT JOIN subscription_skus ss ON s."id" = ss."sku_id"
LEFT JOIN multi_server_skus ms ON s."id" = ms."sku_id"
ORDER BY s."label";
