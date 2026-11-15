package payment

import (
	"context"
	"errors"
	"testing"

	domainpayment "payment-processing-platform/internal/domain/payment"
)

type fakePaymentRepository struct {
	savedPayment   *domainpayment.Payment
	createCalled   bool
	createErr      error
	idempotencyKey string
	payloadHash    string

	paymentToGet *domainpayment.Payment
	getCalled    bool
	getErr       error

	updatedPayment *domainpayment.Payment
	updateCalled   bool
	updateErr      error
}

func (f *fakePaymentRepository) CreateIdempotent(
	ctx context.Context,
	key string,
	payment *domainpayment.Payment,
	payloadHash string,
) (*domainpayment.Payment, error) {
	f.createCalled = true
	f.idempotencyKey = key
	f.payloadHash = payloadHash

	if f.createErr != nil {
		return nil, f.createErr
	}

	f.savedPayment = payment
	return payment, nil
}

func (f *fakePaymentRepository) Get(
	ctx context.Context,
	id string,
) (*domainpayment.Payment, error) {
	f.getCalled = true
	return f.paymentToGet, f.getErr
}

func (f *fakePaymentRepository) Update(
	ctx context.Context,
	payment *domainpayment.Payment,
) error {
	f.updatedPayment = payment
	f.updateCalled = true
	return f.updateErr
}

func newTestPayment(t *testing.T, status domainpayment.PaymentStatus) *domainpayment.Payment {
	t.Helper()

	currency, err := domainpayment.NewCurrency("USD")
	if err != nil {
		t.Fatalf("failed to create currency: %v", err)
	}

	money, err := domainpayment.NewMoney(1500, *currency)
	if err != nil {
		t.Fatalf("failed to create money: %v", err)
	}

	payment, err := domainpayment.NewPayment("payment-123", "user-123", *money)
	if err != nil {
		t.Fatalf("failed to create payment: %v", err)
	}

	switch status {
	case domainpayment.PaymentStatusPending:
		// Already PENDING.
	case domainpayment.PaymentStatusProcessing:
		if err := payment.MoveToProcessing(); err != nil {
			t.Fatalf("failed to move payment to processing: %v", err)
		}
	case domainpayment.PaymentStatusCompleted:
		if err := payment.MoveToProcessing(); err != nil {
			t.Fatalf("failed to move payment to processing: %v", err)
		}
		if err := payment.Complete(); err != nil {
			t.Fatalf("failed to complete payment: %v", err)
		}
	default:
		t.Fatalf("unsupported test status: %s", status)
	}

	return payment
}

// CREATE

func TestPaymentService_Create_Success(t *testing.T) {
	req := PaymentRequest{
		UserID:   "user-123",
		Amount:   "15.00",
		Currency: "USD",
	}

	repo := &fakePaymentRepository{}
	service := NewPaymentService(repo)

	payment, err := service.Create(context.Background(), "test-key", req)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if payment == nil {
		t.Fatal("expected payment to be created")
	}

	if !repo.createCalled {
		t.Fatal("expected CreateIdempotent to be called")
	}

	if repo.idempotencyKey != "test-key" {
		t.Fatalf("expected idempotency key = test-key, got: %s", repo.idempotencyKey)
	}

	if repo.payloadHash == "" {
		t.Fatal("expected payload hash to be generated")
	}

	if repo.savedPayment == nil {
		t.Fatal("expected payment to be saved")
	}

	if repo.savedPayment.Amount() != 1500 {
		t.Fatalf("expected amount = 1500, got: %d", repo.savedPayment.Amount())
	}

	if repo.savedPayment.Currency().Code() != "USD" {
		t.Fatalf("expected currency = USD, got: %s", repo.savedPayment.Currency().Code())
	}

	if repo.savedPayment.Status() != domainpayment.PaymentStatusPending {
		t.Fatalf(
			"expected status = %s, got: %s",
			domainpayment.PaymentStatusPending,
			repo.savedPayment.Status(),
		)
	}

	if repo.savedPayment.UserID() != "user-123" {
		t.Fatalf("expected userID = user-123, got: %s", repo.savedPayment.UserID())
	}
}

func TestPaymentService_Create_InvalidCurrency(t *testing.T) {
	req := PaymentRequest{
		UserID:   "user-123",
		Amount:   "15.00",
		Currency: "XXX",
	}

	repo := &fakePaymentRepository{}
	service := NewPaymentService(repo)

	payment, err := service.Create(context.Background(), "test-key", req)

	if err == nil {
		t.Fatal("expected error for unsupported currency")
	}

	if payment != nil {
		t.Fatal("expected payment to be nil")
	}

	if repo.createCalled {
		t.Fatal("repository should not be called")
	}
}

func TestPaymentService_Create_InvalidAmount(t *testing.T) {
	req := PaymentRequest{
		UserID:   "user-123",
		Amount:   "abc",
		Currency: "USD",
	}

	repo := &fakePaymentRepository{}
	service := NewPaymentService(repo)

	payment, err := service.Create(context.Background(), "test-key", req)

	if err == nil {
		t.Fatal("expected error for invalid amount")
	}

	if payment != nil {
		t.Fatal("expected payment to be nil")
	}

	if repo.createCalled {
		t.Fatal("repository should not be called")
	}
}

