SELECT
    pe."polar_subscription_id",
    pe."entitlement_id",
    pe."user_id",
    pe."polar_product_id",
    pe."status",
    e."sku_id",
    s."label" AS "sku_label",
    ss."tier",
    e."expires_at",
    e."guild_id",
    ms."servers_permitted"
FROM polar_entitlements pe
INNER JOIN entitlements e ON pe."entitlement_id" = e."id"
INNER JOIN skus s ON e."sku_id" = s."id"
INNER JOIN subscription_skus ss ON s."id" = ss."sku_id"
LEFT JOIN multi_server_skus ms ON s."id" = ms."sku_id"
WHERE pe."user_id" = $1;
