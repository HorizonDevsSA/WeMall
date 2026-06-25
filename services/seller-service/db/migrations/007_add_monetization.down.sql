DROP TABLE IF EXISTS seller_earnings;

ALTER TABLE seller_payouts
DROP COLUMN IF EXISTS gross_amount,
DROP COLUMN IF EXISTS platform_fee,
DROP COLUMN IF EXISTS net_amount;

ALTER TABLE sellers
DROP COLUMN IF EXISTS commission_rate,
DROP COLUMN IF EXISTS ad_credit_balance;
