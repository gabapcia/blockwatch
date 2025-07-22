-- Create table to store the latest processed block checkpoints for each blockchain network
CREATE TABLE IF NOT EXISTS "chainstream_checkpoint" (
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT NOW(),  -- Timestamp when the checkpoint was created
    "id" UUID NOT NULL PRIMARY KEY,                   -- Unique identifier for the checkpoint
    "network" TEXT NOT NULL,                          -- Blockchain network name (e.g., ETHEREUM), stored in uppercase
    "height" BIGINT NOT NULL,                         -- Block height associated with the checkpoint

    CONSTRAINT "chainstream_checkpoint_unique" UNIQUE ("network", "height") -- Ensures no duplicate checkpoint per height and network
);

COMMENT ON TABLE "chainstream_checkpoint" IS 'Stores block height checkpoints per blockchain network for stream processing';
COMMENT ON COLUMN "chainstream_checkpoint"."created_at" IS 'Timestamp when the checkpoint record was inserted';
COMMENT ON COLUMN "chainstream_checkpoint"."id" IS 'Unique UUID identifier for the checkpoint entry';
COMMENT ON COLUMN "chainstream_checkpoint"."network" IS 'Blockchain network name (stored in uppercase)';
COMMENT ON COLUMN "chainstream_checkpoint"."height" IS 'Block height being tracked as a checkpoint';
COMMENT ON CONSTRAINT "chainstream_checkpoint_unique" ON "chainstream_checkpoint" IS 'Enforces uniqueness of height per network to avoid duplicate checkpoints';

-- Create descending index to efficiently retrieve latest height per network
CREATE INDEX IF NOT EXISTS "idx_chainstream_checkpoint_network_height_desc"
ON "chainstream_checkpoint" ("network", "height" DESC);

COMMENT ON INDEX "idx_chainstream_checkpoint_network_height_desc" IS 'Supports efficient queries for latest block height per network';

-- Trigger to force uppercase storage for the "network" field
CREATE TRIGGER trg_uppercase_network
BEFORE INSERT OR UPDATE ON "chainstream_checkpoint"
FOR EACH ROW
EXECUTE FUNCTION enforce_uppercase_network();

COMMENT ON TRIGGER trg_uppercase_network ON "chainstream_checkpoint" IS 'Ensures the "network" field is always stored in uppercase';
