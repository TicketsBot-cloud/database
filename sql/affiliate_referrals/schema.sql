CREATE TABLE IF NOT EXISTS affiliate_referrals (
    "id"                     UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    "affiliate_code_id"      UUID NOT NULL REFERENCES affiliate_codes("id"),
    "affiliate_user_id"      INT8 NOT NULL,
    "referred_user_id"       INT8 NOT NULL,
    "polar_subscription_id"  TEXT NOT NULL UNIQUE,
    "referred_tier"          TEXT NOT NULL,
    "referred_sku_id"        UUID NOT NULL,
    "purchased_days"         INT NOT NULL,
    "credit_days"            INT NOT NULL,
    "status"                 TEXT NOT NULL DEFAULT 'pending',
    "created_at"             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "redeemable_at"          TIMESTAMPTZ NOT NULL,
    "redeemed_at"            TIMESTAMPTZ,
    "entitlement_id"         UUID,
    "voided_at"              TIMESTAMPTZ,
    "voided_reason"          TEXT
);

CREATE INDEX IF NOT EXISTS idx_affiliate_referrals_affiliate_user_id ON affiliate_referrals("affiliate_user_id");
CREATE INDEX IF NOT EXISTS idx_affiliate_referrals_status ON affiliate_referrals("status");
CREATE INDEX IF NOT EXISTS idx_affiliate_referrals_redeemable_at ON affiliate_referrals("redeemable_at");
