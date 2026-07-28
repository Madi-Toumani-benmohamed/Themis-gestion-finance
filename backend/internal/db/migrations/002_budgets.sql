-- Budgets mensuels par catégorie
CREATE TABLE IF NOT EXISTS budgets (
    id           BIGSERIAL PRIMARY KEY,
    category_id  BIGINT NOT NULL UNIQUE REFERENCES categories(id) ON DELETE CASCADE,
    monthly_limit NUMERIC(12,2) NOT NULL CHECK (monthly_limit > 0),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Index pour la recherche plein texte simple sur les libellés
CREATE INDEX IF NOT EXISTS idx_transactions_label ON transactions USING gin (to_tsvector('simple', label));
