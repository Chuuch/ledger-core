package http_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jsonv2 "encoding/json/v2"

	"github.com/chuuch/ledger-core/internal/account/service"
	"github.com/chuuch/ledger-core/internal/account/store"
	accounthttp "github.com/chuuch/ledger-core/internal/account/transport/http"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// minimal fake (same shape as service_test; duplicate is OK for now)
type fakeStore struct {
	accounts map[uuid.UUID]store.Account
}

func newFakeStore() *fakeStore {
	return &fakeStore{accounts: map[uuid.UUID]store.Account{}}
}

func (f *fakeStore) CreateAccount(_ context.Context, arg store.CreateAccountParams) (store.Account, error) {
	id := uuid.New()
	acc := store.Account{
		ID:        id,
		OwnerID:   arg.OwnerID,
		Currency:  arg.Currency,
		Status:    arg.Status,
		Version:   1,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	f.accounts[id] = acc
	return acc, nil
}

func (f *fakeStore) GetAccountByID(_ context.Context, id uuid.UUID) (store.Account, error) {
	acc, ok := f.accounts[id]
	if !ok {
		return store.Account{}, pgx.ErrNoRows
	}
	return acc, nil
}

func (f *fakeStore) ListAccountsByOwner(_ context.Context, ownerID string) ([]store.Account, error) {
	var out []store.Account
	for _, a := range f.accounts {
		if a.OwnerID == ownerID {
			out = append(out, a)
		}
	}
	return out, nil
}

func (f *fakeStore) UpdateAccountStatus(_ context.Context, arg store.UpdateAccountStatusParams) (store.Account, error) {
	acc, ok := f.accounts[arg.ID]
	if !ok || acc.Version != arg.Version || acc.Status == "closed" {
		return store.Account{}, pgx.ErrNoRows
	}
	acc.Status = arg.Status
	acc.Version++
	acc.UpdatedAt = time.Now().UTC()
	f.accounts[arg.ID] = acc
	return acc, nil
}

func TestCreateAndGetAccount(t *testing.T) {
	t.Parallel()
	svc := service.New(newFakeStore())
	// pool unused by these routes; pass nil only if /readyz isn't hit.
	// Prefer a noop: skip readyz in this test by not calling it.
	r := accounthttp.NewRouter(svc, nil, nil)

	body, _ := jsonv2.Marshal(map[string]string{
		"owner_id": "user-1",
		"currency": "USD",
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/accounts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rr.Code, rr.Body.String())
	}

	var created store.Account
	if err := jsonv2.Unmarshal(rr.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/accounts/"+created.ID.String(), nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestUpdateStatusConflict(t *testing.T) {
	t.Parallel()
	fs := newFakeStore()
	svc := service.New(fs)
	r := accounthttp.NewRouter(svc, nil, nil)

	acc, err := svc.Create(context.Background(), service.CreateInput{OwnerID: "u1", Currency: "USD"})
	if err != nil {
		t.Fatal(err)
	}

	body, _ := jsonv2.Marshal(map[string]any{"status": "suspended", "version": acc.Version})
	req := httptest.NewRequest(http.MethodPatch, "/v1/accounts/"+acc.ID.String()+"/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("first patch status=%d body=%s", rr.Code, rr.Body.String())
	}

	body, _ = jsonv2.Marshal(map[string]any{"status": "active", "version": acc.Version}) // stale
	req = httptest.NewRequest(http.MethodPatch, "/v1/accounts/"+acc.ID.String()+"/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("stale patch status=%d want 409 body=%s", rr.Code, rr.Body.String())
	}
}
