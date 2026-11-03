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

	if err := godotenv.Load(); err != nil {
		t.Fatalf("load .env: %v", err)
	}

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
	id := fmt.Sprintf("missing-payment-%d", time.Now().UnixNano())

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
		ID:       fmt.Sprintf("payment-%d", time.Now().UnixNano()),
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
		ID:       fmt.Sprintf("payment-test-%d", time.Now().UnixNano()),
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
		ID:       fmt.Sprintf("payment-%d", time.Now().UnixNano()),
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

	var wg sync.WaitGroup
	results := make(chan Payment, workers)
	errs := make(chan error, workers)

	wg.Add(workers)

	for range workers {
		go func() {
			defer wg.Done()

			p, err := store.CreateIdempotent(ctx, key, payment, hash)
			if err != nil {
				errs <- err
				return
			}

			results <- p
		}()
	}

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
	key := fmt.Sprintf("idempotency-test-%d", time.Now().UnixNano())

	req1 := PaymentRequest{
		UserID:   "user-123",
		Amount:   "100.50",
		Currency: "USD",
	}

	req2 := PaymentRequest{
		UserID:   "user-456",
		Amount:   "200.50",
		Currency: "EUR",
	}

	req3 := PaymentRequest{
		UserID:   "user-789",
		Amount:   "300.50",
		Currency: "USD",
	}

	req4 := PaymentRequest{
		UserID:   "user-987",
		Amount:   "400.50",
		Currency: "EUR",
	}

	req5 := PaymentRequest{
		UserID:   "user-10123",
		Amount:   "500.50",
		Currency: "USD",
	}

	hash1, err := payloadHash(req1)
	if err != nil {
		t.Fatalf("hash req1: %v", err)
	}

	hash2, err := payloadHash(req2)
	if err != nil {
		t.Fatalf("hash req2: %v", err)
	}

	hash3, err := payloadHash(req3)
	if err != nil {
		t.Fatalf("hash req3: %v", err)
	}

	hash4, err := payloadHash(req4)
	if err != nil {
		t.Fatalf("hash req4: %v", err)
	}

	hash5, err := payloadHash(req5)
	if err != nil {
		t.Fatalf("hash req5: %v", err)
	}

	store := newTestPostgresStore(t)
	ctx := context.Background()
	testID := fmt.Sprintf("%d", time.Now().UnixNano())

	amount1, err := ParseAmount(req1.Amount, req1.Currency)
	if err != nil {
		t.Fatalf("parse req1 amount: %v", err)
	}

	amount2, err := ParseAmount(req2.Amount, req2.Currency)
	if err != nil {
		t.Fatalf("parse req2 amount: %v", err)
	}

	amount3, err := ParseAmount(req3.Amount, req3.Currency)
	if err != nil {
		t.Fatalf("parse req3 amount: %v", err)
	}

	amount4, err := ParseAmount(req4.Amount, req4.Currency)
	if err != nil {
		t.Fatalf("parse req4 amount: %v", err)
	}

	amount5, err := ParseAmount(req5.Amount, req5.Currency)
	if err != nil {
		t.Fatalf("parse req5 amount: %v", err)
	}

	payments := []Payment{
		{ID: "payment-diff-1-" + testID, UserID: req1.UserID, Amount: amount1, Currency: req1.Currency, Status: PaymentStatusPending},
		{ID: "payment-diff-2-" + testID, UserID: req2.UserID, Amount: amount2, Currency: req2.Currency, Status: PaymentStatusPending},
		{ID: "payment-diff-3-" + testID, UserID: req3.UserID, Amount: amount3, Currency: req3.Currency, Status: PaymentStatusPending},
		{ID: "payment-diff-4-" + testID, UserID: req4.UserID, Amount: amount4, Currency: req4.Currency, Status: PaymentStatusPending},
		{ID: "payment-diff-5-" + testID, UserID: req5.UserID, Amount: amount5, Currency: req5.Currency, Status: PaymentStatusPending},
	}

	hashes := []string{hash1, hash2, hash3, hash4, hash5}

	var wg sync.WaitGroup
	errs := make(chan error, len(payments))

	wg.Add(len(payments))

	for i := range payments {
		go func(i int) {
			defer wg.Done()

			_, err := store.CreateIdempotent(ctx, key, payments[i], hashes[i])
			errs <- err
		}(i)
	}

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

	if failures != 4 {
		t.Fatalf("failures: got %d, want 4", failures)
	}

	var count int

	err = store.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM payments
		WHERE id LIKE 'payment-diff-%-' || $1
	`, testID).Scan(&count)

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
