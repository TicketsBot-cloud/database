DO $$
BEGIN
    CREATE TYPE sku_type AS ENUM ('subscription', 'consumable', 'durable');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS skus
(
    id    UUID DEFAULT gen_random_uuid(),
    label VARCHAR(255) NOT NULL,
    type  sku_type     NOT NULL,
    PRIMARY KEY (id)
);
