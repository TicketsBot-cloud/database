UPDATE entitlements
SET expires_at = $2
WHERE id = $1;
