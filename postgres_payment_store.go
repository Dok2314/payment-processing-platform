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

	row := ps.pool.QueryRow(ctx, `
		SELECT id, user_id, amount, currency, status 
		FROM payments
		WHERE id = $1
	`, id)

	err := row.Scan(&payment.ID, &payment.UserID, &payment.Amount, &payment.Currency, &payment.Status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Payment{}, ErrPaymentNotFound
		}

		return Payment{}, fmt.Errorf("get payment: %w", err)
	}

	return payment, nil
}
