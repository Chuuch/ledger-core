//Package service
package service

import (
	"context"

	"github.com/chuuch/ledger-core/internal/ledger/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Store is the persistence port used by Service.
type Store interface {
	CreateJournalEntry(ctx context.Context, arg store.CreateJournalEntryParams) (store.JournalEntry, error)
	GetJournalEntryByID(ctx context.Context, id uuid.UUID) (store.JournalEntry, error)
	GetJournalEntryByReference(ctx context.Context, reference *string) (store.JournalEntry, error)

	CreatePosting(ctx context.Context, arg store.CreatePostingParams) (store.Posting, error)
	ListPostingsByJournalEntryID(ctx context.Context, journalEntryID uuid.UUID) ([]store.Posting, error)
	ListPostingsByAccountID(ctx context.Context, accountID uuid.UUID) ([]store.Posting, error)

	GetAccountBalance(ctx context.Context, arg store.GetAccountBalanceParams) (store.AccountBalance, error)
	InsertAccountBalance(ctx context.Context, arg store.InsertAccountBalanceParams) (store.AccountBalance, error)
	UpdateAccountBalance(ctx context.Context, arg store.UpdateAccountBalanceParams) (store.AccountBalance, error)
}

// TxRunner runs work inside a DB transaction.
type TxRunner interface {
	WithinTx(ctx context.Context, fn func(Store) error) error
}

// QueriesStore adapts sqlc Queries to Store.
type QueriesStore struct {
	Q *store.Queries
}

func NewQueriesStore(db store.DBTX) *QueriesStore {
	return &QueriesStore{Q: store.New(db)}
}

func (s *QueriesStore) CreateJournalEntry(ctx context.Context, arg store.CreateJournalEntryParams) (store.JournalEntry, error) {
	return s.Q.CreateJournalEntry(ctx, arg)
}
func (s *QueriesStore) GetJournalEntryByID(ctx context.Context, id uuid.UUID) (store.JournalEntry, error) {
	return s.Q.GetJournalEntryByID(ctx, id)
}
func (s *QueriesStore) GetJournalEntryByReference(ctx context.Context, reference *string) (store.JournalEntry, error) {
	return s.Q.GetJournalEntryByReference(ctx, reference)
}
func (s *QueriesStore) CreatePosting(ctx context.Context, arg store.CreatePostingParams) (store.Posting, error) {
	return s.Q.CreatePosting(ctx, arg)
}
func (s *QueriesStore) ListPostingsByJournalEntryID(ctx context.Context, journalEntryID uuid.UUID) ([]store.Posting, error) {
	return s.Q.ListPostingsByJournalEntryID(ctx, journalEntryID)
}
func (s *QueriesStore) ListPostingsByAccountID(ctx context.Context, accountID uuid.UUID) ([]store.Posting, error) {
	return s.Q.ListPostingsByAccountID(ctx, accountID)
}
func (s *QueriesStore) GetAccountBalance(ctx context.Context, arg store.GetAccountBalanceParams) (store.AccountBalance, error) {
	return s.Q.GetAccountBalance(ctx, arg)
}
func (s *QueriesStore) InsertAccountBalance(ctx context.Context, arg store.InsertAccountBalanceParams) (store.AccountBalance, error) {
	return s.Q.InsertAccountBalance(ctx, arg)
}
func (s *QueriesStore) UpdateAccountBalance(ctx context.Context, arg store.UpdateAccountBalanceParams) (store.AccountBalance, error) {
	return s.Q.UpdateAccountBalance(ctx, arg)
}

// PoolTxRunner starts pgx transactions from a pool.
type PoolTxRunner struct {
	Pool interface {
		Begin(ctx context.Context) (pgx.Tx, error)
	}
}

func (r *PoolTxRunner) WithinTx(ctx context.Context, fn func(Store) error) error {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := fn(NewQueriesStore(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
