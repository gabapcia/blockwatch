-- name: FilterMonitoredWallets :many
SELECT w."address"
FROM "monitored_wallets" w
WHERE w."network" = UPPER(sqlc.arg('network')) AND w."address" = ANY(sqlc.arg('addresses')::TEXT[]);
