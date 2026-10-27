CREATE TABLE idempotency_keys(
    key TEXT PRIMARY KEY,
    payment_id TEXT NOT NULL REFERENCES payments(id),
    payload_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);