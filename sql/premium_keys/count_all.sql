SELECT (SELECT COUNT(*) FROM premium_keys) + (SELECT COUNT(*) FROM used_keys);
