CREATE TABLE IF NOT EXISTS categories (
    id          SERIAL PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,
    color       TEXT NOT NULL DEFAULT '#8896A6'
);

CREATE TABLE IF NOT EXISTS rules (
    id          SERIAL PRIMARY KEY,
    pattern     TEXT NOT NULL,              -- sous-chaîne insensible à la casse ou regex si préfixée par "re:"
    category_id INT  NOT NULL REFERENCES categories(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS statements (
    id              SERIAL PRIMARY KEY,
    filename        TEXT NOT NULL,
    period_start    DATE,
    period_end      DATE,
    opening_balance NUMERIC(12,2),
    closing_balance NUMERIC(12,2),
    parser          TEXT NOT NULL DEFAULT 'local',   -- local | claude
    validated       BOOLEAN NOT NULL DEFAULT FALSE,  -- somme des transactions == delta de solde
    uploaded_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS transactions (
    id           SERIAL PRIMARY KEY,
    statement_id INT REFERENCES statements(id) ON DELETE CASCADE,
    tx_date      DATE NOT NULL,
    label        TEXT NOT NULL,
    amount       NUMERIC(12,2) NOT NULL,             -- négatif = débit, positif = crédit
    category_id  INT REFERENCES categories(id) ON DELETE SET NULL,
    hash         TEXT NOT NULL UNIQUE                -- déduplication inter-relevés
);
CREATE INDEX IF NOT EXISTS idx_tx_date ON transactions(tx_date);
CREATE INDEX IF NOT EXISTS idx_tx_category ON transactions(category_id);

CREATE TABLE IF NOT EXISTS analyses (
    id         SERIAL PRIMARY KEY,
    month      TEXT NOT NULL UNIQUE,                 -- YYYY-MM
    content    TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Catégories par défaut
INSERT INTO categories (name, color) VALUES
    ('Courses',        '#4CAF7D'),
    ('Logement',       '#5B7FD4'),
    ('Transport',      '#D4A24C'),
    ('Restaurants',    '#D46A5B'),
    ('Abonnements',    '#9B6BD4'),
    ('Santé',          '#4CBFBF'),
    ('Loisirs',        '#D45BA6'),
    ('Salaire',        '#3E9E6E'),
    ('Virements',      '#8896A6'),
    ('Frais bancaires','#B0574A')
ON CONFLICT (name) DO NOTHING;

-- Règles de base (à enrichir depuis l'interface)
INSERT INTO rules (pattern, category_id)
SELECT p, c.id FROM (VALUES
    ('CARREFOUR',   'Courses'),
    ('LECLERC',     'Courses'),
    ('AUCHAN',      'Courses'),
    ('LIDL',        'Courses'),
    ('INTERMARCHE', 'Courses'),
    ('MONOPRIX',    'Courses'),
    ('LOYER',       'Logement'),
    ('EDF',         'Logement'),
    ('ENGIE',       'Logement'),
    ('SNCF',        'Transport'),
    ('UBER',        'Transport'),
    ('TOTALENERGIES','Transport'),
    ('re:\yESSO\y',   'Transport'),
    ('NETFLIX',     'Abonnements'),
    ('SPOTIFY',     'Abonnements'),
    ('AMAZON PRIME','Abonnements'),
    ('FREE MOBILE', 'Abonnements'),
    ('ORANGE',      'Abonnements'),
    ('PHARMACIE',   'Santé'),
    ('MCDONALD',    'Restaurants'),
    ('DELIVEROO',   'Restaurants'),
    ('UBER EATS',   'Restaurants'),
    ('SALAIRE',     'Salaire'),
    ('VIREMENT',    'Virements'),
    ('COTISATION',  'Frais bancaires'),
    ('FRAIS',       'Frais bancaires')
) AS v(p, cat)
JOIN categories c ON c.name = v.cat
ON CONFLICT DO NOTHING;
