package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chuuch/ledger-core/internal/account/service"
	"github.com/chuuch/ledger-core/internal/account/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type fakeStore struct {
	createFn func(context.Context, store.CreateAccountParams) (store.Account, error)
	getFn    func(context.Context, uuid.UUID) (store.Account, error)
	listFn   func(context.Context, string) ([]store.Account, error)
	updateFn func(context.Context, store.UpdateAccountStatusParams) (store.Account, error)
}

func (f *fakeStore) CreateAccount(ctx context.Context, arg store.CreateAccountParams) (store.Account, error) {
	return f.createFn(ctx, arg)
}
func (f *fakeStore) GetAccountByID(ctx context.Context, id uuid.UUID) (store.Account, error) {
	return f.getFn(ctx, id)
}
func (f *fakeStore) ListAccountsByOwner(ctx context.Context, ownerID string) ([]store.Account, error) {
	return f.listFn(ctx, ownerID)
}
func (f *fakeStore) UpdateAccountStatus(ctx context.Context, arg store.UpdateAccountStatusParams) (store.Account, error) {
	return f.updateFn(ctx, arg)
}

func TestCreate_ValidatesInput(t *testing.T) {
	t.Parallel()
	svc := service.New(&fakeStore{})

	_, err := svc.Create(context.Background(), service.CreateInput{OwnerID: "", Currency: "USD"})
	if !errors.Is(err, service.ErrInvalidInput) {
		t.Fatalf("got %v, want ErrInvalidInput", err)
	}
}

func TestCreate_DefaultsActive(t *testing.T) {
	t.Parallel()
	var gotStatus string
	svc := service.New(&fakeStore{
		createFn: func(_ context.Context, arg store.CreateAccountParams) (store.Account, error) {
			gotStatus = arg.Status
			return store.Account{
				ID: uuid.New(), OwnerID: arg.OwnerID, Currency: arg.Currency,
				Status: arg.Status, Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
			}, nil
		},
	})

	acc, err := svc.Create(context.Background(), service.CreateInput{OwnerID: "u1", Currency: "usd"})
	if err != nil {
		t.Fatal(err)
	}
	if gotStatus != "active" || acc.Status != "active" {
		t.Fatalf("status=%q want active", gotStatus)
	}
	if acc.Currency != "USD" {
		t.Fatalf("currency=%q want USD", acc.Currency)
	}
}

func TestGet_NotFound(t *testing.T) {
	t.Parallel()
	svc := service.New(&fakeStore{
		getFn: func(context.Context, uuid.UUID) (store.Account, error) {
			return store.Account{}, pgx.ErrNoRows
		},
	})
	_, err := svc.Get(context.Background(), uuid.New())
	if !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

func TestUpdateStatus_Conflict(t *testing.T) {
	t.Parallel()
	svc := service.New(&fakeStore{
		updateFn: func(context.Context, store.UpdateAccountStatusParams) (store.Account, error) {
			return store.Account{}, pgx.ErrNoRows
		},
	})
	_, err := svc.UpdateStatus(context.Background(), service.UpdateStatusInput{
		ID: uuid.New(), Status: "suspended", Version: 1,
	})
	if !errors.Is(err, service.ErrConflict) {
		t.Fatalf("got %v, want ErrConflict", err)
	}
}
