CREATE TABLE IF NOT EXISTS credit_payments (
    id          BIGSERIAL PRIMARY KEY,
    credit_id   BIGINT NOT NULL REFERENCES credits(id) ON DELETE CASCADE,
    amount_cents  BIGINT NOT NULL CHECK (amount_cents > 0),
    payment_date DATE NOT NULL,
    note        TEXT NOT NULL DEFAULT '',
    owner_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_credit_payments_credit ON credit_payments(credit_id);
CREATE INDEX IF NOT EXISTS idx_credit_payments_owner ON credit_payments(owner_id);