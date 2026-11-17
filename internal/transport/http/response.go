package http

import (
	"fmt"

	domainpayment "payment-processing-platform/internal/domain/payment"
)

type PaymentResponse struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

func newPaymentResponse(payment *domainpayment.Payment) PaymentResponse {
	return PaymentResponse{
		ID:       payment.ID(),
		Status:   string(payment.Status()),
		Amount:   fmt.Sprintf("%.2f", float64(payment.Amount())/100),
		Currency: payment.Currency().Code(),
	}
}
