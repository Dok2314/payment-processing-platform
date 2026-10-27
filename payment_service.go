package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

var ErrPaymentNotFound = errors.New("payment not found")
var ErrInvalidPayment = errors.New("invalid payment")
var ErrPaymentCannotBeCancelled = errors.New("payment cannot be cancelled")
var ErrPaymentCannotBeUpdated = errors.New("payment cannot be updated")

type PaymentRepository interface {
	CreateIdempotent(key string, payment Payment, payloadHash string) (Payment, error)
	Get(ctx context.Context, id string) (Payment, error)
	UpdatePending(payment Payment) error
	CancelPending(id string) (Payment, error)
}

type PaymentService struct {
	store PaymentRepository
}

func NewPaymentService(store PaymentRepository) *PaymentService {
	return &PaymentService{
		store: store,
	}
}

func (s *PaymentService) Create(key string, payment PaymentRequest) (Payment, error) {
	if err := validatePayment(payment); err != nil {
		return Payment{}, err
	}

	hash, err := payloadHash(payment)
	if err != nil {
		return Payment{}, fmt.Errorf("encode payment request: %w", err)
	}

	createdPayment := Payment{
		UserID:   payment.UserID,
		Amount:   payment.Amount,
		Currency: payment.Currency,
		Status:   PaymentStatusPending,
	}

	createdPayment.ID = uuid.NewString()

	return s.store.CreateIdempotent(key, createdPayment, hash)
}

func (s *PaymentService) Get(ctx context.Context, id string) (Payment, error) {
	return s.store.Get(ctx, id)
}

func (s *PaymentService) Update(id string, payment PaymentRequest) (Payment, error) {
	if err := validatePayment(payment); err != nil {
		return Payment{}, err
	}

	updatedPayment := Payment{
		ID:       id,
		UserID:   payment.UserID,
		Amount:   payment.Amount,
		Currency: payment.Currency,
	}

	if err := s.store.UpdatePending(updatedPayment); err != nil {
		return Payment{}, err
	}

	updatedPayment.Status = PaymentStatusPending
	return updatedPayment, nil
}

func (s *PaymentService) Cancel(id string) (Payment, error) {
	return s.store.CancelPending(id)
}

func validatePayment(payment PaymentRequest) error {
	if payment.Amount <= 0 {
		return fmt.Errorf("%w: amount must be greater than zero", ErrInvalidPayment)
	}
	if payment.UserID == "" {
		return fmt.Errorf("%w: user_id is required", ErrInvalidPayment)
	}
	if payment.Currency == "" {
		return fmt.Errorf("%w: currency is required", ErrInvalidPayment)
	}
	return nil
}
