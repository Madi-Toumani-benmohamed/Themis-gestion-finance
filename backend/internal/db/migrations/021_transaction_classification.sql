-- Add provenance only. Historical categories and raw bank records are not rewritten.
ALTER TABLE transactions ADD COLUMN classification_source TEXT NOT NULL DEFAULT 'unknown'
    CHECK (classification_source IN ('unknown','rule','manual','ai'));
CREATE INDEX IF NOT EXISTS idx_transactions_owner_date_id ON transactions(owner_id, tx_date DESC, id DESC);

-- Unverified proposals never substitute for the transaction's assigned category.
CREATE TABLE transaction_suggestions (
    transaction_id BIGINT PRIMARY KEY REFERENCES transactions(id) ON DELETE CASCADE,
    owner_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    domain_id BIGINT NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
    tags TEXT[] NOT NULL DEFAULT '{}',
    source TEXT NOT NULL DEFAULT 'ai' CHECK(source='ai'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_transaction_suggestions_owner ON transaction_suggestions(owner_id);
