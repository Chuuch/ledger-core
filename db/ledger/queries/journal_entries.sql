-- name: CreateJournalEntry :one
INSERT INTO journal_entries (reference, description, status)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetJournalEntryByID :one
SELECT *
FROM journal_entries
WHERE id = $1;

-- name: GetJournalEntryByReference :one
SELECT *
FROM journal_entries
WHERE reference = $1;
