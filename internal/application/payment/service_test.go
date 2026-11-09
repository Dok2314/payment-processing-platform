package payment

import (
	"context"
	"errors"
	"testing"

	domainpayment "payment-processing-platform/internal/domain/payment"
)

type fakePaymentRepository struct {
	savedPayment *domainpayment.Payment
	saveCalled   bool
	saveErr      error
}

func (fr *fakePaymentRepository) Save(ctx context.Context, payment *domainpayment.Payment) error {
	fr.savedPayment = payment
	fr.saveCalled = true
	return fr.saveErr
}

func (fr *fakePaymentRepository) Get(ctx context.Context, id string) (*domainpayment.Payment, error) {
	return nil, nil
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
