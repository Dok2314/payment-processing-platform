package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func newTestPostgresStore(t *testing.T) *PostgresPaymentStore {
	t.Helper()

	_ = godotenv.Load()

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
	id := uuid.NewString()

	_, err := store.Get(ctx, id)
	if !errors.Is(err, ErrPaymentNotFound) {
		t.Fatalf("error %v", err)
	}
}

func TestPostgresCreateAndGet(t *testing.T) {
	store := newTestPostgresStore(t)

	ctx := context.Background()

	key := fmt.Sprintf("order-%d", time.Now().UnixNano())

	req := PaymentRequest{
		UserID:   "user-123",
		Amount:   "100.50",
		Currency: "USD",
	}

	payment := Payment{
		ID:       uuid.NewString(),
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

	if got.ID != payment.ID {
		t.Fatalf("ID: got %q, want %q", got.ID, payment.ID)
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
	key := fmt.Sprintf("idempotency-test-%d", time.Now().UnixNano())

	req := PaymentRequest{
		UserID:   "user-123",
		Amount:   "100.50",
		Currency: "USD",
	}

	payment := Payment{
		ID:       uuid.NewString(),
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
	payment2.ID = uuid.NewString()

	second, err := store.CreateIdempotent(ctx, key, payment2, hash)
	if err != nil {
		t.Fatalf("second create: %v", err)
	}

	if first.ID != second.ID {
		t.Fatalf("ID mismatch: first got %q, second got %q", first.ID, second.ID)
	}

	var count int

	err = store.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM payments
		WHERE id = $1
	`, first.ID).Scan(&count)

	if err != nil {
		t.Fatalf("count payments: %v", err)
	}

	if count != 1 {
		t.Fatalf("payment count: got %d, want 1", count)
	}

	_, err = store.CreateIdempotent(ctx, key, payment2, "different-hash")
	if err == nil {
		t.Fatal("expected error for reused idempotency key with different payload")
	}
}

func TestPostgresCreateIdempotentConcurrent(t *testing.T) {
	store := newTestPostgresStore(t)
	ctx := context.Background()
	key := fmt.Sprintf("test-key-%d", time.Now().UnixNano())

	req := PaymentRequest{
		UserID:   "user-123",
		Amount:   "100.50",
		Currency: "USD",
	}

	amount, err := ParseAmount(req.Amount, req.Currency)
	if err != nil {
		t.Fatalf("parse amount: %v", err)
	}

	payment := Payment{
		ID:       uuid.NewString(),
		UserID:   req.UserID,
		Amount:   amount,
		Currency: req.Currency,
		Status:   PaymentStatusPending,
	}

	hash, err := payloadHash(req)
	if err != nil {
		t.Fatalf("error hashing: %v", err)
	}

	const workers = 5

	start := make(chan struct{})
	var wg sync.WaitGroup

	results := make(chan Payment, workers)
	errs := make(chan error, workers)

	wg.Add(workers)

	for range workers {
		go func() {
			defer wg.Done()

			<-start

			p, err := store.CreateIdempotent(ctx, key, payment, hash)
			if err != nil {
				errs <- err
				return
			}

			results <- p
		}()
	}

	close(start)
	wg.Wait()
	close(results)
	close(errs)

	for err := range errs {
		t.Fatalf("create idempotent: %v", err)
	}

	for range workers {
		got := <-results

		if got.ID != payment.ID {
			t.Fatalf("ID: got %q, want %q", got.ID, payment.ID)
		}
	}

	var count int
	err = store.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM payments
		WHERE id = $1
	`, payment.ID).Scan(&count)

	if err != nil {
		t.Fatalf("count payments: %v", err)
	}

	if count != 1 {
		t.Fatalf("payment count: got %d, want 1", count)
	}
}

func TestPostgresCreateIdempotentConcurrentDifferentPayload(t *testing.T) {
	store := newTestPostgresStore(t)
	ctx := context.Background()

	key := fmt.Sprintf("idempotency-test-%d", time.Now().UnixNano())

	requests := []PaymentRequest{
		{UserID: "user-123", Amount: "100.50", Currency: "USD"},
		{UserID: "user-456", Amount: "200.50", Currency: "EUR"},
		{UserID: "user-789", Amount: "300.50", Currency: "USD"},
		{UserID: "user-987", Amount: "400.50", Currency: "EUR"},
		{UserID: "user-10123", Amount: "500.50", Currency: "USD"},
	}

	payments := make([]Payment, len(requests))
	hashes := make([]string, len(requests))

	for i, req := range requests {
		amount, err := ParseAmount(req.Amount, req.Currency)
		if err != nil {
			t.Fatalf("parse amount %d: %v", i, err)
		}

		hash, err := payloadHash(req)
		if err != nil {
			t.Fatalf("payload hash %d: %v", i, err)
		}

		payments[i] = Payment{
			ID:       uuid.NewString(),
			UserID:   req.UserID,
			Amount:   amount,
			Currency: req.Currency,
			Status:   PaymentStatusPending,
		}

		hashes[i] = hash
	}

	start := make(chan struct{})
	errs := make(chan error, len(payments))

	var wg sync.WaitGroup

	for i := range payments {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			<-start

			_, err := store.CreateIdempotent(
				ctx,
				key,
				payments[i],
				hashes[i],
			)

			errs <- err
		}(i)
	}

	close(start)
	wg.Wait()
	close(errs)

	successes := 0
	failures := 0

	for err := range errs {
		if err != nil {
			failures++
		} else {
			successes++
		}
	}

	if successes != 1 {
		t.Fatalf("successes: got %d, want 1", successes)
	}

	if failures != len(payments)-1 {
		t.Fatalf("failures: got %d, want %d", failures, len(payments)-1)
	}

	var count int

	err := store.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM payments
		WHERE id = ANY($1)
	`, []string{
		payments[0].ID,
		payments[1].ID,
		payments[2].ID,
		payments[3].ID,
		payments[4].ID,
	}).Scan(&count)

	if err != nil {
		t.Fatalf("count payments: %v", err)
	}

	if count != 1 {
		t.Fatalf("payment count: got %d, want 1", count)
	}
}

func TestMoveToProcessing(t *testing.T) {
	store := newTestPostgresStore(t)
	ctx := context.Background()

	key := fmt.Sprintf("move-processing-%d", time.Now().UnixNano())

	req := PaymentRequest{
		UserID:   "test-user",
		Amount:   "10.00",
		Currency: "USD",
	}

	amount, err := ParseAmount(req.Amount, req.Currency)
	if err != nil {
		t.Fatalf("parse amount: %v", err)
	}

	hash, err := payloadHash(req)
	if err != nil {
		t.Fatalf("payload hash: %v", err)
	}

	payment := Payment{
		ID:       uuid.NewString(),
		UserID:   req.UserID,
		Amount:   amount,
		Currency: req.Currency,
		Status:   PaymentStatusPending,
	}

	_, err = store.CreateIdempotent(ctx, key, payment, hash)
	if err != nil {
		t.Fatalf("create payment: %v", err)
	}

	updatedPayment, err := store.MoveToProcessing(ctx, payment.ID)
	if err != nil {
		t.Fatalf("move to processing: %v", err)
	}

	if updatedPayment.Status != PaymentStatusProcessing {
		t.Fatalf(
			"status: got %q, want %q",
			updatedPayment.Status,
			PaymentStatusProcessing,
		)
	}

	_, err = store.MoveToProcessing(ctx, payment.ID)

	if !errors.Is(err, ErrPaymentCannotBeProcessed) {
		t.Fatalf("expected ErrPaymentCannotBeProcessed, got %v", err)
	}

	_, err = store.MoveToProcessing(ctx, uuid.NewString())

	if !errors.Is(err, ErrPaymentNotFound) {
		t.Fatalf("expected ErrPaymentNotFound, got %v", err)
	}
}

func TestMoveToProcessingConcurrent(t *testing.T) {
	store := newTestPostgresStore(t)
	ctx := context.Background()

	key := fmt.Sprintf("move-processing-concurrent-%d", time.Now().UnixNano())

	req := PaymentRequest{
		UserID:   "test-user",
		Amount:   "100.00",
		Currency: "USD",
	}

	amount, err := ParseAmount(req.Amount, req.Currency)
	if err != nil {
		t.Fatalf("parse amount: %v", err)
	}

	hash, err := payloadHash(req)
	if err != nil {
		t.Fatalf("payload hash: %v", err)
	}

	payment := Payment{
		ID:       uuid.NewString(),
		UserID:   req.UserID,
		Amount:   amount,
		Currency: req.Currency,
		Status:   PaymentStatusPending,
	}

	_, err = store.CreateIdempotent(ctx, key, payment, hash)
	if err != nil {
		t.Fatalf("create payment: %v", err)
	}

	start := make(chan struct{})
	results := make(chan error, 2)

	var wg sync.WaitGroup

	for i := 0; i < 2; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			<-start

			_, err := store.MoveToProcessing(ctx, payment.ID)
			results <- err
		}()
	}

	close(start)
	wg.Wait()
	close(results)

	successCount := 0
	rejectedCount := 0

	for err := range results {
		if err == nil {
			successCount++
		} else if errors.Is(err, ErrPaymentCannotBeProcessed) {
			rejectedCount++
		} else {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if successCount != 1 || rejectedCount != 1 {
		t.Fatalf("expected 1 success and 1 rejection, got %d success and %d rejections",
			successCount, rejectedCount)
	}
}

func TestPostgresComplete(t *testing.T) {
	store := newTestPostgresStore(t)
	ctx := context.Background()

	paymentID := uuid.NewString()

	payment := Payment{
		ID:       paymentID,
		UserID:   uuid.NewString(),
		Amount:   1000,
		Currency: "USD",
		Status:   PaymentStatusProcessing,
	}

	key := uuid.NewString()
	payloadHash := "test-payload-hash"

	_, err := store.CreateIdempotent(ctx, key, payment, payloadHash)
	if err != nil {
		t.Fatalf("create payment: %v", err)
	}

	updated, err := store.Complete(ctx, paymentID)
	if err != nil {
		t.Fatalf("complete payment: %v", err)
	}

	if updated.Status != PaymentStatusCompleted {
		t.Fatalf("status: got %q, want %q",
			updated.Status, PaymentStatusCompleted)
	}

	_, err = store.Complete(ctx, paymentID)

	if !errors.Is(err, ErrPaymentCannotBeCompleted) {
		t.Fatalf("expected ErrPaymentCannotBeCompleted, got %v", err)
	}

	_, err = store.Complete(ctx, uuid.NewString())

	if !errors.Is(err, ErrPaymentNotFound) {
		t.Fatalf("expected ErrPaymentNotFound, got %v", err)
	}
}

func TestPostgresFail(t *testing.T) {
	store := newTestPostgresStore(t)
	ctx := context.Background()

	payment := Payment{
		ID:       uuid.NewString(),
		UserID:   uuid.NewString(),
		Amount:   1000,
		Currency: "USD",
		Status:   PaymentStatusProcessing,
	}

	_, err := store.CreateIdempotent(
		ctx,
		uuid.NewString(),
		payment,
		"test-payload-hash",
	)
	if err != nil {
		t.Fatalf("create payment: %v", err)
	}

	updated, err := store.Fail(ctx, payment.ID)
	if err != nil {
		t.Fatalf("fail payment: %v", err)
	}

	if updated.Status != PaymentStatusFailed {
		t.Fatalf("status: got %q, want %q",
			updated.Status, PaymentStatusFailed)
	}

	_, err = store.Fail(ctx, payment.ID)
	if !errors.Is(err, ErrPaymentCannotBeFailed) {
		t.Fatalf("expected ErrPaymentCannotBeFailed, got %v", err)
	}

	_, err = store.Fail(ctx, uuid.NewString())
	if !errors.Is(err, ErrPaymentNotFound) {
		t.Fatalf("expected ErrPaymentNotFound, got %v", err)
	}
}

func TestPostgresCompleteConcurrent(t *testing.T) {
	store := newTestPostgresStore(t)
	ctx := context.Background()

	payment := Payment{
		ID:       uuid.NewString(),
		UserID:   uuid.NewString(),
		Amount:   1000,
		Currency: "USD",
		Status:   PaymentStatusProcessing,
	}

	_, err := store.CreateIdempotent(
		ctx,
		uuid.NewString(),
		payment,
		"test-payload-hash",
	)
	if err != nil {
		t.Fatalf("create payment: %v", err)
	}

	start := make(chan struct{})
	results := make(chan error, 2)

	var wg sync.WaitGroup

	for i := 0; i < 2; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			<-start

			_, err := store.Complete(ctx, payment.ID)
			results <- err
		}()
	}

	close(start)
	wg.Wait()
	close(results)

	successes := 0
	rejections := 0

	for err := range results {
		if err == nil {
			successes++
		} else if errors.Is(err, ErrPaymentCannotBeCompleted) {
			rejections++
		} else {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if successes != 1 || rejections != 1 {
		t.Fatalf("expected 1 success and 1 rejection, got %d and %d",
			successes, rejections)
	}
}

func TestPostgresFailConcurrent(t *testing.T) {
	store := newTestPostgresStore(t)
	ctx := context.Background()

	payment := Payment{
		ID:       uuid.NewString(),
		UserID:   uuid.NewString(),
		Amount:   1000,
		Currency: "USD",
		Status:   PaymentStatusProcessing,
	}

	_, err := store.CreateIdempotent(
		ctx,
		uuid.NewString(),
		payment,
		"test-payload-hash",
	)
	if err != nil {
		t.Fatalf("create payment: %v", err)
	}

	start := make(chan struct{})
	results := make(chan error, 2)

	var wg sync.WaitGroup

	for i := 0; i < 2; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			<-start

			_, err := store.Fail(ctx, payment.ID)
			results <- err
		}()
	}

	close(start)
	wg.Wait()
	close(results)

	successes := 0
	rejections := 0

	for err := range results {
		if err == nil {
			successes++
		} else if errors.Is(err, ErrPaymentCannotBeFailed) {
			rejections++
		} else {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if successes != 1 || rejections != 1 {
		t.Fatalf("expected 1 success and 1 rejection, got %d and %d",
			successes, rejections)
	}
}