func TestPaymentService_Create_InvalidUserID(t *testing.T) {
	req := PaymentRequest{
		UserID:   "",
		Amount:   "15.00",
		Currency: "USD",
	}

	repo := &fakePaymentRepository{}
	service := NewPaymentService(repo)

	payment, err := service.Create(context.Background(), "test-key", req)

	if err == nil {
		t.Fatal("expected error for empty user ID")
	}

	if payment != nil {
		t.Fatal("expected payment to be nil")
	}

	if repo.createCalled {
		t.Fatal("repository should not be called")
	}
}

func TestPaymentService_Create_RepositoryError(t *testing.T) {
	repositoryErr := errors.New("database connection failed")

	req := PaymentRequest{
		UserID:   "user-123",
		Amount:   "15.00",
		Currency: "USD",
	}

	repo := &fakePaymentRepository{
		createErr: repositoryErr,
	}

	service := NewPaymentService(repo)

	payment, err := service.Create(context.Background(), "test-key", req)

	if !errors.Is(err, repositoryErr) {
		t.Fatalf("expected repository error, got: %v", err)
	}

	if payment != nil {
		t.Fatal("expected payment to be nil")
	}

	if !repo.createCalled {
		t.Fatal("expected CreateIdempotent to be called")
	}
}

// CANCEL

func TestPaymentService_Cancel_Success(t *testing.T) {
	payment := newTestPayment(t, domainpayment.PaymentStatusPending)

	repo := &fakePaymentRepository{
		paymentToGet: payment,
	}

	service := NewPaymentService(repo)

	cancelledPayment, err := service.Cancel(context.Background(), "payment-123")
	if err != nil {
		t.Fatalf("failed to cancel payment: %v", err)
	}

	if cancelledPayment.Status() != domainpayment.PaymentStatusCancelled {
		t.Fatalf(
			"expected status = %s, got: %s",
			domainpayment.PaymentStatusCancelled,
			cancelledPayment.Status(),
		)
	}

	if !repo.getCalled || !repo.updateCalled {
		t.Fatal("expected Get and Update to be called")
	}

	if repo.updatedPayment.Status() != domainpayment.PaymentStatusCancelled {
		t.Fatal("repository received payment with incorrect status")
	}
}

func TestPaymentService_Cancel_GetError(t *testing.T) {
	repoErr := errors.New("database connection failed")

	repo := &fakePaymentRepository{
		getErr: repoErr,
	}

	service := NewPaymentService(repo)

	payment, err := service.Cancel(context.Background(), "payment-123")

	if !errors.Is(err, repoErr) {
		t.Fatalf("expected error: %v, got: %v", repoErr, err)
	}

	if payment != nil {
		t.Fatal("expected payment to be nil")
	}

	if repo.updateCalled {
		t.Fatal("Update should not be called")
	}
}

func TestPaymentService_Cancel_InvalidTransition(t *testing.T) {
	payment := newTestPayment(t, domainpayment.PaymentStatusProcessing)

	repo := &fakePaymentRepository{
		paymentToGet: payment,
	}

	service := NewPaymentService(repo)

	cancelledPayment, err := service.Cancel(context.Background(), "payment-123")

	if err == nil {
		t.Fatal("expected error when cancelling payment in PROCESSING status")
	}

	if cancelledPayment != nil {
		t.Fatal("expected payment to be nil")
	}

	if repo.updateCalled {
		t.Fatal("Update should not be called")
	}
}

func TestPaymentService_Cancel_UpdateError(t *testing.T) {
	repoErr := errors.New("database connection failed")
	payment := newTestPayment(t, domainpayment.PaymentStatusPending)

	repo := &fakePaymentRepository{
		updateErr:    repoErr,
		paymentToGet: payment,
	}

	service := NewPaymentService(repo)

	cancelledPayment, err := service.Cancel(context.Background(), "payment-123")

	if !errors.Is(err, repoErr) {
		t.Fatalf("expected error: %v, got: %v", repoErr, err)
	}

	if cancelledPayment != nil {
		t.Fatal("expected payment to be nil")
	}

	if !repo.updateCalled {
		t.Fatal("expected Update to be called")
	}
}

// PROCESS

func TestPaymentService_Process_Success(t *testing.T) {
	payment := newTestPayment(t, domainpayment.PaymentStatusPending)

	repo := &fakePaymentRepository{
		paymentToGet: payment,
	}

	service := NewPaymentService(repo)

	processedPayment, err := service.Process(context.Background(), "payment-123")
	if err != nil {
		t.Fatalf("failed to process payment: %v", err)
	}

	if processedPayment.Status() != domainpayment.PaymentStatusProcessing {
		t.Fatalf(
			"expected status = %s, got: %s",
			domainpayment.PaymentStatusProcessing,
			processedPayment.Status(),
		)
	}

	if !repo.getCalled || !repo.updateCalled {
		t.Fatal("expected Get and Update to be called")
	}

	if repo.updatedPayment.Status() != domainpayment.PaymentStatusProcessing {
		t.Fatal("repository received payment with incorrect status")
	}
}

