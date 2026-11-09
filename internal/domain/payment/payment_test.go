package payment

import (
	"errors"
	"testing"
)

func newTestMoney(t *testing.T, amount int64, currencyCode string) Money {
	t.Helper()

	currency, err := NewCurrency(currencyCode)
	if err != nil {
		t.Fatalf("NewCurrency() returned unexpected error: %v", err)
	}

	money, err := NewMoney(amount, *currency)
	if err != nil {
		t.Fatalf("NewMoney() returned unexpected error: %v", err)
	}

	return *money
}

func TestNewPayment(t *testing.T) {
	money := newTestMoney(t, 1500, "USD")

	payment, err := NewPayment("payment-123", "user-123", money)
	if err != nil {
		t.Fatalf("NewPayment() returned unexpected error: %v", err)
	}

	if payment == nil {
		t.Fatal("expected payment to be not nil")
	}

	if payment.ID() != "payment-123" {
		t.Fatalf("expected ID payment-123, got %s", payment.ID())
	}

	if payment.UserID() != "user-123" {
		t.Fatalf("expected UserID user-123, got %s", payment.UserID())
	}

	if payment.Amount() != 1500 {
		t.Fatalf("expected amount 1500, got %d", payment.Amount())
	}

	if payment.Currency().Code() != "USD" {
		t.Fatalf("expected currency USD, got %s", payment.Currency().Code())
	}

	if payment.Status() != PaymentStatusPending {
		t.Fatalf("expected status %s, got %s", PaymentStatusPending, payment.Status())
	}
}

func TestNewPayment_InvalidInput(t *testing.T) {
	tests := []struct {
		name   string
		id     string
		userID string
	}{
		{
			name:   "empty ID",
			id:     "",
			userID: "user-123",
		},
		{
			name:   "empty UserID",
			id:     "payment-123",
			userID: "",
		},
	}

	money := newTestMoney(t, 1500, "USD")

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			payment, err := NewPayment(tc.id, tc.userID, money)

			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if payment != nil {
				t.Fatal("expected payment to be nil")
			}
		})
	}
}

func TestPayment_MoveToProcessing(t *testing.T) {
	money := newTestMoney(t, 1500, "USD")
	payment, err := NewPayment("payment-123", "user-123", money)
	if err != nil {
		t.Fatalf("NewPayment() returned unexpected error: %v", err)
	}

	if err := payment.MoveToProcessing(); err != nil {
		t.Fatalf("MoveToProcessing() returned unexpected error: %v", err)
	}

	if payment.Status() != PaymentStatusProcessing {
		t.Fatalf("expected status %s, got %s", PaymentStatusProcessing, payment.Status())
	}

	err = payment.MoveToProcessing()
	if !errors.Is(err, ErrPaymentCannotBeProcessed) {
		t.Fatalf("expected error %v, got %v", ErrPaymentCannotBeProcessed, err)
	}

	if payment.Status() != PaymentStatusProcessing {
		t.Fatalf("expected status %s, got %s", PaymentStatusProcessing, payment.Status())
	}
}

func TestPayment_Complete(t *testing.T) {
	money := newTestMoney(t, 1500, "USD")
	payment, err := NewPayment("payment-123", "user-123", money)
	if err != nil {
		t.Fatalf("NewPayment() returned unexpected error: %v", err)
	}

	if err := payment.MoveToProcessing(); err != nil {
		t.Fatalf("MoveToProcessing() returned unexpected error: %v", err)
	}

	if err := payment.Complete(); err != nil {
		t.Fatalf("Complete() returned unexpected error: %v", err)
	}

	if payment.Status() != PaymentStatusCompleted {
		t.Fatalf("expected status %s, got %s", PaymentStatusCompleted, payment.Status())
	}

	if err := payment.Complete(); !errors.Is(err, ErrPaymentCannotBeCompleted) {
		t.Fatalf("expected error %v, got %v", ErrPaymentCannotBeCompleted, err)
	}

	if payment.Status() != PaymentStatusCompleted {
		t.Fatalf("expected status %s, got %s", PaymentStatusCompleted, payment.Status())
	}
}

func TestPayment_Cancel(t *testing.T) {
	money := newTestMoney(t, 1500, "USD")
	payment, err := NewPayment("payment-123", "user-123", money)
	if err != nil {
		t.Fatalf("NewPayment() returned unexpected error: %v", err)
	}

	if err := payment.Cancel(); err != nil {
		t.Fatalf("Cancel() returned unexpected error: %v", err)
	}

	if payment.Status() != PaymentStatusCancelled {
		t.Fatalf("expected status %s, got %s", PaymentStatusCancelled, payment.Status())
	}

	if err := payment.Cancel(); !errors.Is(err, ErrPaymentCannotBeCancelled) {
		t.Fatalf("expected error %v, got %v", ErrPaymentCannotBeCancelled, err)
	}

	if payment.Status() != PaymentStatusCancelled {
		t.Fatalf("expected status %s, got %s", PaymentStatusCancelled, payment.Status())
	}
}

func TestPayment_Fail(t *testing.T) {
	money := newTestMoney(t, 1500, "USD")
	payment, err := NewPayment("payment-123", "user-123", money)
	if err != nil {
		t.Fatalf("NewPayment() returned unexpected error: %v", err)
	}

	if err := payment.MoveToProcessing(); err != nil {
		t.Fatalf("MoveToProcessing() returned unexpected error: %v", err)
	}

	if err := payment.Fail(); err != nil {
		t.Fatalf("Fail() returned unexpected error: %v", err)
	}

	if payment.Status() != PaymentStatusFailed {
		t.Fatalf("expected status %s, got %s", PaymentStatusFailed, payment.Status())
	}

	if err := payment.Fail(); !errors.Is(err, ErrPaymentCannotBeFailed) {
		t.Fatalf("expected error %v, got %v", ErrPaymentCannotBeFailed, err)
	}

	if payment.Status() != PaymentStatusFailed {
		t.Fatalf("expected status %s, got %s", PaymentStatusFailed, payment.Status())
	}
}
