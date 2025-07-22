-- name: ClaimOrExtendIfExpired :execrows
INSERT INTO "walletwatch_idempotency" ("id", "network", "block_hash", "in_progress_until")
VALUES ($1, $2, $3, $4)
ON CONFLICT ON CONSTRAINT "walletwatch_idempotency_unique"
DO UPDATE
SET
    "in_progress_until" = EXCLUDED."in_progress_until"
WHERE
    "walletwatch_idempotency"."finished_at" IS NULL AND
    "walletwatch_idempotency"."in_progress_until" < NOW();

-- name: FindClaim :one
SELECT i.*
FROM "walletwatch_idempotency" i
WHERE
    i."network" = UPPER(sqlc.arg('network')) AND
    i."block_hash" = sqlc.arg('block_hash');

-- name: MarkClaimAsCompleted :exec
UPDATE "walletwatch_idempotency"
SET
    "finished_at" = NOW()
WHERE
    "network" = UPPER(sqlc.arg('network')) AND
    "block_hash" = sqlc.arg('block_hash') AND
    "finished_at" IS NULL;
