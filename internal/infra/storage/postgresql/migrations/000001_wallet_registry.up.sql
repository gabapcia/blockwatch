-- Create the table to store monitored wallets
CREATE TABLE IF NOT EXISTS "monitored_wallets" (
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT NOW(), -- Timestamp when the wallet was registered
    "id" UUID NOT NULL PRIMARY KEY,                  -- Unique identifier for the monitored wallet
    "network" TEXT NOT NULL,                         -- Blockchain network name (e.g., ETHEREUM, SOLANA)
    "address" TEXT NOT NULL,                         -- Wallet address being monitored

    CONSTRAINT "monitored_wallet_unique" UNIQUE ("network", "address") -- Ensures uniqueness of wallet per network
);

-- Add metadata comments to the table and columns
COMMENT ON TABLE "monitored_wallets" IS 'Stores wallets being monitored by network and address';
COMMENT ON COLUMN "monitored_wallets"."created_at" IS 'Timestamp when the wallet was registered';
COMMENT ON COLUMN "monitored_wallets"."id" IS 'Unique identifier for the monitored wallet';
COMMENT ON COLUMN "monitored_wallets"."network" IS 'Blockchain network name in uppercase (e.g., ETHEREUM)';
COMMENT ON COLUMN "monitored_wallets"."address" IS 'Wallet address being monitored';
COMMENT ON CONSTRAINT "monitored_wallet_unique" ON "monitored_wallets" IS 'Ensures that no duplicate wallet exists for the same network';

-- Trigger to enforce that the network name is always stored in uppercase
CREATE FUNCTION enforce_uppercase_network() RETURNS trigger AS $$
BEGIN
    NEW."network" := UPPER(NEW."network");
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Attach the trigger to the monitored_wallets table
CREATE TRIGGER trg_uppercase_network
BEFORE INSERT OR UPDATE ON "monitored_wallets"
FOR EACH ROW
EXECUTE FUNCTION enforce_uppercase_network();

-- Add comment to the trigger function
COMMENT ON FUNCTION enforce_uppercase_network() IS 'Ensures the "network" field is always stored in uppercase letters';
