package main

import (
	"errors"

	"github.com/google/uuid"
)

type PaymentService struct {
	store *PaymentStore
}

func NewPaymentService(store *PaymentStore) *PaymentService {
	return &PaymentService{
		store: store,
	}
}

func (s *PaymentService) Create(payment PaymentRequest) (Payment, error) {
	if payment.Amount <= 0 {
		return Payment{}, errors.New("amount must be greater than zero")
	}
	if payment.UserID == "" {
		return Payment{}, errors.New("user_id is required")
	}
	if payment.Currency == "" {
		return Payment{}, errors.New("currency is required")
	}

	createdPayment := Payment{
		UserID:   payment.UserID,
		Amount:   payment.Amount,
		Currency: payment.Currency,
	}

	createdPayment.ID = uuid.NewString()

	s.store.Save(createdPayment)

	return createdPayment, nil
}

func (s *PaymentService) Get(id string) (Payment, bool) {
	return s.store.Get(id)
}

func (s *PaymentService) Update(id string, payment PaymentRequest) (Payment, error) {
	if payment.Amount <= 0 {
		return Payment{}, errors.New("amount must be greater than zero")
	}
	if payment.UserID == "" {
		return Payment{}, errors.New("user_id is required")
	}
	if payment.Currency == "" {
		return Payment{}, errors.New("currency is required")
	}

	updatedPayment := Payment{
		ID:       id,
		UserID:   payment.UserID,
		Amount:   payment.Amount,
		Currency: payment.Currency,
	}

	if !s.store.Update(updatedPayment) {
		return Payment{}, errors.New("payment not found")
	}

	return updatedPayment, nil
}

func (s *PaymentService) Delete(id string) bool {
	return s.store.Delete(id)
}
