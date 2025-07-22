-- name: InsertCheckpointIfNotExists :exec
INSERT INTO "chainstream_checkpoint" ("id", "network", "height")
VALUES ($1, $2, $3)
ON CONFLICT ON CONSTRAINT "chainstream_checkpoint_unique" DO NOTHING;

-- name: GetLatestCheckpointByNetwork :one
SELECT c."height"
FROM "chainstream_checkpoint" c
WHERE
    c."network" = UPPER(sqlc.arg('network'))
ORDER BY
    c."height" DESC
LIMIT 1;
