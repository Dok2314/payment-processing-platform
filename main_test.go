package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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
	r := newRouter()
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
	r := newRouter()
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
	r := newRouter()
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
	r := newRouter()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestCreatePaymentSuccess(t *testing.T) {
	body := strings.NewReader(`{"user_id":"123","amount":100.50,"currency":"USD"}`)
	req := httptest.NewRequest("POST", "/payments", body)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r := newRouter()
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
