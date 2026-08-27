DO $$ BEGIN
    ALTER TYPE premium_source ADD VALUE IF NOT EXISTS 'affiliate';
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS affiliate_codes (
    "id"                     UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    "user_id"                INT8 NOT NULL UNIQUE,
    "code"                   TEXT NOT NULL UNIQUE,
    "polar_discount_id"      TEXT,
    "status"                 TEXT NOT NULL DEFAULT 'pending',
    "discount_basis_points"  INT NOT NULL DEFAULT 500,
    "credit_percentage"      INT,
    "created_at"             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "approved_at"            TIMESTAMPTZ,
    "approved_by"            INT8,
    "revoked_at"             TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_affiliate_codes_user_id ON affiliate_codes("user_id");
CREATE INDEX IF NOT EXISTS idx_affiliate_codes_status ON affiliate_codes("status");
