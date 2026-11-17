package payment

import "testing"

func TestNewMoney(t *testing.T) {
	currency, err := NewCurrency("USD")
	if err != nil {
		t.Fatal("err should be nil")
	}

	money, err := NewMoney(1500, *currency)
	if err != nil {
		t.Fatal("err should be nil")
	}

	if money == nil {
		t.Fatal("money should not be nil")
	}

	if money.Amount() != 1500 {
		t.Fatalf("expected amount 1500, got %d", money.Amount())
	}

	if money.Currency().Code() != "USD" {
		t.Fatalf("expected currency USD, got %s", money.Currency().Code())
	}
}

func TestNewMoney_InvalidAmount(t *testing.T) {
	currency, err := NewCurrency("USD")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name   string
		amount int64
	}{
		{name: "zero amount", amount: 0},
		{name: "negative amount", amount: -100},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			money, err := NewMoney(tc.amount, *currency)

			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if money != nil {
				t.Fatal("expected money to be nil")
			}
		})
	}
}
