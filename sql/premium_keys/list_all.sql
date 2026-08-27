SELECT pk."key", pk."length", pk."sku_id", pk."generated_at",
       skus."label" AS sku_label, subscription_skus."tier",
       NULL::int8 AS guild_id, NULL::int8 AS activated_by, false AS is_used
FROM premium_keys pk
INNER JOIN skus ON pk.sku_id = skus.id
INNER JOIN subscription_skus ON skus.id = subscription_skus.sku_id

UNION ALL

SELECT uk."key", NULL::interval AS "length", NULL::uuid AS sku_id,
       NULL::timestamptz AS generated_at, NULL AS sku_label, NULL::premium_tier AS tier,
       uk."guild_id", uk."activated_by", true AS is_used
FROM used_keys uk

ORDER BY is_used ASC, generated_at DESC NULLS LAST
LIMIT $1 OFFSET $2;
