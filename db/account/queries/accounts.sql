-- name: CreateAccount :one
INSERT INTO accounts (owner_id, currency, status)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetAccountByID :one
SELECT *
FROM accounts
WHERE id = $1;

-- name: ListAccountsByOwner :many
SELECT *
FROM accounts
WHERE owner_id = $1
ORDER BY created_at DESC;

-- name: UpdateAccountStatus :one
UPDATE accounts
SET status = $2,
  version = version + 1,
  updated_at = now()
WHERE id = $1
  AND version = $3
  AND status <> 'closed'
RETURNING *;
