package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	applicationpayment "payment-processing-platform/internal/application/payment"
	domainpayment "payment-processing-platform/internal/domain/payment"
)

type fakePaymentRepository struct {
	createCalled bool
	payment      *domainpayment.Payment
	err          error
}

func (r *fakePaymentRepository) CreateIdempotent(
	ctx context.Context,
	key string,
	payment *domainpayment.Payment,
	payloadHash string,
) (*domainpayment.Payment, error) {
	r.createCalled = true

	if r.err != nil {
		return nil, r.err
	}

	if r.payment != nil {
		return r.payment, nil
	}

	return payment, nil
}

func (r *fakePaymentRepository) Get(
	ctx context.Context,
	id string,
) (*domainpayment.Payment, error) {
	return nil, nil
}

func (r *fakePaymentRepository) Update(
	ctx context.Context,
	payment *domainpayment.Payment,
) error {
	return nil
}

func TestCreatePaymentHTTPHandler_IdempotencyKeyReuse(t *testing.T) {
	repo := fakePaymentRepository{
		err: applicationpayment.ErrIdempotencyKeyReuse,
	}

	service := applicationpayment.NewPaymentService(&repo)

	req := httptest.NewRequest(
		"POST",
		"/payments",
		strings.NewReader(`{"user_id":"user-1","amount":"10.00","currency":"USD"}`),
	)
	req.Header.Set("Idempotency-Key", "payment-123")

	recorder := httptest.NewRecorder()

	createPaymentHTTPHandler(service, recorder, req)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected status = %v, got %v", http.StatusConflict, recorder.Code)
	}

	if !repo.createCalled {
		t.Fatal("expected repository CreateIdempotent to be called")
	}

	if !strings.Contains(
		recorder.Body.String(),
		"Idempotency key reused with different request payload",
	) {
		t.Fatalf("unexpected response body: %s", recorder.Body.String())
	}
}

func TestCreatePaymentHTTPHandler_MissedIdempotencyKey(t *testing.T) {
	var repo fakePaymentRepository

	service := applicationpayment.NewPaymentService(&repo)

	req := httptest.NewRequest(
		"POST",
		"/payments",
		strings.NewReader(`{"user_id":"user-1","amount":"10.00","currency":"USD"}`),
	)

	recorder := httptest.NewRecorder()

	createPaymentHTTPHandler(service, recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status = %v, got %v", http.StatusBadRequest, recorder.Code)
	}

	if repo.createCalled {
		t.Fatal("repository CreateIdempotent shouldn't be called")
	}
}

func TestCreatePaymentHTTPHandler_WhitespaceIdempotencyKey(t *testing.T) {
	var repo fakePaymentRepository

	service := applicationpayment.NewPaymentService(&repo)

	req := httptest.NewRequest(
		"POST",
		"/payments",
		strings.NewReader(`{"user_id":"user-1","amount":"10.00","currency":"USD"}`),
	)
	req.Header.Set("Idempotency-Key", "   ")

	recorder := httptest.NewRecorder()

	createPaymentHTTPHandler(service, recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status = %v, got %v", http.StatusBadRequest, recorder.Code)
	}

	if repo.createCalled {
		t.Fatal("repository CreateIdempotent shouldn't be called")
	}
}

func TestCreatePaymentHTTPHandler_InvalidJSON(t *testing.T) {
	var repo fakePaymentRepository

	service := applicationpayment.NewPaymentService(&repo)

	req := httptest.NewRequest(
		"POST",
		"/payments",
		strings.NewReader(`{"user_id":"use`),
	)
	req.Header.Set("Idempotency-Key", "payment-123")

	recorder := httptest.NewRecorder()

	createPaymentHTTPHandler(service, recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status = %v, got %v", http.StatusBadRequest, recorder.Code)
	}

	if repo.createCalled {
		t.Fatal("repository CreateIdempotent shouldn't be called")
	}
}

func TestCreatePaymentHTTPHandler_EmptyBody(t *testing.T) {
	var repo fakePaymentRepository

	service := applicationpayment.NewPaymentService(&repo)

	req := httptest.NewRequest(
		"POST",
		"/payments",
		strings.NewReader(""),
	)
	req.Header.Set("Idempotency-Key", "payment-123")

	recorder := httptest.NewRecorder()

	createPaymentHTTPHandler(service, recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status = %v, got %v", http.StatusBadRequest, recorder.Code)
	}

	if repo.createCalled {
		t.Fatal("repository CreateIdempotent shouldn't be called")
	}
}

func TestCreatePaymentHTTPHandler_InvalidPaymentRequest(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "empty_user_id",
			body: `{"user_id":"","amount":"10.00","currency":"USD"}`,
		},
		{
			name: "invalid_amount",
			body: `{"user_id":"user-1","amount":"abc","currency":"USD"}`,
		},
		{
			name: "invalid_currency",
			body: `{"user_id":"user-1","amount":"10.00","currency":"INVALID"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var repo fakePaymentRepository

			service := applicationpayment.NewPaymentService(&repo)

			req := httptest.NewRequest(
				"POST",
				"/payments",
				strings.NewReader(tt.body),
			)
			req.Header.Set("Idempotency-Key", "payment-123")

			recorder := httptest.NewRecorder()

			createPaymentHTTPHandler(service, recorder, req)

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf(
					"expected status = %v, got %v",
					http.StatusBadRequest,
					recorder.Code,
				)
			}

			if repo.createCalled {
				t.Fatal("repository CreateIdempotent shouldn't be called")
			}
		})
	}
}

func TestCreatePaymentHTTPHandler_Success(t *testing.T) {
	var repo fakePaymentRepository

	service := applicationpayment.NewPaymentService(&repo)

	req := httptest.NewRequest(
		"POST",
		"/payments",
		strings.NewReader(`{"user_id":"user-1","amount":"10.00","currency":"USD"}`),
	)
	req.Header.Set("Idempotency-Key", "payment-123")

	recorder := httptest.NewRecorder()

	createPaymentHTTPHandler(service, recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status = %v, got %v", http.StatusCreated, recorder.Code)
	}

	if !repo.createCalled {
		t.Fatal("expected repository CreateIdempotent to be called")
	}

	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("expected Content-Type = application/json, got %v", contentType)
	}

	if recorder.Body.Len() == 0 {
		t.Fatal("expected non-empty response body")
	}
}
