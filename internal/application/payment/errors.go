package payment

import "errors"

var ErrIdempotencyKeyReuse = errors.New("idempotency key reused with different payload")
