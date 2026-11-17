package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	applicationpayment "payment-processing-platform/internal/application/payment"
	domainpayment "payment-processing-platform/internal/domain/payment"

	"github.com/go-chi/chi/v5"
)

type paymentHandlersTestRepository struct {
	payment    *domainpayment.Payment
	getErr     error
	updateErr  error
	updateCall bool
}

func (r *paymentHandlersTestRepository) CreateIdempotent(
	ctx context.Context,
	key string,
	payment *domainpayment.Payment,
	payloadHash string,
) (*domainpayment.Payment, error) {
	return payment, nil
}

func (r *paymentHandlersTestRepository) Get(
	ctx context.Context,
	id string,
) (*domainpayment.Payment, error) {
	return r.payment, r.getErr
}

func (r *paymentHandlersTestRepository) Update(
	ctx context.Context,
	payment *domainpayment.Payment,
) error {
	r.updateCall = true
	return r.updateErr
}

func newPaymentForHandlerTest(
	t *testing.T,
	status domainpayment.PaymentStatus,
) *domainpayment.Payment {
	t.Helper()

	currency, err := domainpayment.NewCurrency("USD")
	if err != nil {
		t.Fatal(err)
	}

	money, err := domainpayment.NewMoney(1500, *currency)
	if err != nil {
		t.Fatal(err)
	}

	payment, err := domainpayment.NewPayment("payment-123", "user-123", *money)
	if err != nil {
		t.Fatal(err)
	}

	switch status {
	case domainpayment.PaymentStatusPending:
	case domainpayment.PaymentStatusProcessing:
		if err := payment.MoveToProcessing(); err != nil {
			t.Fatal(err)
		}
	case domainpayment.PaymentStatusCompleted:
		if err := payment.MoveToProcessing(); err != nil {
			t.Fatal(err)
		}
		if err := payment.Complete(); err != nil {
			t.Fatal(err)
		}
	default:
		t.Fatalf("unsupported status: %s", status)
	}

	return payment
}

func newHandlerRequest(
	method string,
	path string,
) (*httptest.ResponseRecorder, *http.Request) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "payment-123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	return recorder, req
}

func TestGetPaymentHTTPHandler_Success(t *testing.T) {
	repo := &paymentHandlersTestRepository{
		payment: newPaymentForHandlerTest(t, domainpayment.PaymentStatusPending),
	}
	service := applicationpayment.NewPaymentService(repo)

	recorder, req := newHandlerRequest(http.MethodGet, "/payments/payment-123")

	getPaymentHTTPHandler(service, recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("expected application/json, got %q", contentType)
	}
}

func TestGetPaymentHTTPHandler_NotFound(t *testing.T) {
	repo := &paymentHandlersTestRepository{
		getErr: applicationpayment.ErrPaymentNotFound,
	}
	service := applicationpayment.NewPaymentService(repo)

	recorder, req := newHandlerRequest(http.MethodGet, "/payments/payment-123")

	getPaymentHTTPHandler(service, recorder, req)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, recorder.Code)
	}
}

func TestMoveToProcessingHTTPHandler_Success(t *testing.T) {
	repo := &paymentHandlersTestRepository{
		payment: newPaymentForHandlerTest(t, domainpayment.PaymentStatusPending),
	}
	service := applicationpayment.NewPaymentService(repo)

	recorder, req := newHandlerRequest(http.MethodPost, "/payments/payment-123/process")

	moveToProcessingHTTPHandler(service, recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if !repo.updateCall {
		t.Fatal("expected repository Update to be called")
	}
}

func TestMoveToProcessingHTTPHandler_InvalidTransition(t *testing.T) {
	repo := &paymentHandlersTestRepository{
		payment: newPaymentForHandlerTest(t, domainpayment.PaymentStatusProcessing),
	}
	service := applicationpayment.NewPaymentService(repo)

	recorder, req := newHandlerRequest(http.MethodPost, "/payments/payment-123/process")

	moveToProcessingHTTPHandler(service, recorder, req)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, recorder.Code)
	}

	if repo.updateCall {
		t.Fatal("Update should not be called")
	}
}

