package service

import (
	"context"

	"github.com/chuuch/ledger-core/internal/account/store"
	"github.com/google/uuid"
)

type AccountStore interface {
	CreateAccount(ctx context.Context, arg store.CreateAccountParams) (store.Account, error)
	GetAccountByID(ctx context.Context, id uuid.UUID) (store.Account, error)
	ListAccountsByOwner(ctx context.Context, ownerID string) ([]store.Account, error)
	UpdateAccountStatus(ctx context.Context, arg store.UpdateAccountStatusParams) (store.Account, error)
}
