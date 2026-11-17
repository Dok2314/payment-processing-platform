package payment

import "errors"

var ErrIdempotencyKeyReuse = errors.New("idempotency key reused with different payload")
var ErrPaymentNotFound = errors.New("payment not found")
