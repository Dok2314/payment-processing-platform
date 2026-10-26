package main

import (
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

func TestCreatePaymentInvalidAmount(t *testing.T) {
	body := strings.NewReader(`{"user_id":"123","amount":-100,"currency":"USD"}`)
	req := httptest.NewRequest("POST", "/payments", body)
	req.Header.Set("Content-Type", "application/json")

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
	body := strings.NewReader(`{"user_id":"","amount":100,"currency":"USD"}`)
	req := httptest.NewRequest("POST", "/payments", body)
	req.Header.Set("Content-Type", "application/json")

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
	body := strings.NewReader(`{"user_id":"123","amount":100.50,"currency":""}`)
	req := httptest.NewRequest("POST", "/payments", body)
	req.Header.Set("Content-Type", "application/json")

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
	body := strings.NewReader(`{"user_id":"123","amount":100.50,"currency":"USD"}`)
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

	if payment.Amount != 100.50 {
		t.Fatalf("expected amount %v, got %v", 100.50, payment.Amount)
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
		Amount:   100.50,
		Currency: "USD",
	}

	store.Save(payment)

	got, ok := store.Get("payment-123")
	if !ok {
		t.Fatal("expected payment to exist")
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
		Amount:   100.50,
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

func TestUpdatePaymentSuccess(t *testing.T) {
	store := NewPaymentStore()
	service := NewPaymentService(store)

	store.Save(Payment{
		ID:       "payment-123",
		UserID:   "user-123",
		Amount:   100.50,
		Currency: "USD",
	})

	body := strings.NewReader(`{"user_id":"user-456","amount":200,"currency":"EUR"}`)
	req := httptest.NewRequest("PUT", "/payments/payment-123", body)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r := newRouter(service)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var got Payment
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if got.ID != "payment-123" ||
		got.UserID != "user-456" ||
		got.Amount != 200 ||
		got.Currency != "EUR" {
		t.Fatalf("unexpected updated payment: %+v", got)
	}
}

func TestUpdatePaymentNotFound(t *testing.T) {
	store := NewPaymentStore()
	service := NewPaymentService(store)
	r := newRouter(service)

	body := strings.NewReader(`{"user_id":"user-123","amount":100,"currency":"USD"}`)
	req := httptest.NewRequest("PUT", "/payments/nonexistent", body)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestDeletePaymentSuccess(t *testing.T) {
	store := NewPaymentStore()
	service := NewPaymentService(store)
	store.Save(Payment{
		ID:       "payment-123",
		UserID:   "user-123",
		Amount:   100.50,
		Currency: "USD",
	})

	r := newRouter(service)
	req := httptest.NewRequest("DELETE", "/payments/payment-123", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}

	_, ok := store.Get("payment-123")
	if ok {
		t.Fatal("expected payment to be deleted")
	}
}

func TestDeletePaymentNotFound(t *testing.T) {
	store := NewPaymentStore()
	service := NewPaymentService(store)
	r := newRouter(service)

	req := httptest.NewRequest("DELETE", "/payments/nonexistent", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestUpdatePaymentInvalidAmount(t *testing.T) {
	store := NewPaymentStore()

	payment := Payment{
		ID:       "payment-123",
		UserID:   "user-123",
		Amount:   100.50,
		Currency: "USD",
	}

	store.Save(payment)

	service := NewPaymentService(store)
	r := newRouter(service)

	body := strings.NewReader(`{"user_id":"user-321","amount":-100,"currency":"EUR"}`)
	req := httptest.NewRequest("PUT", "/payments/payment-123", body)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	payment, ok := store.Get("payment-123")
	if !ok {
		t.Fatalf("expected payment to still exist")
	}

	if payment.Amount != 100.50 {
		t.Fatalf("expected amount %.2f, got %.2f", 100.50, payment.Amount)
	}
}

func TestUpdatePaymentInvalidJSON(t *testing.T) {
	payment := Payment{
		ID:       "payment-123",
		UserID:   "user-123",
		Amount:   100.50,
		Currency: "USD",
	}

	store := NewPaymentStore()
	service := NewPaymentService(store)
	r := newRouter(service)

	store.Save(payment)

	body := strings.NewReader(`"{user_id: "user-321", "amount": 50, "currency":"USD"invalid"`)
	req := httptest.NewRequest("PUT", "/payments/payment-123", body)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	payment, ok := store.Get("payment-123")
	if !ok {
		t.Fatalf("expected payment to still exist")
	}

	if payment.Amount != 100.50 {
		t.Fatalf("expected payment amount = %.2f", 100.50)
	}

	if payment.UserID != "user-123" {
		t.Fatalf("expected payment user id = %s", "user-123")
	}

	if payment.Currency != "USD" {
		t.Fatalf("expected payment currency = %s", "USD")
	}
}

func TestCreatePaymentInvalidAmountType(t *testing.T) {
	body := strings.NewReader(`{"user_id":"123","amount":"ABC","currency":"USD"}`)
	req := httptest.NewRequest("POST", "/payments", body)
	req.Header.Set("Content-Type", "application/json")

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
		Amount:   100.50,
		Currency: "USD",
	}

	fp, err := store.CreateIdempotent("order-123", first)
	if err != nil {
		t.Fatalf("first CreateIdempotent() error: %v", err)
	}

	second := Payment{
		ID:       "payment-2",
		UserID:   "123",
		Amount:   200,
		Currency: "USD",
	}

	sp, err := store.CreateIdempotent("order-123", second)
	if err != nil {
		t.Fatalf("second CreateIdempotent() error: %v", err)
	}

	if fp.ID != sp.ID {
		t.Fatalf("payment IDs differ: first=%q, second=%q", fp.ID, sp.ID)
	}

	if sp.Amount != 100.50 {
		t.Fatalf("unexpected amount: got=%v, want=100.50", sp.Amount)
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
				Amount:   100.50,
				Currency: "USD",
			}

			p, err := store.CreateIdempotent("order-123", payment)
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

	created, err := service.Create("key-1", PaymentRequest{
		UserID:   "user-1",
		Amount:   100,
		Currency: "USD",
	})
	if err != nil {
		t.Fatal(err)
	}

	cancelled, err := service.Cancel(created.ID)
	if err != nil {
		t.Fatal(err)
	}

	if cancelled.Status != PaymentStatusCancelled {
		t.Errorf("expected status %q, got %q",
			PaymentStatusCancelled, cancelled.Status)
	}

	saved, ok := service.Get(created.ID)
	if !ok {
		t.Fatal("expected cancelled payment to remain in store")
	}

	if saved.Status != PaymentStatusCancelled {
		t.Errorf("expected stored status %q, got %q",
			PaymentStatusCancelled, saved.Status)
	}
}

func TestCancelPaymentAlreadyCancelled(t *testing.T) {
	store := NewPaymentStore()
	service := NewPaymentService(store)

	created, err := service.Create("key-1", PaymentRequest{
		UserID:   "user-1",
		Amount:   100,
		Currency: "USD",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = service.Cancel(created.ID)
	if err != nil {
		t.Fatal(err)
	}

	_, err = service.Cancel(created.ID)
	if !errors.Is(err, ErrPaymentCannotBeCancelled) {
		t.Fatalf("expected ErrPaymentCannotBeCancelled, got %v", err)
	}
}
