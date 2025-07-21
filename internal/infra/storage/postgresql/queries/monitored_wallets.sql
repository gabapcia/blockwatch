-- name: InsertMonitoredWallet :exec
INSERT INTO "monitored_wallets" ("id", "network", "address")
VALUES ($1, $2, $3);

-- name: DeleteMonitoredWalletByAddress :execrows
DELETE FROM "monitored_wallets"
WHERE "network" = UPPER(sqlc.arg('network')) AND "address" = sqlc.arg('address');

-- name: FilterMonitoredWallets :many
SELECT w."address"
FROM "monitored_wallets" w
WHERE w."network" = UPPER(sqlc.arg('network')) AND w."address" = ANY(sqlc.arg('addresses')::TEXT[]);
