package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sync"
)

type PaymentStatus string

const (
	PaymentStatusPending    PaymentStatus = "PENDING"
	PaymentStatusProcessing PaymentStatus = "PROCESSING"
	PaymentStatusCompleted  PaymentStatus = "COMPLETED"
	PaymentStatusFailed     PaymentStatus = "FAILED"
	PaymentStatusCancelled  PaymentStatus = "CANCELLED"
)

type PaymentRequest struct {
	UserID   string `json:"user_id"`
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

type Payment struct {
	ID       string        `json:"id"`
	UserID   string        `json:"user_id"`
	Amount   int64         `json:"amount"`
	Currency string        `json:"currency"`
	Status   PaymentStatus `json:"status"`
}

type PaymentStore struct {
	mu              sync.Mutex
	payments        map[string]Payment           // paymentID → Payment
	idempotencyKeys map[string]IdempotencyRecord // idempotencyKey → IdempotencyRecord
}

type IdempotencyRecord struct {
	PaymentID   string
	PayloadHash string
}

func NewPaymentStore() *PaymentStore {
	return &PaymentStore{
		payments:        make(map[string]Payment),
		idempotencyKeys: make(map[string]IdempotencyRecord),
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

func (s *PaymentStore) CancelPending(id string) (Payment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	payment, ok := s.payments[id]
	if !ok {
		return Payment{}, ErrPaymentNotFound
	}

	if payment.Status != PaymentStatusPending {
		return Payment{}, ErrPaymentCannotBeCancelled
	}

	payment.Status = PaymentStatusCancelled
	s.payments[id] = payment

	return payment, nil
}

func (s *PaymentStore) CreateIdempotent(
	key string,
	payment Payment,
	payloadHash string,
) (Payment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ir, ok := s.idempotencyKeys[key]
	if ok {
		if payloadHash != ir.PayloadHash {
			return Payment{}, errors.New("idempotency key reused with different payload")
		}

		p, exists := s.payments[ir.PaymentID]
		if !exists {
			return Payment{}, errors.New("idempotency key references missing payment")
		}
		return p, nil
	}

	s.payments[payment.ID] = payment
	s.idempotencyKeys[key] = IdempotencyRecord{
		PaymentID:   payment.ID,
		PayloadHash: payloadHash,
	}
	return payment, nil
}

func (s *PaymentStore) UpdatePending(payment Payment) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.payments[payment.ID]
	if !ok {
		return ErrPaymentNotFound
	}

	if existing.Status != PaymentStatusPending {
		return ErrPaymentCannotBeUpdated
	}

	payment.Status = existing.Status
	s.payments[payment.ID] = payment

	return nil
}

func payloadHash(payment PaymentRequest) (string, error) {
	data, err := json.Marshal(payment)
	if err != nil {
		return "", err
	}

	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
