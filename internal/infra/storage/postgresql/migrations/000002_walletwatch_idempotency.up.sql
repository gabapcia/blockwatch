-- Create the walletwatch_idempotency table to track idempotent block processing attempts
CREATE TABLE IF NOT EXISTS "walletwatch_idempotency" (
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT NOW(),    -- Timestamp when the record was created
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT NOW(),    -- Automatically updated on each modification
    "finished_at" TIMESTAMPTZ,                          -- Timestamp when processing finished (nullable)
    "id" UUID NOT NULL,                                 -- Unique identifier for the record
    "network" TEXT NOT NULL,                            -- Blockchain network (e.g., ETHEREUM), stored in uppercase
    "block_hash" TEXT NOT NULL,                         -- Unique hash of the blockchain block
    "in_progress_until" TIMESTAMPTZ NOT NULL,           -- Deadline for in-progress state, calculated externally

    CONSTRAINT "walletwatch_idempotency_unique" UNIQUE ("network", "block_hash") -- Ensures idempotency per network/block pair
);

-- Add comments to describe the purpose of the table and its fields
COMMENT ON TABLE "walletwatch_idempotency" IS 'Tracks block processing attempts in walletwatch with TTL-based in-progress logic for idempotency';
COMMENT ON COLUMN "walletwatch_idempotency"."created_at" IS 'Timestamp when the record was initially inserted';
COMMENT ON COLUMN "walletwatch_idempotency"."updated_at" IS 'Timestamp of the most recent modification, auto-updated via trigger';
COMMENT ON COLUMN "walletwatch_idempotency"."finished_at" IS 'Timestamp when block processing finished (optional)';
COMMENT ON COLUMN "walletwatch_idempotency"."id" IS 'Unique identifier for the idempotency record';
COMMENT ON COLUMN "walletwatch_idempotency"."network" IS 'Blockchain network identifier, always stored in uppercase (e.g., ETHEREUM)';
COMMENT ON COLUMN "walletwatch_idempotency"."block_hash" IS 'Hash of the block being processed, unique per network';
COMMENT ON COLUMN "walletwatch_idempotency"."in_progress_until" IS 'Deadline until which the processing is considered active (used for TTL and concurrency control)';
COMMENT ON CONSTRAINT "walletwatch_idempotency_unique" ON "walletwatch_idempotency" IS 'Guarantees that a specific block is processed only once per network';

-- Trigger function to auto-update the "updated_at" field before every update
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW."updated_at" := NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION update_updated_at_column() IS 'Trigger function that sets updated_at to NOW() before any update';

-- Trigger to automatically update the "updated_at" field
CREATE TRIGGER trg_update_updated_at
BEFORE UPDATE ON "walletwatch_idempotency"
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TRIGGER trg_update_updated_at ON "walletwatch_idempotency" IS 'Keeps updated_at in sync with the current timestamp on every row update';

-- Trigger to enforce uppercase values in the "network" field
CREATE TRIGGER trg_uppercase_network
BEFORE INSERT OR UPDATE ON "walletwatch_idempotency"
FOR EACH ROW
EXECUTE FUNCTION enforce_uppercase_network();

COMMENT ON TRIGGER trg_uppercase_network ON "walletwatch_idempotency" IS 'Ensures the network field is stored in uppercase during inserts and updates';
