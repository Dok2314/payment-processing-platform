package payment

import (
	"context"

	domainpayment "payment-processing-platform/internal/domain/payment"

	"github.com/google/uuid"
)

type PaymentRepository interface {
	CreateIdempotent(
		ctx context.Context,
		key string,
		payment *domainpayment.Payment,
		payloadHash string,
	) (*domainpayment.Payment, error)
	Get(ctx context.Context, id string) (*domainpayment.Payment, error)
	Update(ctx context.Context, payment *domainpayment.Payment) error
}

type PaymentService struct {
	repository PaymentRepository
}

func NewPaymentService(repository PaymentRepository) *PaymentService {
	return &PaymentService{
		repository: repository,
	}
}

func (s *PaymentService) Create(
	ctx context.Context,
	key string,
	req PaymentRequest,
) (*domainpayment.Payment, error) {
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

	payloadHash, err := calculatePayloadHash(payment)
	if err != nil {
		return nil, err
	}

	return s.repository.CreateIdempotent(ctx, key, payment, payloadHash)
}

func (s *PaymentService) Get(ctx context.Context, id string) (*domainpayment.Payment, error) {
	return s.repository.Get(ctx, id)
}

func (s *PaymentService) Cancel(ctx context.Context, id string) (*domainpayment.Payment, error) {
	payment, err := s.repository.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	err = payment.Cancel()
	if err != nil {
		return nil, err
	}

	err = s.repository.Update(ctx, payment)
	if err != nil {
		return nil, err
	}

	return payment, nil
}

func (s *PaymentService) Process(ctx context.Context, id string) (*domainpayment.Payment, error) {
	payment, err := s.repository.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	err = payment.MoveToProcessing()
	if err != nil {
		return nil, err
	}

	err = s.repository.Update(ctx, payment)
	if err != nil {
		return nil, err
	}

	return payment, nil
}

func (s *PaymentService) Complete(ctx context.Context, id string) (*domainpayment.Payment, error) {
	payment, err := s.repository.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	err = payment.Complete()
	if err != nil {
		return nil, err
	}

	err = s.repository.Update(ctx, payment)
	if err != nil {
		return nil, err
	}

	return payment, nil
}
