-- Drop the trigger that uppercases the "network" field
DROP TRIGGER IF EXISTS trg_uppercase_network ON "chainstream_checkpoint";

-- Drop the descending index used for retrieving latest block heights
DROP INDEX IF EXISTS "idx_chainstream_checkpoint_network_height_desc";

-- Drop the checkpoint table
DROP TABLE IF EXISTS "chainstream_checkpoint";
