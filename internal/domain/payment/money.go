package payment

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
