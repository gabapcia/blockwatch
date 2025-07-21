-- name: InsertMonitoredWallet :exec
INSERT INTO "monitored_wallets" ("id", "network", "address")
VALUES ($1, $2, $3);

-- name: DeleteMonitoredWalletByAddress :execrows
DELETE FROM "monitored_wallets"
WHERE "network" = $1 AND "address" = $2;
