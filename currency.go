package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type CurrencyConfig struct {
	DecimalPlaces int
}

var SupportedCurrencies = map[string]CurrencyConfig{
	"USD": {DecimalPlaces: 2},
	"EUR": {DecimalPlaces: 2},
	"JPY": {DecimalPlaces: 0},
}

var amountPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)(\.[0-9]+)?$`)

func GetCurrencyConfig(currency string) (CurrencyConfig, error) {
	val, ok := SupportedCurrencies[currency]
	if !ok {
		return CurrencyConfig{}, fmt.Errorf("unsupported currency: %s", currency)
	}
	return val, nil
}

func IsValidAmountFormat(amount string) bool {
	return amountPattern.MatchString(amount)
}

func ParseAmount(amount string, currency string) (int64, error) {
	curConf, err := GetCurrencyConfig(currency)
	if err != nil {
		return 0, err
	}

	if !IsValidAmountFormat(amount) {
		return 0, fmt.Errorf("invalid amount format: %s", amount)
	}

	dotPos := strings.Index(amount, ".")

	decimalPlaces := 0
	if dotPos != -1 {
		decimalPlaces = len(amount) - dotPos - 1
	}

	if decimalPlaces > curConf.DecimalPlaces {
		return 0, fmt.Errorf("too many decimal places for %s", currency)
	}

	integerAmount := strings.Replace(amount, ".", "", 1)

	for i := decimalPlaces; i < curConf.DecimalPlaces; i++ {
		integerAmount += "0"
	}

	result, err := strconv.ParseInt(integerAmount, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount: %w", err)
	}

	if result <= 0 {
		return 0, fmt.Errorf("amount must be greater than zero")
	}

	return result, nil
}
