-- name: CreatePosting :one
INSERT INTO postings (
  journal_entry_id,
  account_id,
  direction,
  amount,
  currency
)
VALUES ($1, $2, $3, $4, $5)
  RETURNING *;

-- name: ListPostingsByJournalEntryID :many
SELECT *
FROM postings
WHERE journal_entry_id = $1
ORDER BY created_at ASC;

-- name: ListPostingsByAccountID :many
SELECT *
FROM postings
WHERE account_id = $1
ORDER BY created_at DESC;
