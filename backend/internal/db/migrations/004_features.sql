-- Multi-comptes ---------------------------------------------------------
CREATE TABLE IF NOT EXISTS accounts (
    id            BIGSERIAL PRIMARY KEY,
    name          TEXT NOT NULL UNIQUE,
    kind          TEXT NOT NULL DEFAULT 'courant', -- courant, livret_a, ldds, pel, bourse, autre
    interest_rate NUMERIC(5,2),                    -- % annuel, informatif
    deposit_limit NUMERIC(12,2),                   -- plafond de dépôt (livrets)
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO accounts (id, name, kind) VALUES (1, 'Compte courant', 'courant')
ON CONFLICT (id) DO NOTHING;
SELECT setval('accounts_id_seq', GREATEST((SELECT max(id) FROM accounts), 1));

ALTER TABLE statements ADD COLUMN IF NOT EXISTS account_id BIGINT NOT NULL DEFAULT 1 REFERENCES accounts(id);

-- Relevé de position manuel pour les comptes sans relevés PDF (livrets, bourse)
CREATE TABLE IF NOT EXISTS account_snapshots (
    id         BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    snap_date  DATE NOT NULL,
    balance    NUMERIC(12,2) NOT NULL,
    UNIQUE (account_id, snap_date)
);

-- Crédits & taxes ---------------------------------------------------------
CREATE TABLE IF NOT EXISTS credits (
    id              BIGSERIAL PRIMARY KEY,
    name            TEXT NOT NULL,
    monthly_amount  NUMERIC(12,2) NOT NULL CHECK (monthly_amount > 0),
    months_total    INT NOT NULL CHECK (months_total > 0),
    start_month     CHAR(7) NOT NULL, -- YYYY-MM de la première mensualité
    note            TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS taxes (
    id       BIGSERIAL PRIMARY KEY,
    name     TEXT NOT NULL,
    amount   NUMERIC(12,2) NOT NULL CHECK (amount > 0),
    due_date DATE NOT NULL,
    paid     BOOLEAN NOT NULL DEFAULT false
);

-- Domaines (hiérarchiques) & tags ----------------------------------------
CREATE TABLE IF NOT EXISTS domains (
    id        BIGSERIAL PRIMARY KEY,
    name      TEXT NOT NULL,
    parent_id BIGINT REFERENCES domains(id) ON DELETE CASCADE,
    UNIQUE (name, parent_id)
);

ALTER TABLE transactions ADD COLUMN IF NOT EXISTS domain_id BIGINT REFERENCES domains(id) ON DELETE SET NULL;

CREATE TABLE IF NOT EXISTS tags (
    id   BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);
CREATE TABLE IF NOT EXISTS transaction_tags (
    transaction_id BIGINT NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    tag_id         BIGINT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (transaction_id, tag_id)
);

-- Utilisateurs & sessions --------------------------------------------------
CREATE TABLE IF NOT EXISTS users (
    id            BIGSERIAL PRIMARY KEY,
    username      TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL, -- pbkdf2$sha256$iter$salt$hash (base64)
    role          TEXT NOT NULL DEFAULT 'user' CHECK (role IN ('admin', 'user')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS sessions (
    token      TEXT PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions (expires_at);
