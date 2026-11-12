package payment

import "testing"

func TestNewCurrency(t *testing.T) {
	type testCase struct {
		name    string
		code    string
		wantErr bool
	}

	tests := []testCase{
		{name: "USD supported", code: "USD", wantErr: false},
		{name: "EUR supported", code: "EUR", wantErr: false},
		{name: "JPY supported", code: "JPY", wantErr: false},
		{name: "BTC unsupported", code: "BTC", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			currency, err := NewCurrency(tc.code)

			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for currency %s", tc.code)
				}
				if currency != nil {
					t.Fatalf("expected nil currency, got %v", currency)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if currency == nil {
				t.Fatal("expected currency, got nil")
			}

			if currency.Code() != tc.code {
				t.Fatalf("expected code %s, got %s", tc.code, currency.Code())
			}
		})
	}
}
