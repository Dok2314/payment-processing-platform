package payment

import (
	"errors"

	domainpayment "payment-processing-platform/internal/domain/payment"
)

var ErrIdempotencyKeyReuse = errors.New("idempotency key reused with different payload")
var ErrPaymentNotFound = errors.New("payment not found")

// Domain errors
var ErrPaymentCannotBeProcessed = domainpayment.ErrPaymentCannotBeProcessed
var ErrPaymentCannotBeCompleted = domainpayment.ErrPaymentCannotBeCompleted
var ErrPaymentCannotBeCancelled = domainpayment.ErrPaymentCannotBeCancelled
var ErrPaymentCannotBeFailed = domainpayment.ErrPaymentCannotBeFailed

var ErrInvalidPaymentAmount = domainpayment.ErrInvalidPaymentAmount
var ErrInvalidPaymentID = domainpayment.ErrInvalidPaymentID
var ErrInvalidPaymentUserID = domainpayment.ErrInvalidPaymentUserID