func TestCompletePaymentHTTPHandler_Success(t *testing.T) {
	repo := &paymentHandlersTestRepository{
		payment: newPaymentForHandlerTest(t, domainpayment.PaymentStatusProcessing),
	}
	service := applicationpayment.NewPaymentService(repo)

	recorder, req := newHandlerRequest(http.MethodPost, "/payments/payment-123/complete")

	completePaymentHTTPHandler(service, recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if !repo.updateCall {
		t.Fatal("expected repository Update to be called")
	}
}

func TestCompletePaymentHTTPHandler_InvalidTransition(t *testing.T) {
	repo := &paymentHandlersTestRepository{
		payment: newPaymentForHandlerTest(t, domainpayment.PaymentStatusPending),
	}
	service := applicationpayment.NewPaymentService(repo)

	recorder, req := newHandlerRequest(http.MethodPost, "/payments/payment-123/complete")

	completePaymentHTTPHandler(service, recorder, req)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, recorder.Code)
	}

	if repo.updateCall {
		t.Fatal("Update should not be called")
	}
}

func TestCancelPaymentHTTPHandler_Success(t *testing.T) {
	repo := &paymentHandlersTestRepository{
		payment: newPaymentForHandlerTest(t, domainpayment.PaymentStatusPending),
	}
	service := applicationpayment.NewPaymentService(repo)

	recorder, req := newHandlerRequest(http.MethodPost, "/payments/payment-123/cancel")

	cancelPaymentHTTPHandler(service, recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if !repo.updateCall {
		t.Fatal("expected repository Update to be called")
	}
}

func TestCancelPaymentHTTPHandler_InvalidTransition(t *testing.T) {
	repo := &paymentHandlersTestRepository{
		payment: newPaymentForHandlerTest(t, domainpayment.PaymentStatusCompleted),
	}
	service := applicationpayment.NewPaymentService(repo)

	recorder, req := newHandlerRequest(http.MethodPost, "/payments/payment-123/cancel")

	cancelPaymentHTTPHandler(service, recorder, req)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, recorder.Code)
	}

	if repo.updateCall {
		t.Fatal("Update should not be called")
	}
}

func TestPaymentHTTPHandlers_RepositoryError(t *testing.T) {
	repoErr := errors.New("database connection failed")

	tests := []struct {
		name    string
		handler func(*applicationpayment.PaymentService, http.ResponseWriter, *http.Request)
	}{
		{
			name: "get",
			handler: func(s *applicationpayment.PaymentService, w http.ResponseWriter, r *http.Request) {
				getPaymentHTTPHandler(s, w, r)
			},
		},
		{
			name: "process",
			handler: func(s *applicationpayment.PaymentService, w http.ResponseWriter, r *http.Request) {
				moveToProcessingHTTPHandler(s, w, r)
			},
		},
		{
			name: "complete",
			handler: func(s *applicationpayment.PaymentService, w http.ResponseWriter, r *http.Request) {
				completePaymentHTTPHandler(s, w, r)
			},
		},
		{
			name: "cancel",
			handler: func(s *applicationpayment.PaymentService, w http.ResponseWriter, r *http.Request) {
				cancelPaymentHTTPHandler(s, w, r)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &paymentHandlersTestRepository{
				getErr: repoErr,
			}
			service := applicationpayment.NewPaymentService(repo)

			recorder, req := newHandlerRequest(http.MethodPost, "/payments/payment-123")

			tc.handler(service, recorder, req)

			expectedStatus := http.StatusInternalServerError
			if tc.name == "get" {
				expectedStatus = http.StatusInternalServerError
			}

			if recorder.Code != expectedStatus {
				t.Fatalf("expected status %d, got %d", expectedStatus, recorder.Code)
			}

			if repo.updateCall {
				t.Fatal("Update should not be called when Get fails")
			}
		})
	}
}