func TestPaymentService_Process_GetError(t *testing.T) {
	repoErr := errors.New("database connection failed")

	repo := &fakePaymentRepository{
		getErr: repoErr,
	}

	service := NewPaymentService(repo)

	payment, err := service.Process(context.Background(), "payment-123")

	if !errors.Is(err, repoErr) {
		t.Fatalf("expected error: %v, got: %v", repoErr, err)
	}

	if payment != nil {
		t.Fatal("expected payment to be nil")
	}

	if repo.updateCalled {
		t.Fatal("Update should not be called")
	}
}

func TestPaymentService_Process_InvalidTransition(t *testing.T) {
	payment := newTestPayment(t, domainpayment.PaymentStatusProcessing)

	repo := &fakePaymentRepository{
		paymentToGet: payment,
	}

	service := NewPaymentService(repo)

	processedPayment, err := service.Process(context.Background(), "payment-123")

	if err == nil {
		t.Fatal("expected error when processing payment in PROCESSING status")
	}

	if processedPayment != nil {
		t.Fatal("expected payment to be nil")
	}

	if repo.updateCalled {
		t.Fatal("Update should not be called")
	}
}

func TestPaymentService_Process_UpdateError(t *testing.T) {
	repoErr := errors.New("database connection failed")
	payment := newTestPayment(t, domainpayment.PaymentStatusPending)

	repo := &fakePaymentRepository{
		updateErr:    repoErr,
		paymentToGet: payment,
	}

	service := NewPaymentService(repo)

	processedPayment, err := service.Process(context.Background(), "payment-123")

	if !errors.Is(err, repoErr) {
		t.Fatalf("expected error: %v, got: %v", repoErr, err)
	}

	if processedPayment != nil {
		t.Fatal("expected payment to be nil")
	}

	if !repo.updateCalled {
		t.Fatal("expected Update to be called")
	}
}

// COMPLETE

func TestPaymentService_Complete_Success(t *testing.T) {
	payment := newTestPayment(t, domainpayment.PaymentStatusProcessing)

	repo := &fakePaymentRepository{
		paymentToGet: payment,
	}

	service := NewPaymentService(repo)

	completedPayment, err := service.Complete(context.Background(), "payment-123")
	if err != nil {
		t.Fatalf("failed to complete payment: %v", err)
	}

	if completedPayment.Status() != domainpayment.PaymentStatusCompleted {
		t.Fatalf(
			"expected status = %s, got: %s",
			domainpayment.PaymentStatusCompleted,
			completedPayment.Status(),
		)
	}

	if !repo.getCalled || !repo.updateCalled {
		t.Fatal("expected Get and Update to be called")
	}

	if repo.updatedPayment.Status() != domainpayment.PaymentStatusCompleted {
		t.Fatal("repository received payment with incorrect status")
	}
}

func TestPaymentService_Complete_GetError(t *testing.T) {
	repoErr := errors.New("database connection failed")

	repo := &fakePaymentRepository{
		getErr: repoErr,
	}

	service := NewPaymentService(repo)

	payment, err := service.Complete(context.Background(), "payment-123")

	if !errors.Is(err, repoErr) {
		t.Fatalf("expected error: %v, got: %v", repoErr, err)
	}

	if payment != nil {
		t.Fatal("expected payment to be nil")
	}

	if repo.updateCalled {
		t.Fatal("Update should not be called")
	}
}

func TestPaymentService_Complete_InvalidTransition(t *testing.T) {
	payment := newTestPayment(t, domainpayment.PaymentStatusPending)

	repo := &fakePaymentRepository{
		paymentToGet: payment,
	}

	service := NewPaymentService(repo)

	completedPayment, err := service.Complete(context.Background(), "payment-123")

	if err == nil {
		t.Fatal("expected error when completing payment in PENDING status")
	}

	if completedPayment != nil {
		t.Fatal("expected payment to be nil")
	}

	if repo.updateCalled {
		t.Fatal("Update should not be called")
	}
}

func TestPaymentService_Complete_UpdateError(t *testing.T) {
	repoErr := errors.New("database connection failed")
	payment := newTestPayment(t, domainpayment.PaymentStatusProcessing)

	repo := &fakePaymentRepository{
		updateErr:    repoErr,
		paymentToGet: payment,
	}

	service := NewPaymentService(repo)

	completedPayment, err := service.Complete(context.Background(), "payment-123")

	if !errors.Is(err, repoErr) {
		t.Fatalf("expected error: %v, got: %v", repoErr, err)
	}

	if completedPayment != nil {
		t.Fatal("expected payment to be nil")
	}

	if !repo.updateCalled {
		t.Fatal("expected Update to be called")
	}
}
