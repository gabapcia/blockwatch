-- Drop the trigger that enforces uppercase for the network field
DROP TRIGGER IF EXISTS trg_uppercase_network ON "monitored_wallets";

-- Drop the trigger function
DROP FUNCTION IF EXISTS enforce_uppercase_network;

-- Drop the monitored_wallets table
DROP TABLE IF EXISTS "monitored_wallets";
