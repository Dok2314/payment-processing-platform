package payment

import (
	"testing"

	domainpayment "payment-processing-platform/internal/domain/payment"
)

func newPaymentForHashTest(
	t *testing.T,
	userID string,
	amount int64,
	currencyCode string,
) *domainpayment.Payment {
	t.Helper()

	currency, err := domainpayment.NewCurrency(currencyCode)
	if err != nil {
		t.Fatalf("failed to create currency: %v", err)
	}

	money, err := domainpayment.NewMoney(amount, *currency)
	if err != nil {
		t.Fatalf("failed to create money: %v", err)
	}

	payment, err := domainpayment.NewPayment("payment-123", userID, *money)
	if err != nil {
		t.Fatalf("failed to create payment: %v", err)
	}

	return payment
}

func TestCalculatePayloadHash(t *testing.T) {
	payment := newPaymentForHashTest(t, "user-123", 1500, "USD")

	hash, err := calculatePayloadHash(payment)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(hash) != 64 {
		t.Fatalf("expected SHA-256 hash length to be 64, got: %d", len(hash))
	}

	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
}

func TestCalculatePayloadHash_Deterministic(t *testing.T) {
	payment1 := newPaymentForHashTest(t, "user-123", 1500, "USD")
	payment2 := newPaymentForHashTest(t, "user-123", 1500, "USD")

	hash1, err := calculatePayloadHash(payment1)
	if err != nil {
		t.Fatalf("failed to calculate first hash: %v", err)
	}

	hash2, err := calculatePayloadHash(payment2)
	if err != nil {
		t.Fatalf("failed to calculate second hash: %v", err)
	}

	if hash1 != hash2 {
		t.Fatalf("expected identical hashes, got %s and %s", hash1, hash2)
	}
}

func TestCalculatePayloadHash_DifferentPayload(t *testing.T) {
	tests := []struct {
		name     string
		userID   string
		amount   int64
		currency string
	}{
		{
			name:     "different user ID",
			userID:   "user-456",
			amount:   1500,
			currency: "USD",
		},
		{
			name:     "different amount",
			userID:   "user-123",
			amount:   2000,
			currency: "USD",
		},
		{
			name:     "different currency",
			userID:   "user-123",
			amount:   1500,
			currency: "EUR",
		},
	}

	originalPayment := newPaymentForHashTest(t, "user-123", 1500, "USD")

	originalHash, err := calculatePayloadHash(originalPayment)
	if err != nil {
		t.Fatalf("failed to calculate original hash: %v", err)
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			payment := newPaymentForHashTest(t, tc.userID, tc.amount, tc.currency)

			hash, err := calculatePayloadHash(payment)
			if err != nil {
				t.Fatalf("failed to calculate hash: %v", err)
			}

			if hash == originalHash {
				t.Fatal("expected different payload to produce a different hash")
			}
		})
	}
}
