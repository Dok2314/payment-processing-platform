package payment

import (
	"context"

	domainpayment "payment-processing-platform/internal/domain/payment"

	"github.com/google/uuid"
)

type PaymentRepository interface {
	Save(ctx context.Context, payment *domainpayment.Payment) error
	Get(ctx context.Context, id string) (*domainpayment.Payment, error)
}

type PaymentService struct {
	repository PaymentRepository
}

func NewPaymentService(repository PaymentRepository) *PaymentService {
	return &PaymentService{
		repository: repository,
	}
}

func (s *PaymentService) Create(ctx context.Context, req PaymentRequest) (*domainpayment.Payment, error) {
	id := uuid.NewString()

	currency, err := domainpayment.NewCurrency(req.Currency)
	if err != nil {
		return nil, err
	}

	amount, err := domainpayment.ParseAmount(req.Amount, *currency)
	if err != nil {
		return nil, err
	}

	money, err := domainpayment.NewMoney(amount, *currency)
	if err != nil {
		return nil, err
	}

	payment, err := domainpayment.NewPayment(id, req.UserID, *money)
	if err != nil {
		return nil, err
	}

	if err := s.repository.Save(ctx, payment); err != nil {
		return nil, err
	}

	return payment, nil
}
