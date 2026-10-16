package service_test

import (
	"context"
	"os"
	"testing"

	"github.com/chuuch/ledger-core/internal/ledger/service"
	"github.com/chuuch/ledger-core/internal/platform/postgres"
	"github.com/google/uuid"
)

func TestPostJournal_Integration(t *testing.T) {
	dsn := os.Getenv("LEDGER_DATABASE_URL")
	if dsn == "" {
		t.Skip("LEDGER_DATABASE_URL not set")
	}

	ctx := context.Background()
	pool, err := postgres.NewPool(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	qstore := service.NewQueriesStore(pool)
	svc := service.New(qstore, &service.PoolTxRunner{Pool: pool})

	a1, a2 := uuid.New(), uuid.New()
	view, err := svc.PostJournal(ctx, service.PostJournalInput{
		Description: "integration",
		Currency:    "EUR",
		Postings: []service.PostingInput{
			{AccountID: a1, Direction: "debit", Amount: 250},
			{AccountID: a2, Direction: "credit", Amount: 250},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.GetJournal(ctx, view.Entry.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Postings) != 2 {
		t.Fatalf("postings=%d", len(got.Postings))
	}
}
