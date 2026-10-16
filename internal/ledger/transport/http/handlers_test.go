package http_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jsonv2 "encoding/json/v2"

	"github.com/chuuch/ledger-core/internal/ledger/service"
	"github.com/chuuch/ledger-core/internal/ledger/store"
	ledgerhttp "github.com/chuuch/ledger-core/internal/ledger/transport/http"
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
		ID: id, Reference: arg.Reference, Description: arg.Description,
		Status: arg.Status, CreatedAt: time.Now().UTC(),
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
		ID: uuid.New(), JournalEntryID: arg.JournalEntryID, AccountID: arg.AccountID,
		Direction: arg.Direction, Amount: arg.Amount, Currency: arg.Currency, CreatedAt: time.Now().UTC(),
	}
	f.postings[arg.JournalEntryID] = append(f.postings[arg.JournalEntryID], p)
	return p, nil
}
func (f *fakeStore) ListPostingsByJournalEntryID(_ context.Context, id uuid.UUID) ([]store.Posting, error) {
	return f.postings[id], nil
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
		AccountID: arg.AccountID, Currency: arg.Currency, Balance: arg.Balance,
		Version: 1, UpdatedAt: time.Now().UTC(),
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

type fakeTx struct{ store *fakeStore }

func (t *fakeTx) WithinTx(_ context.Context, fn func(service.Store) error) error {
	return fn(t.store)
}

func TestPostJournalHTTP(t *testing.T) {
	t.Parallel()
	fs := newFakeStore()
	svc := service.New(fs, &fakeTx{store: fs})
	r := ledgerhttp.NewRouter(svc, nil, nil)

	a1, a2 := uuid.New(), uuid.New()
	body, err := jsonv2.Marshal(map[string]any{
		"description": "seed",
		"currency":    "USD",
		"postings": []map[string]any{
			{"account_id": a1.String(), "direction": "debit", "amount": 1000},
			{"account_id": a2.String(), "direction": "credit", "amount": 1000},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/journals", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestPostJournalHTTP_Unbalanced(t *testing.T) {
	t.Parallel()
	fs := newFakeStore()
	svc := service.New(fs, &fakeTx{store: fs})
	r := ledgerhttp.NewRouter(svc, nil, nil)

	body, _ := jsonv2.Marshal(map[string]any{
		"description": "bad",
		"currency":    "USD",
		"postings": []map[string]any{
			{"account_id": uuid.New().String(), "direction": "debit", "amount": 100},
			{"account_id": uuid.New().String(), "direction": "credit", "amount": 50},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/journals", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}
