package payment

import (
	"testing"
)

func TestReconstitutePayment(t *testing.T) {
	currency, err := NewCurrency("USD")
	if err != nil {
		t.Fatal(err)
	}

	money, err := NewMoney(10050, *currency)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name           string
		id             string
		userID         string
		status         PaymentStatus
		expectedStatus PaymentStatus
		wantErr        bool
	}{
		{
			name:           "pending payment",
			id:             "payment-1",
			userID:         "user-1",
			status:         PaymentStatusPending,
			expectedStatus: PaymentStatusPending,
		},
		{
			name:           "processing payment",
			id:             "payment-2",
			userID:         "user-1",
			status:         PaymentStatusProcessing,
			expectedStatus: PaymentStatusProcessing,
		},
		{
			name:           "completed payment",
			id:             "payment-3",
			userID:         "user-1",
			status:         PaymentStatusCompleted,
			expectedStatus: PaymentStatusCompleted,
		},
		{
			name:           "failed payment",
			id:             "payment-4",
			userID:         "user-1",
			status:         PaymentStatusFailed,
			expectedStatus: PaymentStatusFailed,
		},
		{
			name:           "cancelled payment",
			id:             "payment-5",
			userID:         "user-1",
			status:         PaymentStatusCancelled,
			expectedStatus: PaymentStatusCancelled,
		},
		{
			name:    "invalid status",
			id:      "payment-6",
			userID:  "user-1",
			status:  PaymentStatus("UNKNOWN"),
			wantErr: true,
		},
		{
			name:    "empty payment ID",
			id:      "",
			userID:  "user-1",
			status:  PaymentStatusPending,
			wantErr: true,
		},
		{
			name:    "empty user ID",
			id:      "payment-7",
			userID:  "",
			status:  PaymentStatusPending,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			payment, err := ReconstitutePayment(
				tc.id,
				tc.userID,
				*money,
				tc.status,
			)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				if payment != nil {
					t.Fatal("expected nil payment when error occurs")
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if payment == nil {
				t.Fatal("expected payment, got nil")
			}

			if payment.ID() != tc.id {
				t.Errorf("expected ID %q, got %q", tc.id, payment.ID())
			}

			if payment.UserID() != tc.userID {
				t.Errorf("expected UserID %q, got %q", tc.userID, payment.UserID())
			}

			if payment.Status() != tc.expectedStatus {
				t.Errorf(
					"expected status %q, got %q",
					tc.expectedStatus,
					payment.Status(),
				)
			}

			if payment.Amount() != 10050 {
				t.Errorf("expected amount 10050, got %d", payment.Amount())
			}

			if payment.Currency().Code() != "USD" {
				t.Errorf(
					"expected currency USD, got %s",
					payment.Currency().Code(),
				)
			}
		})
	}
}

func TestReconstitutePayment_InvalidPaymentID(t *testing.T) {
	currency, err := NewCurrency("USD")
	if err != nil {
		t.Fatal(err)
	}

	money, err := NewMoney(10000, *currency)
	if err != nil {
		t.Fatal(err)
	}

	_, err = ReconstitutePayment(
		"",
		"user-1",
		*money,
		PaymentStatusPending,
	)

	if err != ErrInvalidPaymentID {
		t.Fatalf("expected ErrInvalidPaymentID, got %v", err)
	}
}

func TestReconstitutePayment_InvalidUserID(t *testing.T) {
	currency, err := NewCurrency("USD")
	if err != nil {
		t.Fatal(err)
	}

	money, err := NewMoney(10000, *currency)
	if err != nil {
		t.Fatal(err)
	}

	_, err = ReconstitutePayment(
		"payment-1",
		"",
		*money,
		PaymentStatusPending,
	)

	if err != ErrInvalidPaymentUserID {
		t.Fatalf("expected ErrInvalidPaymentUserID, got %v", err)
	}
}
