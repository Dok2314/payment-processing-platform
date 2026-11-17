package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
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
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, applicationpayment.ErrPaymentNotFound
		}

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
	tag, err := p.pool.Exec(ctx, `
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
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return applicationpayment.ErrPaymentNotFound
	}

	return nil
}

func (p *PaymentRepository) CreateIdempotent(
	ctx context.Context,
	key string,
	payment *domainpayment.Payment,
	payloadHash string,
) (*domainpayment.Payment, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`,
		key,
	)
	if err != nil {
		return nil, err
	}

	var existingPaymentID string
	var existingPayloadHash string

	err = tx.QueryRow(ctx, `
		SELECT payment_id, payload_hash
		FROM idempotency_keys
		WHERE key = $1
	`, key).Scan(&existingPaymentID, &existingPayloadHash)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	if err == nil {
		if existingPayloadHash != payloadHash {
			return nil, applicationpayment.ErrIdempotencyKeyReuse
		}

		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}

		return p.Get(ctx, existingPaymentID)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO payments(id, user_id, amount, currency, status)
		VALUES($1, $2, $3, $4, $5)
	`,
		payment.ID(),
		payment.UserID(),
		payment.Amount(),
		payment.Currency().Code(),
		payment.Status(),
	)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO idempotency_keys(key, payment_id, payload_hash)
		VALUES($1, $2, $3)
	`,
		key,
		payment.ID(),
		payloadHash,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return payment, nil
}
