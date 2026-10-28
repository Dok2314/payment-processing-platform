package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresPaymentStore struct {
	pool *pgxpool.Pool
}

func NewPostgresPaymentStore(pool *pgxpool.Pool) *PostgresPaymentStore {
	return &PostgresPaymentStore{
		pool: pool,
	}
}

func (ps *PostgresPaymentStore) Get(ctx context.Context, id string) (Payment, error) {
	var payment Payment

	err := ps.pool.QueryRow(ctx, `
		SELECT id, user_id, amount, currency, status 
		FROM payments
		WHERE id = $1
	`, id).Scan(
		&payment.ID,
		&payment.UserID,
		&payment.Amount,
		&payment.Currency,
		&payment.Status)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Payment{}, ErrPaymentNotFound
		}

		return Payment{}, fmt.Errorf("get payment: %w", err)
	}

	return payment, nil
}

func (ps *PostgresPaymentStore) CreateIdempotent(
	ctx context.Context,
	key string,
	payment Payment,
	payloadHash string,
) (Payment, error) {
	tx, err := ps.pool.Begin(ctx)
	if err != nil {
		return Payment{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO payments (id, user_id, amount, currency, status)
		VALUES ($1, $2, $3, $4, $5)
	`,
		payment.ID,
		payment.UserID,
		payment.Amount,
		payment.Currency,
		payment.Status,
	)
	if err != nil {
		return Payment{}, fmt.Errorf("insert payment: %w", err)
	}

	tag, err := tx.Exec(ctx, `
		INSERT INTO idempotency_keys (key, payment_id, payload_hash)
		VALUES ($1, $2, $3)
		ON CONFLICT (key) DO NOTHING
	`, key, payment.ID, payloadHash)
	if err != nil {
		return Payment{}, fmt.Errorf("insert idempotency key: %w", err)
	}

	if tag.RowsAffected() == 0 {
		var existingPaymentID string
		var existingPayloadHash string

		err := tx.QueryRow(ctx, `
			SELECT payment_id, payload_hash
			FROM idempotency_keys
			WHERE key = $1
		`, key).Scan(&existingPaymentID, &existingPayloadHash)
		if err != nil {
			return Payment{}, fmt.Errorf("get existing idempotency key: %w", err)
		}

		if err := tx.Rollback(ctx); err != nil {
			return Payment{}, fmt.Errorf("rollback duplicate payment: %w", err)
		}

		if existingPayloadHash != payloadHash {
			return Payment{}, errors.New("idempotency key reused with different payload")
		}

		return ps.Get(ctx, existingPaymentID)
	}

	if err := tx.Commit(ctx); err != nil {
		return Payment{}, fmt.Errorf("commit transaction: %w", err)
	}

	return payment, nil
}

func (ps *PostgresPaymentStore) UpdatePending(ctx context.Context, payment Payment) error {
	tag, err := ps.pool.Exec(ctx, `
		UPDATE payments
		SET user_id = $1,
		    amount = $2,
		    currency = $3,
		    updated_at = NOW()
		WHERE id = $4 AND status = $5
	`,
		payment.UserID,
		payment.Amount,
		payment.Currency,
		payment.ID,
		PaymentStatusPending,
	)
	if err != nil {
		return fmt.Errorf("update payment: %w", err)
	}

	if tag.RowsAffected() == 0 {
		_, err := ps.Get(ctx, payment.ID)
		if err != nil {
			return err
		}

		return ErrPaymentCannotBeUpdated
	}

	return nil
}

func (ps *PostgresPaymentStore) CancelPending(ctx context.Context, id string) (Payment, error) {
	tag, err := ps.pool.Exec(ctx, `
		UPDATE payments
		SET status = $1,
		    updated_at = NOW()
		WHERE id = $2 AND status = $3
	`, PaymentStatusCancelled, id, PaymentStatusPending)
	if err != nil {
		return Payment{}, fmt.Errorf("cancel payment: %w", err)
	}

	if tag.RowsAffected() == 0 {
		_, err := ps.Get(ctx, id)
		if err != nil {
			return Payment{}, err
		}

		return Payment{}, ErrPaymentCannotBeCancelled
	}

	return ps.Get(ctx, id)
}
