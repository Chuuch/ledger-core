package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/chuuch/ledger-core/internal/ledger/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrNotFound = errors.New("not found")
	ErrInvalidInput = errors.New("invalid input")
	ErrUnbalanced = errors.New("journal entry is unbalanced")
	ErrConflict = errors.New("balance version conflict")
)

type Service struct {
	store Store
	tx TxRunner
}

func New(store Store, tx TxRunner) *Service {
	return &Service{store: store, tx: tx}
}

type PostingInput struct {
	AccountID uuid.UUID
	Direction string
	Amount int64
}

type PostJournalInput struct {
	Reference *string
	Description string
	Currency string
	Postings []PostingInput
}

type JournalView struct {
	Entry store.JournalEntry `json:"entry"`
	Postings []store.Posting `json:"postings"`
}

func (s *Service) PostJournal(ctx context.Context, in PostJournalInput) (JournalView, error) {
	in.Description = strings.TrimSpace(in.Description)
	in.Currency = strings.ToUpper(strings.TrimSpace(in.Currency))
	if in.Description == "" || len(in.Currency) != 3 {
		return JournalView{}, fmt.Errorf("%w: description and currency (ISO-4217) required", ErrInvalidInput)
	}
	if len(in.Postings) < 2 {
		return JournalView{}, fmt.Errorf("%w: at least two postings required", ErrInvalidInput)
	}

	var debits, credits int64
	for i, p := range in.Postings {
		dir := strings.ToLower(strings.TrimSpace(p.Direction))
		if dir != "debit" && dir != "credit" {
			return JournalView{}, fmt.Errorf("%w: posting[%d] invalid direction", ErrInvalidInput, i)
		}
		if p.Amount <= 0 {
			return JournalView{}, fmt.Errorf("%w: posting[%d] amount must be > 0", ErrInvalidInput, i)
		}
		if p.AccountID == uuid.Nil {
			return JournalView{}, fmt.Errorf("%w: posting[%d] account_id required", ErrInvalidInput, i)
		}
		in.Postings[i].Direction = dir
		switch dir {
		case "debit":
			debits += p.Amount
		case "credit":
			credits += p.Amount
		}
	}
	if debits != credits {
		return JournalView{}, fmt.Errorf("%w: debits=%d credits=%d", ErrUnbalanced, debits, credits)
	}

	var out JournalView
	err := s.tx.WithinTx(ctx, func(st Store) error {
		var ref *string
		if in.Reference != nil {
			r := strings.TrimSpace(*in.Reference)
			if r != "" {
				ref = &r
			}
		}

		entry, err := st.CreateJournalEntry(ctx, store.CreateJournalEntryParams{
			Reference: ref,
			Description: in.Description,
			Status: "posted",
		})
		if err != nil {
			return err
		}

		posted := make([]store.Posting, 0, len(in.Postings))
		for _, p := range in.Postings {
			row, err := st.CreatePosting(ctx, store.CreatePostingParams{
				JournalEntryID: entry.ID,
				AccountID: p.AccountID,
				Direction: p.Direction,
				Amount: p.Amount,
				Currency: in.Currency,
			})
			if err != nil {
				return err
			}
			posted = append(posted, row)

			if err := applyBalance(ctx, st, p.AccountID, in.Currency, p.Direction, p.Amount); err != nil {
				return err
			}
		}

		out = JournalView{Entry: entry, Postings: posted}
		return nil
	})
	if err != nil {
		return JournalView{}, err
	}
	return out, nil
}

func applyBalance(ctx context.Context, st Store, accountID uuid.UUID, currency, direction string, amount int64) error {
	bal, err := st.GetAccountBalance(ctx, store.GetAccountBalanceParams{
		AccountID: accountID,
		Currency: currency,
	})
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		initial := amount
		if direction == "credit" {
			initial = -amount
		}
		_, err := st.InsertAccountBalance(ctx, store.InsertAccountBalanceParams{
			AccountID: accountID,
			Currency: currency,
			Balance: initial,
		})
		return err
	}

	next := bal.Balance + amount
	if direction == "credit" {
		next = bal.Balance - amount
	}
	_, err = st.UpdateAccountBalance(ctx, store.UpdateAccountBalanceParams{
		AccountID: accountID,
		Currency: currency,
		Balance: next,
		Version: bal.Version,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrConflict
		}
		return err
	}
	return nil
}

func (s *Service) GetJournal(ctx context.Context, id uuid.UUID) (JournalView, error) {
	entry, err := s.store.GetJournalEntryByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return JournalView{}, ErrNotFound
		}
		return JournalView{}, err
	}
	posts, err := s.store.ListPostingsByJournalEntryID(ctx, id)
	if err != nil {
		return JournalView{}, err
	}
	return JournalView{Entry: entry, Postings: posts}, nil
}

func (s *Service) GetBalance(ctx context.Context, accountID uuid.UUID, currency string) (store.AccountBalance, error) {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if accountID == uuid.Nil || len(currency) != 3 {
		return store.AccountBalance{}, fmt.Errorf("%w: account_id and currency required", ErrInvalidInput)
	}
	bal, err := s.store.GetAccountBalance(ctx, store.GetAccountBalanceParams{
		AccountID: accountID,
		Currency: currency,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return store.AccountBalance{}, ErrNotFound
		}
		return store.AccountBalance{}, err
	}
	return bal, nil
}
