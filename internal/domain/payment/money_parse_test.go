package payment

import (
	"errors"
	"testing"
)

func TestParseAmount(t *testing.T) {
	tests := []struct {
		name         string
		amount       string
		currencyCode string
		expected     int64
		wantErr      bool
	}{
		{
			name:         "USD integer amount",
			amount:       "100",
			currencyCode: "USD",
			expected:     10000,
		},
		{
			name:         "USD decimal amount",
			amount:       "100.50",
			currencyCode: "USD",
			expected:     10050,
		},
		{
			name:         "USD one decimal place",
			amount:       "10.5",
			currencyCode: "USD",
			expected:     1050,
		},
		{
			name:         "EUR decimal amount",
			amount:       "25.99",
			currencyCode: "EUR",
			expected:     2599,
		},
		{
			name:         "JPY integer amount",
			amount:       "1000",
			currencyCode: "JPY",
			expected:     1000,
		},
		{
			name:         "zero amount",
			amount:       "0",
			currencyCode: "USD",
			wantErr:      true,
		},
		{
			name:         "negative amount",
			amount:       "-10",
			currencyCode: "USD",
			wantErr:      true,
		},
		{
			name:         "too many decimal places",
			amount:       "10.999",
			currencyCode: "USD",
			wantErr:      true,
		},
		{
			name:         "JPY with decimal places",
			amount:       "100.5",
			currencyCode: "JPY",
			wantErr:      true,
		},
		{
			name:         "empty amount",
			amount:       "",
			currencyCode: "USD",
			wantErr:      true,
		},
		{
			name:         "invalid amount format",
			amount:       "abc",
			currencyCode: "USD",
			wantErr:      true,
		},
		{
			name:         "amount with comma",
			amount:       "10,50",
			currencyCode: "USD",
			wantErr:      true,
		},
		{
			name:         "amount with leading zero",
			amount:       "010.50",
			currencyCode: "USD",
			wantErr:      true,
		},
		{
			name:         "amount with only decimal part",
			amount:       ".50",
			currencyCode: "USD",
			wantErr:      true,
		},
		{
			name:         "amount with trailing decimal point",
			amount:       "10.",
			currencyCode: "USD",
			wantErr:      true,
		},
		{
			name:         "amount with whitespace",
			amount:       " 10.50 ",
			currencyCode: "USD",
			wantErr:      true,
		},
		{
			name:         "int64 overflow",
			amount:       "999999999999999999999999",
			currencyCode: "USD",
			wantErr:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			currency, err := NewCurrency(tc.currencyCode)
			if err != nil {
				t.Fatalf("NewCurrency() returned unexpected error: %v", err)
			}

			got, err := ParseAmount(tc.amount, *currency)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("ParseAmount() returned unexpected error: %v", err)
			}

			if got != tc.expected {
				t.Fatalf("expected amount %d, got %d", tc.expected, got)
			}
		})
	}
}

func TestParseAmount_InvalidAmountError(t *testing.T) {
	currency, err := NewCurrency("USD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = ParseAmount("0", *currency)

	if !errors.Is(err, ErrInvalidPaymentAmount) {
		t.Fatalf("expected ErrInvalidPaymentAmount, got %v", err)
	}
}
