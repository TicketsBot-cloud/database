SELECT COUNT(*)
FROM affiliate_codes
WHERE ($1::TEXT IS NULL OR "status" = $1);
