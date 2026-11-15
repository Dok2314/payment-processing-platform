package payment

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	domainpayment "payment-processing-platform/internal/domain/payment"
)

func calculatePayloadHash(payment *domainpayment.Payment) (string, error) {
	payload := struct {
		UserID   string `json:"user_id"`
		Amount   int64  `json:"amount"`
		Currency string `json:"currency"`
	}{
		UserID:   payment.UserID(),
		Amount:   payment.Amount(),
		Currency: payment.Currency().Code(),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}
