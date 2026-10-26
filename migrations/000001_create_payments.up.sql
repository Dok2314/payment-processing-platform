CREATE TABLE payments (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL,
    amount     BIGINT NOT NULL CHECK (amount > 0),
    currency   VARCHAR(3) NOT NULL,
    status     TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);