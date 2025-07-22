-- Drop triggers in reverse order of creation
DROP TRIGGER IF EXISTS trg_uppercase_network ON "walletwatch_idempotency";
DROP TRIGGER IF EXISTS trg_update_updated_at ON "walletwatch_idempotency";

-- Drop trigger functions
DROP FUNCTION IF EXISTS enforce_uppercase_network();
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop table
DROP TABLE IF EXISTS "walletwatch_idempotency";
