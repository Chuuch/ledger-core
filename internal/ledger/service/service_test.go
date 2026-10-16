package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chuuch/ledger-core/internal/ledger/service"
	"github.com/chuuch/ledger-core/internal/ledger/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type fakeStore struct {
	entries  map[uuid.UUID]store.JournalEntry
	postings map[uuid.UUID][]store.Posting
	balances map[string]store.AccountBalance
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		entries:  map[uuid.UUID]store.JournalEntry{},
		postings: map[uuid.UUID][]store.Posting{},
		balances: map[string]store.AccountBalance{},
	}
}

func balKey(accountID uuid.UUID, currency string) string {
	return accountID.String() + ":" + currency
}

func (f *fakeStore) CreateJournalEntry(_ context.Context, arg store.CreateJournalEntryParams) (store.JournalEntry, error) {
	id := uuid.New()
	e := store.JournalEntry{
		ID:          id,
		Reference:   arg.Reference,
		Description: arg.Description,
		Status:      arg.Status,
		CreatedAt:   time.Now().UTC(),
	}
	f.entries[id] = e
	return e, nil
}

func (f *fakeStore) GetJournalEntryByID(_ context.Context, id uuid.UUID) (store.JournalEntry, error) {
	e, ok := f.entries[id]
	if !ok {
		return store.JournalEntry{}, pgx.ErrNoRows
	}
	return e, nil
}

func (f *fakeStore) GetJournalEntryByReference(context.Context, *string) (store.JournalEntry, error) {
	return store.JournalEntry{}, pgx.ErrNoRows
}

func (f *fakeStore) CreatePosting(_ context.Context, arg store.CreatePostingParams) (store.Posting, error) {
	p := store.Posting{
		ID:             uuid.New(),
		JournalEntryID: arg.JournalEntryID,
		AccountID:      arg.AccountID,
		Direction:      arg.Direction,
		Amount:         arg.Amount,
		Currency:       arg.Currency,
		CreatedAt:      time.Now().UTC(),
	}
	f.postings[arg.JournalEntryID] = append(f.postings[arg.JournalEntryID], p)
	return p, nil
}

func (f *fakeStore) ListPostingsByJournalEntryID(_ context.Context, journalEntryID uuid.UUID) ([]store.Posting, error) {
	return f.postings[journalEntryID], nil
}

func (f *fakeStore) ListPostingsByAccountID(context.Context, uuid.UUID) ([]store.Posting, error) {
	return nil, nil
}

func (f *fakeStore) GetAccountBalance(_ context.Context, arg store.GetAccountBalanceParams) (store.AccountBalance, error) {
	b, ok := f.balances[balKey(arg.AccountID, arg.Currency)]
	if !ok {
		return store.AccountBalance{}, pgx.ErrNoRows
	}
	return b, nil
}

func (f *fakeStore) InsertAccountBalance(_ context.Context, arg store.InsertAccountBalanceParams) (store.AccountBalance, error) {
	b := store.AccountBalance{
		AccountID: arg.AccountID,
		Currency:  arg.Currency,
		Balance:   arg.Balance,
		Version:   1,
		UpdatedAt: time.Now().UTC(),
	}
	f.balances[balKey(arg.AccountID, arg.Currency)] = b
	return b, nil
}

func (f *fakeStore) UpdateAccountBalance(_ context.Context, arg store.UpdateAccountBalanceParams) (store.AccountBalance, error) {
	key := balKey(arg.AccountID, arg.Currency)
	b, ok := f.balances[key]
	if !ok || b.Version != arg.Version {
		return store.AccountBalance{}, pgx.ErrNoRows
	}
	b.Balance = arg.Balance
	b.Version++
	b.UpdatedAt = time.Now().UTC()
	f.balances[key] = b
	return b, nil
}

type fakeTx struct {
	store *fakeStore
}

func (t *fakeTx) WithinTx(ctx context.Context, fn func(service.Store) error) error {
	return fn(t.store)
}

func TestPostJournal_Unbalanced(t *testing.T) {
	t.Parallel()
	fs := newFakeStore()
	svc := service.New(fs, &fakeTx{store: fs})

	_, err := svc.PostJournal(context.Background(), service.PostJournalInput{
		Description: "x",
		Currency:    "USD",
		Postings: []service.PostingInput{
			{AccountID: uuid.New(), Direction: "debit", Amount: 100},
			{AccountID: uuid.New(), Direction: "credit", Amount: 50},
		},
	})
	if !errors.Is(err, service.ErrUnbalanced) {
		t.Fatalf("got %v, want ErrUnbalanced", err)
	}
}

func TestPostJournal_OK(t *testing.T) {
	t.Parallel()
	fs := newFakeStore()
	svc := service.New(fs, &fakeTx{store: fs})

	a1, a2 := uuid.New(), uuid.New()
	view, err := svc.PostJournal(context.Background(), service.PostJournalInput{
		Description: "seed",
		Currency:    "USD",
		Postings: []service.PostingInput{
			{AccountID: a1, Direction: "debit", Amount: 1000},
			{AccountID: a2, Direction: "credit", Amount: 1000},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Postings) != 2 {
		t.Fatalf("postings=%d", len(view.Postings))
	}

	b1, err := svc.GetBalance(context.Background(), a1, "USD")
	if err != nil {
		t.Fatal(err)
	}
	b2, err := svc.GetBalance(context.Background(), a2, "USD")
	if err != nil {
		t.Fatal(err)
	}
	if b1.Balance != 1000 || b2.Balance != -1000 {
		t.Fatalf("balances %d %d", b1.Balance, b2.Balance)
	}
}

func TestGetJournal_NotFound(t *testing.T) {
	t.Parallel()
	fs := newFakeStore()
	svc := service.New(fs, &fakeTx{store: fs})
	_, err := svc.GetJournal(context.Background(), uuid.New())
	if !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}
