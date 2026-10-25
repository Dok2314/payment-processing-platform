package main

import (
	"errors"
	"sync"
)

type PaymentRequest struct {
	UserID   string  `json:"user_id"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

type Payment struct {
	ID       string  `json:"id"`
	UserID   string  `json:"user_id"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

type PaymentStore struct {
	mu              sync.Mutex
	payments        map[string]Payment // paymentID → Payment
	idempotencyKeys map[string]string  // idempotencyKey → paymentID
}

func NewPaymentStore() *PaymentStore {
	return &PaymentStore{
		payments:        make(map[string]Payment),
		idempotencyKeys: make(map[string]string),
	}
}

func (s *PaymentStore) Save(payment Payment) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.payments[payment.ID] = payment
}

func (s *PaymentStore) Get(id string) (Payment, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	payment, ok := s.payments[id]
	return payment, ok
}

func (s *PaymentStore) Update(payment Payment) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.payments[payment.ID]; !ok {
		return false
	}

	s.payments[payment.ID] = payment
	return true
}

func (s *PaymentStore) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.payments[id]; !ok {
		return false
	}

	delete(s.payments, id)
	return true
}

func (s *PaymentStore) CreateIdempotent(key string, payment Payment) (Payment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pid, ok := s.idempotencyKeys[key]
	if ok {
		p, exists := s.payments[pid]
		if !exists {
			return Payment{}, errors.New("idempotency key references missing payment")
		}
		return p, nil
	}

	s.payments[payment.ID] = payment
	s.idempotencyKeys[key] = payment.ID
	return payment, nil
}
