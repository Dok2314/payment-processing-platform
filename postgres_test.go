package main

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func newTestPostgresStore(t *testing.T) *PostgresPaymentStore {
	t.Helper()

	ctx := context.Background()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("TEST_DATABASE_URL is required")
	}

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("create postgres pool: %v", err)
	}

	err = pool.Ping(ctx)
	if err != nil {
		pool.Close()
		t.Fatalf("ping postgres: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
	})

	return NewPostgresPaymentStore(pool)
}

func TestPostgresGetNotFound(t *testing.T) {
	store := newTestPostgresStore(t)

	ctx := context.Background()
	id := "missing-payment"

	_, err := store.Get(ctx, id)
	if !errors.Is(err, ErrPaymentNotFound) {
		t.Fatalf("error %v", err)
	}
}

func TestPostgresCreateAndGet(t *testing.T) {
	store := newTestPostgresStore(t)

	ctx := context.Background()

	key := "order-123"

	req := PaymentRequest{
		UserID:   "user-123",
		Amount:   10050,
		Currency: "USD",
	}

	payment := Payment{
		ID:       "payment-123",
		UserID:   "user-123",
		Amount:   10050,
		Currency: "USD",
		Status:   PaymentStatusPending,
	}

	hash, err := payloadHash(req)
	if err != nil {
		t.Fatalf("error hashing: %v", err)
	}

	_, err = store.CreateIdempotent(ctx, key, payment, hash)
	if err != nil {
		t.Fatalf("create payment: %v", err)
	}

	got, err := store.Get(ctx, payment.ID)
	if err != nil {
		t.Fatalf("get payment: %v", err)
	}

	if got.ID != "payment-123" {
		t.Fatalf("ID: got %q, want %q", got.ID, "payment-123")
	}

	if got.UserID != "user-123" {
		t.Fatalf("UserID: got %q, want %q", got.UserID, "user-123")
	}

	if got.Amount != 10050 {
		t.Fatalf("Amount: got %d, want %d", got.Amount, 10050)
	}

	if got.Currency != "USD" {
		t.Fatalf("Currency: got %q, want %q", got.Currency, "USD")
	}

	if got.Status != PaymentStatusPending {
		t.Fatalf("Status: got %q, want %q", got.Status, PaymentStatusPending)
	}
}

func TestPostgresCreateIdempotent(t *testing.T) {
	store := newTestPostgresStore(t)

	ctx := context.Background()
	key := "idempotency-test-123"

	req := PaymentRequest{
		UserID:   "user-123",
		Amount:   10050,
		Currency: "USD",
	}

	payment := Payment{
		ID:       "payment-test-123",
		UserID:   "user-123",
		Amount:   10050,
		Currency: "USD",
		Status:   PaymentStatusPending,
	}

	hash, err := payloadHash(req)
	if err != nil {
		t.Fatalf("error hashing: %v", err)
	}

	first, err := store.CreateIdempotent(ctx, key, payment, hash)
	if err != nil {
		t.Fatalf("first create: %v", err)
	}

	payment2 := payment
	payment2.ID = "payment-test-456"

	second, err := store.CreateIdempotent(ctx, key, payment2, hash)
	if err != nil {
		t.Fatalf("second create: %v", err)
	}

	if first.ID != second.ID {
		t.Fatalf("ID mismatch: first got %q, second got %q", first.ID, second.ID)
	}
}
