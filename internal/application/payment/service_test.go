package payment

import (
	"context"
	"errors"
	"testing"

	domainpayment "payment-processing-platform/internal/domain/payment"
)

type fakePaymentRepository struct {
	savedPayment   *domainpayment.Payment
	saveCalled     bool
	saveErr        error
	paymentToGet   *domainpayment.Payment
	getCalled      bool
	updatedPayment *domainpayment.Payment
	updateCalled   bool
	getErr         error
	updateErr      error
}

func (fr *fakePaymentRepository) Save(ctx context.Context, payment *domainpayment.Payment) error {
	fr.savedPayment = payment
	fr.saveCalled = true
	return fr.saveErr
}

func (fr *fakePaymentRepository) Get(ctx context.Context, id string) (*domainpayment.Payment, error) {
	fr.getCalled = true
	return fr.paymentToGet, fr.getErr
}

func (fr *fakePaymentRepository) Update(ctx context.Context, payment *domainpayment.Payment) error {
	fr.updatedPayment = payment
	fr.updateCalled = true

	return fr.updateErr
}

func TestPaymentService_Create_Success(t *testing.T) {
	req := PaymentRequest{
		UserID:   "user-123",
		Amount:   "15.00",
		Currency: "USD",
	}

	repo := &fakePaymentRepository{}
	service := NewPaymentService(repo)

	payment, err := service.Create(context.Background(), req)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if payment == nil {
		t.Fatal("expected payment to be created")
	}

	if !repo.saveCalled {
		t.Fatal("expected repository Save to be called")
	}

	if repo.savedPayment == nil {
		t.Fatal("expected payment to be saved in repository")
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

	payment, err := service.Create(context.Background(), req)

	if err == nil {
		t.Fatal("expected error for unsupported currency")
	}

	if payment != nil {
		t.Fatal("expected payment to be nil")
	}

	if repo.saveCalled {
		t.Fatal("repository Save should not be called")
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

	payment, err := service.Create(context.Background(), req)

	if err == nil {
		t.Fatal("expected error for invalid amount")
	}

	if payment != nil {
		t.Fatal("expected payment to be nil")
	}

	if repo.saveCalled {
		t.Fatal("repository Save should not be called")
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

	payment, err := service.Create(context.Background(), req)

	if err == nil {
		t.Fatal("expected error for empty user ID")
	}

	if payment != nil {
		t.Fatal("expected payment to be nil")
	}

	if repo.saveCalled {
		t.Fatal("repository Save should not be called")
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
		saveErr: repositoryErr,
	}

	service := NewPaymentService(repo)

	payment, err := service.Create(context.Background(), req)

	if err == nil {
		t.Fatal("expected repository error")
	}

	if !errors.Is(err, repositoryErr) {
		t.Fatalf("expected repository error, got: %v", err)
	}

	if payment != nil {
		t.Fatal("expected payment to be nil")
	}

	if !repo.saveCalled {
		t.Fatal("expected repository Save to be called")
	}
}

func TestPaymentService_Cancel_Success(t *testing.T) {
	id := "payment-123"

	currency, err := domainpayment.NewCurrency("USD")
	if err != nil {
		t.Fatalf("failed to create currency: %v", err)
	}

	money, err := domainpayment.NewMoney(1500, *currency)
	if err != nil {
		t.Fatalf("failed to create money: %v", err)
	}

	payment, err := domainpayment.NewPayment(id, "user-123", *money)
	if err != nil {
		t.Fatalf("failed to create payment: %v", err)
	}

	repo := fakePaymentRepository{
		paymentToGet: payment,
	}

	service := NewPaymentService(&repo)

	cancelledPayment, err := service.Cancel(context.Background(), id)
	if err != nil {
		t.Fatalf("failed to cancel payment: %v", err)
	}

	if cancelledPayment == nil {
		t.Fatalf("cancelled payment should not be nil")
	}

	if cancelledPayment.Status() != domainpayment.PaymentStatusCancelled {
		t.Fatalf("payment status should be: %s, given %s", domainpayment.PaymentStatusCancelled, cancelledPayment.Status())
	}

	if !repo.getCalled {
		t.Fatal("get method should be called")
	}

	if !repo.updateCalled {
		t.Fatal("update method should be called")
	}

	if repo.updatedPayment == nil {
		t.Fatal("updated payment should not be nil")
	}

	if repo.updatedPayment.Status() != domainpayment.PaymentStatusCancelled {
		t.Fatalf("updated payment status should be: %s, given: %s", domainpayment.PaymentStatusCancelled, repo.updatedPayment.Status())
	}
}

func TestPaymentService_Cancel_GetError(t *testing.T) {
	err := errors.New("database connection failed")

	repo := fakePaymentRepository{
		getErr: err,
	}

	service := NewPaymentService(&repo)

	payment, cancelErr := service.Cancel(context.Background(), "payment-123")

	if !errors.Is(cancelErr, err) {
		t.Fatalf("expected error: %v, got %v", err, cancelErr)
	}

	if payment != nil {
		t.Fatal("payment should be nil")
	}

	if repo.updateCalled {
		t.Fatalf("update shouldn't be called")
	}
}

func TestPaymentService_Cancel_InvalidTransition(t *testing.T) {
	id := "payment-123"

	currency, err := domainpayment.NewCurrency("USD")
	if err != nil {
		t.Fatalf("failed to create currency: %v", err)
	}

	money, err := domainpayment.NewMoney(1500, *currency)
	if err != nil {
		t.Fatalf("failed to create money: %v", err)
	}

	payment, err := domainpayment.NewPayment(id, "user-123", *money)
	if err != nil {
		t.Fatalf("failed to create payment: %v", err)
	}

	err = payment.MoveToProcessing()
	if err != nil {
		t.Fatalf("failed move payment to processing: %v", err)
	}

	fakeRepo := fakePaymentRepository{
		paymentToGet: payment,
	}

	service := NewPaymentService(&fakeRepo)

	cancelledPayment, cancelErr := service.Cancel(context.Background(), id)

	if cancelErr == nil {
		t.Fatal("expected error when cancelling payment in PROCESSING status")
	}

	if cancelledPayment != nil {
		t.Fatal("cancelled payment should be nil")
	}
}

func TestPaymentService_Cancel_UpdateError(t *testing.T) {
	id := "payment-123"

	repoErr := errors.New("database connection failed")

	currency, err := domainpayment.NewCurrency("USD")
	if err != nil {
		t.Fatalf("failed to create currency: %v", err)
	}

	money, err := domainpayment.NewMoney(1500, *currency)
	if err != nil {
		t.Fatalf("failed to create money: %v", err)
	}

	payment, err := domainpayment.NewPayment(id, "user-123", *money)
	if err != nil {
		t.Fatalf("failed to create payment: %v", err)
	}

	repo := fakePaymentRepository{
		updateErr:    repoErr,
		paymentToGet: payment,
	}

	service := NewPaymentService(&repo)

	cancelledPayment, cancelErr := service.Cancel(context.Background(), id)

	if !errors.Is(cancelErr, repoErr) {
		t.Fatalf("expected err: %v, got %v", repoErr, cancelErr)
	}

	if cancelledPayment != nil {
		t.Fatal("expected payment to be nil")
	}

	if !repo.updateCalled {
		t.Fatal("expected repository Update to be called")
	}
}
