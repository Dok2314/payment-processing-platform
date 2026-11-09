package payment

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var amountPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)(\.[0-9]+)?$`)

type Money struct {
	amount   int64
	currency Currency
}

func NewMoney(amount int64, currency Currency) (*Money, error) {
	if amount <= 0 {
		return nil, ErrInvalidPaymentAmount
	}

	return &Money{
		amount:   amount,
		currency: currency,
	}, nil
}

func (m *Money) Amount() int64 {
	return m.amount
}

func (m *Money) Currency() Currency {
	return m.currency
}

func IsValidAmountFormat(amount string) bool {
	return amountPattern.MatchString(amount)
}

func ParseAmount(amount string, currency Currency) (int64, error) {
	currencyDecimalPlaces := currency.DecimalPlaces()

	if !IsValidAmountFormat(amount) {
		return 0, fmt.Errorf("invalid amount format: %s", amount)
	}

	dotPos := strings.Index(amount, ".")

	amountDecimalPlaces := 0
	if dotPos != -1 {
		amountDecimalPlaces = len(amount) - dotPos - 1
	}

	if amountDecimalPlaces > currencyDecimalPlaces {
		return 0, fmt.Errorf("too many decimal places for %s", currency.Code())
	}

	integerAmount := strings.Replace(amount, ".", "", 1)

	for i := amountDecimalPlaces; i < currencyDecimalPlaces; i++ {
		integerAmount += "0"
	}

	result, err := strconv.ParseInt(integerAmount, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount: %w", err)
	}

	if result <= 0 {
		return 0, ErrInvalidPaymentAmount
	}

	return result, nil
}
