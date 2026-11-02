package main

import "testing"

func TestGetCurrencyConfig(t *testing.T) {
	curConf, err := GetCurrencyConfig("USD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if curConf.DecimalPlaces != 2 {
		t.Errorf("expected 2 decimal places, got %d", curConf.DecimalPlaces)
	}

	_, err = GetCurrencyConfig("XYZ")
	if err == nil {
		t.Fatal("expected error for unsupported currency")
	}
}

func TestParseAmount(t *testing.T) {
	got, err := ParseAmount("19.99", "USD")

	if err != nil {
		t.Fatal(err)
	}

	if got != 1999 {
		t.Fatalf("expected 1999, got %d", got)
	}

	tests := []struct {
		amount   string
		currency string
		want     int64
	}{
		{"19.99", "USD", 1999},
		{"19.9", "USD", 1990},
		{"150", "JPY", 150},
	}

	for _, tt := range tests {
		got, err := ParseAmount(tt.amount, tt.currency)

		if err != nil {
			t.Errorf("ParseAmount(%q, %q) unexpected error: %v", tt.amount, tt.currency, err)
			continue
		}

		if got != tt.want {
			t.Errorf("ParseAmount(%q, %q) = %d, want %d", tt.amount, tt.currency, got, tt.want)
		}
	}
}

func TestParseAmountInvalid(t *testing.T) {
	tests := []struct {
		amount   string
		currency string
	}{
		{"00019.99", "USD"},
		{"19.999", "USD"},
		{"19.", "USD"},
		{"0", "USD"},
		{"100", "GBP"},
		{"150.1", "JPY"},
	}

	for _, tt := range tests {
		_, err := ParseAmount(tt.amount, tt.currency)

		if err == nil {
			t.Errorf("ParseAmount(%q, %q) expected error", tt.amount, tt.currency)
		}
	}
}
