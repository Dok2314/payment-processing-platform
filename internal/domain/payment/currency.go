package payment

import "fmt"

type CurrencyConfig struct {
	decimalPlaces int
}

var supportedCurrencies = map[string]CurrencyConfig{
	"USD": {decimalPlaces: 2},
	"EUR": {decimalPlaces: 2},
	"JPY": {decimalPlaces: 0},
}

type Currency struct {
	code string
}

func NewCurrency(code string) (*Currency, error) {
	_, ok := supportedCurrencies[code]
	if !ok {
		return nil, fmt.Errorf("unsupported currency: %s", code)
	}

	return &Currency{
		code: code,
	}, nil
}

func (c Currency) Code() string {
	return c.code
}
