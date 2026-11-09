package payment

import "errors"

var ErrPaymentCannotBeProcessed = errors.New("payment cannot be processed")
var ErrPaymentCannotBeCompleted = errors.New("payment cannot be completed")
var ErrPaymentCannotBeCancelled = errors.New("payment cannot be cancelled")
var ErrPaymentCannotBeFailed = errors.New("payment cannot be failed")

var ErrInvalidPaymentAmount = errors.New("invalid payment amount")
var ErrInvalidPaymentID = errors.New("invalid payment id")
var ErrInvalidPaymentUserID = errors.New("invalid payment user_id")

type PaymentStatus string

const (
	PaymentStatusPending    PaymentStatus = "PENDING"
	PaymentStatusProcessing PaymentStatus = "PROCESSING"
	PaymentStatusCompleted  PaymentStatus = "COMPLETED"
	PaymentStatusFailed     PaymentStatus = "FAILED"
	PaymentStatusCancelled  PaymentStatus = "CANCELLED"
)

type Payment struct {
	id     string
	userID string
	money  Money
	status PaymentStatus
}

func NewPayment(id string, userID string, money Money) (*Payment, error) {
	var payment Payment

	if id == "" {
		return nil, ErrInvalidPaymentID
	}

	if userID == "" {
		return nil, ErrInvalidPaymentUserID
	}

	payment.id = id
	payment.userID = userID
	payment.money = money
	payment.status = PaymentStatusPending

	return &payment, nil
}

func (p *Payment) Complete() error {
	if p.status != PaymentStatusProcessing {
		return ErrPaymentCannotBeCompleted
	}

	p.status = PaymentStatusCompleted
	return nil
}

func (p *Payment) Cancel() error {
	if p.status != PaymentStatusPending {
		return ErrPaymentCannotBeCancelled
	}

	p.status = PaymentStatusCancelled
	return nil
}

func (p *Payment) MoveToProcessing() error {
	if p.status != PaymentStatusPending {
		return ErrPaymentCannotBeProcessed
	}

	p.status = PaymentStatusProcessing
	return nil
}

func (p *Payment) Fail() error {
	if p.status != PaymentStatusProcessing {
		return ErrPaymentCannotBeFailed
	}

	p.status = PaymentStatusFailed
	return nil
}

func (p *Payment) ID() string {
	return p.id
}

func (p *Payment) UserID() string {
	return p.userID
}

func (p *Payment) Amount() int64 {
	return p.money.Amount()
}

func (p *Payment) Currency() Currency {
	return p.money.Currency()
}

func (p *Payment) Status() PaymentStatus {
	return p.status
}
