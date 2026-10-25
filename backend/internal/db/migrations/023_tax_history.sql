CREATE TABLE IF NOT EXISTS tax_history (
    id           BIGSERIAL PRIMARY KEY,
    tax_id       BIGINT NOT NULL REFERENCES taxes(id) ON DELETE CASCADE,
    paid_before  BOOLEAN NOT NULL,
    paid_after   BOOLEAN NOT NULL,
    changed_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    changed_by   BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE
);