package main

import "sync"

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
	mu       sync.Mutex
	payments map[string]Payment
}

func NewPaymentStore() *PaymentStore {
	return &PaymentStore{
		payments: make(map[string]Payment),
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
