-- name: GetAccountBalance :one
SELECT *
FROM account_balances
WHERE account_id = $1
AND currency = $2;

-- name: InsertAccountBalance :one
INSERT INTO account_balances (account_id, currency, balance, version)
VALUES ($1, $2, $3, 1)
  RETURNING *;

-- name: UpdateAccountBalance :one
UPDATE account_balances
SET balance = $3,
    version = version + 1,
    updated_at = now()
WHERE account_id = $1
  AND currency = $2
  AND version = $4
RETURNING *;
