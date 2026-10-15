package service_test

import (
	"context"
	"os"
	"testing"

	"github.com/chuuch/ledger-core/internal/account/service"
	"github.com/chuuch/ledger-core/internal/account/store"
	"github.com/chuuch/ledger-core/internal/platform/postgres"
)

func TestCreateGet_Integration(t *testing.T) {
	dsn := os.Getenv("ACCOUNT_DATABASE_URL")
	if dsn == "" {
		t.Skip("ACCOUNT_DATABASE_URL not set")
	}

	ctx := context.Background()
	pool, err := postgres.NewPool(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	svc := service.New(store.New(pool))
	acc, err := svc.Create(ctx, service.CreateInput{OwnerID: "itest-owner", Currency: "EUR"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.Get(ctx, acc.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != acc.ID || got.Currency != "EUR" {
		t.Fatalf("got %+v", got)
	}
}
