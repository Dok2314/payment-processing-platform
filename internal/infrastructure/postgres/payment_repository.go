package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	applicationpayment "payment-processing-platform/internal/application/payment"
	domainpayment "payment-processing-platform/internal/domain/payment"
)

type PaymentRepository struct {
	pool *pgxpool.Pool
}

var _ applicationpayment.PaymentRepository = (*PaymentRepository)(nil)

func NewPaymentRepository(pool *pgxpool.Pool) *PaymentRepository {
	return &PaymentRepository{
		pool: pool,
	}
}

func (p *PaymentRepository) Save(ctx context.Context, payment *domainpayment.Payment) error {
	_, err := p.pool.Exec(ctx, `
		INSERT INTO payments(id, user_id, amount, currency, status)
		VALUES($1, $2, $3, $4, $5)
	`,
		payment.ID(),
		payment.UserID(),
		payment.Amount(),
		payment.Currency().Code(),
		payment.Status(),
	)

	return err
}

func (p *PaymentRepository) Get(ctx context.Context, id string) (*domainpayment.Payment, error) {
	row := p.pool.QueryRow(ctx, `
		SELECT id, user_id, amount, currency, status
		FROM payments
		WHERE id = $1
	`, id)

	var paymentID string
	var userID string
	var amount int64
	var currencyCode string
	var status string

	err := row.Scan(
		&paymentID,
		&userID,
		&amount,
		&currencyCode,
		&status,
	)
	if err != nil {
		return nil, err
	}

	currency, err := domainpayment.NewCurrency(currencyCode)
	if err != nil {
		return nil, err
	}

	money, err := domainpayment.NewMoney(amount, *currency)
	if err != nil {
		return nil, err
	}

	payment, err := domainpayment.ReconstitutePayment(
		paymentID,
		userID,
		*money,
		domainpayment.PaymentStatus(status),
	)
	if err != nil {
		return nil, err
	}

	return payment, nil
}

func (p *PaymentRepository) Update(ctx context.Context, payment *domainpayment.Payment) error {
	_, err := p.pool.Exec(ctx, `	
		UPDATE payments
		SET user_id = $1,
		amount = $2,
		currency = $3,
		status = $4,
		updated_at = NOW()
		WHERE id = $5
	`,
		payment.UserID(),
		payment.Amount(),
		payment.Currency().Code(),
		payment.Status(),
		payment.ID(),
	)

	return err
}
