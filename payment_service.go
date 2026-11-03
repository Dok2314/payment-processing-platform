package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

var ErrPaymentNotFound = errors.New("payment not found")
var ErrInvalidPayment = errors.New("invalid payment")
var ErrIdempotencyKeyReuse = errors.New("idempotency key reused with different payload")

var ErrPaymentCannotBeCancelled = errors.New("payment cannot be cancelled")
var ErrPaymentCannotBeProcessed = errors.New("payment cannot be processed")
var ErrPaymentCannotBeCompleted = errors.New("payment cannot be completed")
var ErrPaymentCannotBeFailed = errors.New("payment cannot be failed")

type PaymentRepository interface {
	CreateIdempotent(ctx context.Context, key string, payment Payment, payloadHash string) (Payment, error)
	Get(ctx context.Context, id string) (Payment, error)
	CancelPending(ctx context.Context, id string) (Payment, error)
	MoveToProcessing(ctx context.Context, id string) (Payment, error)
	Complete(ctx context.Context, id string) (Payment, error)
	Fail(ctx context.Context, id string) (Payment, error)
}

type PaymentService struct {
	store PaymentRepository
}

func NewPaymentService(store PaymentRepository) *PaymentService {
	return &PaymentService{
		store: store,
	}
}

func (s *PaymentService) Create(ctx context.Context, key string, payment PaymentRequest) (Payment, error) {
	if err := validatePayment(payment); err != nil {
		return Payment{}, err
	}

	hash, err := payloadHash(payment)
	if err != nil {
		return Payment{}, fmt.Errorf("encode payment request: %w", err)
	}

	amount, err := ParseAmount(payment.Amount, payment.Currency)
	if err != nil {
		return Payment{}, fmt.Errorf("%w: %v", ErrInvalidPayment, err)
	}

	createdPayment := Payment{
		UserID:   payment.UserID,
		Amount:   amount,
		Currency: payment.Currency,
		Status:   PaymentStatusPending,
	}

	createdPayment.ID = uuid.NewString()

	return s.store.CreateIdempotent(ctx, key, createdPayment, hash)
}

func (s *PaymentService) Get(ctx context.Context, id string) (Payment, error) {
	return s.store.Get(ctx, id)
}

func (s *PaymentService) Cancel(ctx context.Context, id string) (Payment, error) {
	return s.store.CancelPending(ctx, id)
}

func validatePayment(payment PaymentRequest) error {
	if payment.UserID == "" {
		return fmt.Errorf("%w: user_id is required", ErrInvalidPayment)
	}
	if payment.Currency == "" {
		return fmt.Errorf("%w: currency is required", ErrInvalidPayment)
	}
	return nil
}

func (s *PaymentService) MoveToProcessing(ctx context.Context, id string) (Payment, error) {
	return s.store.MoveToProcessing(ctx, id)
}

func (s *PaymentService) Complete(ctx context.Context, id string) (Payment, error) {
	return s.store.Complete(ctx, id)
}

func (s *PaymentService) Fail(ctx context.Context, id string) (Payment, error) {
	return s.store.Fail(ctx, id)
}
