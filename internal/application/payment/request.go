package payment

type PaymentRequest struct {
	UserID   string `json:"user_id"`
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}
