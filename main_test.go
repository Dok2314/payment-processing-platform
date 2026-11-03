package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/health", nil)

	healthHandler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Body.String() != "ok" {
		t.Fatalf("expected body %q, got %q", "ok", w.Body.String())
	}
}

func TestVersionHandler(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/version", nil)

	versionHandler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Body.String() != "1.0.0" {
		t.Fatalf("expected body %q, got %q", "1.0.0", w.Body.String())
	}
}

func TestRouter(t *testing.T) {
	store := NewPaymentStore()
	service := NewPaymentService(store)
	r := newRouter(service)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/health", nil)

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Body.String() != "ok" {
		t.Fatalf("expected body %q, got %q", "ok", w.Body.String())
	}

	req = httptest.NewRequest("GET", "/unknown", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestUpdatePaymentRouteNotFound(t *testing.T) {
	store := NewPaymentStore()
	service := NewPaymentService(store)
	r := newRouter(service)

	req := httptest.NewRequest("PUT", "/payments/payment-123", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestCreatePaymentInvalidAmount(t *testing.T) {
	body := strings.NewReader(`{"user_id":"123","amount":-100,"currency":"USD"}`)
	req := httptest.NewRequest("POST", "/payments", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "test-invalid-amount")

	w := httptest.NewRecorder()
	store := NewPaymentStore()
	service := NewPaymentService(store)
	r := newRouter(service)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestCreatePaymentInvalidJSON(t *testing.T) {
	body := strings.NewReader(`{"user_id":"123","amount":-100,"currency":here invalid`)
	req := httptest.NewRequest("POST", "/payments", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "test-invalid-json")

	w := httptest.NewRecorder()
	store := NewPaymentStore()
	service := NewPaymentService(store)
	r := newRouter(service)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestCreatePaymentMissingUserID(t *testing.T) {
	body := strings.NewReader(`{"user_id":"","amount":"100.00","currency":"USD"}`)
	req := httptest.NewRequest("POST", "/payments", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "test-missing-user")

	w := httptest.NewRecorder()
	store := NewPaymentStore()
	service := NewPaymentService(store)
	r := newRouter(service)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestCreatePaymentMissingCurrency(t *testing.T) {
	body := strings.NewReader(`{"user_id":"123","amount":"100.50","currency":""}`)
	req := httptest.NewRequest("POST", "/payments", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "test-missing-currency")

	w := httptest.NewRecorder()
	store := NewPaymentStore()
	service := NewPaymentService(store)
	r := newRouter(service)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestCreatePaymentSuccess(t *testing.T) {
	body := strings.NewReader(`{"user_id":"123","amount":"100.50","currency":"USD"}`)
	req := httptest.NewRequest("POST", "/payments", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "test-order-123")

	w := httptest.NewRecorder()
	store := NewPaymentStore()
	service := NewPaymentService(store)
	r := newRouter(service)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var payment Payment

	if err := json.NewDecoder(w.Body).Decode(&payment); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if payment.ID == "" {
		t.Fatal("expected payment ID to be non-empty")
	}

	if payment.UserID != "123" {
		t.Fatalf("expected user_id %q, got %q", "123", payment.UserID)
	}

	if payment.Amount != 10050 {
		t.Fatalf("expected amount %d, got %d", 10050, payment.Amount)
	}

	if payment.Currency != "USD" {
		t.Fatalf("expected currency %q, got %q", "USD", payment.Currency)
	}
}

func TestPaymentStoreGet(t *testing.T) {
	store := NewPaymentStore()

	payment := Payment{
		ID:       "payment-123",
		UserID:   "user-123",
		Amount:   10050,
		Currency: "USD",
	}

	store.Save(payment)

	got, err := store.Get(context.Background(), "payment-123")
	if err != nil {
		t.Fatalf("expected payment to exist: %v", err)
	}

	if got != payment {
		t.Fatalf("expected %+v, got %+v", payment, got)
	}
}

func TestGetPaymentNotFound(t *testing.T) {
	store := NewPaymentStore()
	service := NewPaymentService(store)
	r := newRouter(service)

	req := httptest.NewRequest("GET", "/payments/nonexistent", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestGetPaymentSuccess(t *testing.T) {
	store := NewPaymentStore()
	service := NewPaymentService(store)

	payment := Payment{
		ID:       "payment-123",
		UserID:   "user-123",
		Amount:   10050,
		Currency: "USD",
	}
	store.Save(payment)

	r := newRouter(service)
	req := httptest.NewRequest("GET", "/payments/payment-123", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var got Payment
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if got != payment {
		t.Fatalf("expected %+v, got %+v", payment, got)
	}
}

func TestCancelPaymentNotFound(t *testing.T) {
	store := NewPaymentStore()
	service := NewPaymentService(store)
	r := newRouter(service)

	req := httptest.NewRequest("POST", "/payments/nonexistent/cancel", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestCreatePaymentInvalidAmountType(t *testing.T) {
	body := strings.NewReader(`{"user_id":"123","amount":"ABC","currency":"USD"}`)
	req := httptest.NewRequest("POST", "/payments", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "test-invalid-amount-type")

	w := httptest.NewRecorder()
	store := NewPaymentStore()
	service := NewPaymentService(store)
	r := newRouter(service)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestCreateIdempotent(t *testing.T) {
	store := NewPaymentStore()

	first := Payment{
		ID:       "payment-1",
		UserID:   "123",
		Amount:   10050,
		Currency: "USD",
	}

	fp, err := store.CreateIdempotent(context.Background(), "order-123", first, "test-hash")
	if err != nil {
		t.Fatalf("first CreateIdempotent() error: %v", err)
	}

	second := Payment{
		ID:       "payment-2",
		UserID:   "123",
		Amount:   10050,
		Currency: "USD",
	}

	sp, err := store.CreateIdempotent(context.Background(), "order-123", second, "test-hash")
	if err != nil {
		t.Fatalf("second CreateIdempotent() error: %v", err)
	}

	if fp.ID != sp.ID {
		t.Fatalf("payment IDs differ: first=%q, second=%q", fp.ID, sp.ID)
	}

	if sp.Amount != 10050 {
		t.Fatalf("unexpected amount: got=%v, want=10050", sp.Amount)
	}

	if sp.ID != "payment-1" {
		t.Fatalf("unexpected payment ID: got=%q, want=%q", sp.ID, "payment-1")
	}
}

func TestCreateIdempotentConcurrent(t *testing.T) {
	store := NewPaymentStore()
	var wg sync.WaitGroup
	results := make(chan string, 100)

	wg.Add(100)

	for i := 0; i < 100; i++ {
		go func(i int) {
			defer wg.Done()

			payment := Payment{
				ID:       fmt.Sprintf("payment-%d", i),
				UserID:   "123",
				Amount:   10050,
				Currency: "USD",
			}

			p, err := store.CreateIdempotent(context.Background(), "order-123", payment, "test-hash")
			if err != nil {
				t.Errorf("CreateIdempotent: %v", err)
				return
			}

			results <- p.ID
		}(i)
	}

	wg.Wait()
	close(results)

	if len(store.idempotencyKeys) != 1 {
		t.Fatalf("expected 1 idempotency key, got %d", len(store.idempotencyKeys))
	}

	if len(store.payments) != 1 {
		t.Fatalf("expected 1 payment, got %d", len(store.payments))
	}

	var firstID string

	for id := range results {
		if firstID == "" {
			firstID = id
			continue
		}

		if id != firstID {
			t.Fatalf("expected same payment ID, got %s and %s", firstID, id)
		}
	}
}

func TestCancelPaymentSuccess(t *testing.T) {
	store := NewPaymentStore()
	service := NewPaymentService(store)
	ctx := context.Background()

	created, err := service.Create(ctx, "key-1", PaymentRequest{
		UserID:   "user-1",
		Amount:   "1.00",
		Currency: "USD",
	})
	if err != nil {
		t.Fatal(err)
	}

	cancelled, err := service.Cancel(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}

	if cancelled.Status != PaymentStatusCancelled {
		t.Errorf("expected status %q, got %q",
			PaymentStatusCancelled, cancelled.Status)
	}

	saved, err := service.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("expected cancelled payment to remain in store: %v", err)
	}

	if saved.Status != PaymentStatusCancelled {
		t.Errorf("expected stored status %q, got %q",
			PaymentStatusCancelled, saved.Status)
	}
}

func TestCancelPaymentAlreadyCancelled(t *testing.T) {
	ctx := context.Background()
	store := NewPaymentStore()
	service := NewPaymentService(store)

	created, err := service.Create(ctx, "key-1", PaymentRequest{
		UserID:   "user-1",
		Amount:   "1.00",
		Currency: "USD",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = service.Cancel(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}

	_, err = service.Cancel(ctx, created.ID)
	if err == nil {
		t.Fatal("expected cancellation of an already cancelled payment to fail")
	}
}

func TestPaymentStoreMoveToProcessing(t *testing.T) {
	store := NewPaymentStore()
	ctx := context.Background()

	payment := Payment{
		ID:     "payment-process-1",
		Status: PaymentStatusPending,
	}

	store.Save(payment)

	updated, err := store.MoveToProcessing(ctx, payment.ID)
	if err != nil {
		t.Fatalf("move to processing: %v", err)
	}

	if updated.Status != PaymentStatusProcessing {
		t.Fatalf("status: got %q, want %q",
			updated.Status, PaymentStatusProcessing)
	}

	_, err = store.MoveToProcessing(ctx, payment.ID)

	if !errors.Is(err, ErrPaymentCannotBeProcessed) {
		t.Fatalf("expected ErrPaymentCannotBeProcessed, got %v", err)
	}

	_, err = store.MoveToProcessing(ctx, "missing-payment")

	if !errors.Is(err, ErrPaymentNotFound) {
		t.Fatalf("expected ErrPaymentNotFound, got %v", err)
	}
}

func TestPaymentStoreComplete(t *testing.T) {
	store := NewPaymentStore()
	ctx := context.Background()

	payment := Payment{
		ID:     "payment-process-1",
		Status: PaymentStatusProcessing,
	}

	store.Save(payment)

	updated, err := store.Complete(ctx, payment.ID)
	if err != nil {
		t.Fatalf("complete payment: %v", err)
	}

	if updated.Status != PaymentStatusCompleted {
		t.Fatalf("status: got %q, want %q",
			updated.Status, PaymentStatusCompleted)
	}

	_, err = store.Complete(ctx, payment.ID)

	if !errors.Is(err, ErrPaymentCannotBeCompleted) {
		t.Fatalf("expected ErrPaymentCannotBeCompleted, got %v", err)
	}

	_, err = store.Complete(ctx, "missing-payment")

	if !errors.Is(err, ErrPaymentNotFound) {
		t.Fatalf("expected ErrPaymentNotFound, got %v", err)
	}
}

func TestPaymentStoreFail(t *testing.T) {
	store := NewPaymentStore()
	ctx := context.Background()

	payment := Payment{
		ID:     "payment-fail-1",
		Status: PaymentStatusProcessing,
	}

	store.Save(payment)

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

	_, err = store.Fail(ctx, "missing-payment")
	if !errors.Is(err, ErrPaymentNotFound) {
		t.Fatalf("expected ErrPaymentNotFound, got %v", err)
	}
}

func TestPaymentStoreMoveToProcessingConcurrent(t *testing.T) {
	store := NewPaymentStore()
	ctx := context.Background()

	payment := Payment{
		ID:     "payment-process-concurrent",
		Status: PaymentStatusPending,
	}

	store.Save(payment)

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

	successes := 0
	rejections := 0

	for err := range results {
		if err == nil {
			successes++
		} else if errors.Is(err, ErrPaymentCannotBeProcessed) {
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

func TestPaymentStoreCompleteConcurrent(t *testing.T) {
	store := NewPaymentStore()
	ctx := context.Background()

	payment := Payment{
		ID:     "payment-complete-concurrent",
		Status: PaymentStatusProcessing,
	}

	store.Save(payment)

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

func TestPaymentStoreFailConcurrent(t *testing.T) {
	store := NewPaymentStore()
	ctx := context.Background()

	payment := Payment{
		ID:     "payment-fail-concurrent",
		Status: PaymentStatusProcessing,
	}

	store.Save(payment)

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
