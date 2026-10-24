-- User decisions only. Transaction amounts, classifications and source records
-- remain untouched. Rollback is deliberately outside the embedded migrations.
CREATE TABLE IF NOT EXISTS recurring_reviews (
    owner_id       BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    recurrence_key TEXT NOT NULL CHECK (recurrence_key ~ '^[a-f0-9]{64}$'),
    status         TEXT NOT NULL DEFAULT 'detected' CHECK (status IN ('detected', 'confirmed', 'ignored')),
    nature         TEXT CHECK (nature IN ('subscription', 'fixed_charge', 'variable_recurring', 'transfer_savings')),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (owner_id, recurrence_key)
);
