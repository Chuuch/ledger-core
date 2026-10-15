// Package service
package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/chuuch/ledger-core/internal/account/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrNotFound = errors.New("account not found")
	ErrConflict = errors.New("account version conflict")
	ErrInvalidInput = errors.New("invalid input")
	ErrInvalidStatus = errors.New("invalid status")
)

type Service struct {
	q AccountStore
}

func New(q AccountStore) *Service {
	return &Service{q: q}
}

type CreateInput struct {
	OwnerID string
	Currency string
	Status string
}

func (s *Service) Create(ctx context.Context, in CreateInput) (store.Account, error) {
	in.OwnerID = strings.TrimSpace(in.OwnerID)
	in.Currency = strings.ToUpper(strings.TrimSpace(in.Currency))
	if in.Status == "" {
		in.Status = "active"
	}
	if in.OwnerID == "" || len(in.Currency) != 3 {
		return store.Account{}, fmt.Errorf("%w: owner_id and currency (ISO-4217) required", ErrInvalidInput)
	}
	if err := validateStatus(in.Status); err != nil {
		return store.Account{}, err
	}

	return s.q.CreateAccount(ctx, store.CreateAccountParams{
		OwnerID: in.OwnerID,
		Currency: in.Currency,
		Status: in.Status,
	})
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (store.Account, error) {
	acc, err := s.q.GetAccountByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return store.Account{}, ErrNotFound
		}
		return store.Account{}, err
	}
	return acc, nil
}

func (s *Service) ListByOwner(ctx context.Context, ownerID string) ([]store.Account, error) {
	ownerID = strings.TrimSpace(ownerID)
	if ownerID == "" {
		return nil, fmt.Errorf("%w: owner_id required", ErrInvalidInput)
	}
	return s.q.ListAccountsByOwner(ctx, ownerID)
}

type UpdateStatusInput struct {
	ID uuid.UUID
	Status string
	Version int64
}

func (s *Service) UpdateStatus(ctx context.Context, in UpdateStatusInput) (store.Account, error) {
	if err := validateStatus(in.Status); err != nil {
		return store.Account{}, err
	}
	if in.Version < 1 {
		return store.Account{}, fmt.Errorf("%w: version must be >= 1", ErrInvalidInput)
	}

	acc, err := s.q.UpdateAccountStatus(ctx, store.UpdateAccountStatusParams{
		ID: in.ID,
		Status: in.Status,
		Version: in.Version,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return store.Account{}, ErrConflict
		}
		return store.Account{}, err
	}
	return acc, nil
}

func validateStatus(status string) error {
	switch status {
	case "active", "suspended", "closed":
		return nil
	default:
		return fmt.Errorf("%w: %s", ErrInvalidStatus, status)
	}
}
