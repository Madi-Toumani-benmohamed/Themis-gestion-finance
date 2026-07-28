-- Isolation des données par utilisateur. Jusqu'ici, toutes les données étaient
-- partagées entre tous les comptes. On ajoute un propriétaire (owner_id) à
-- chaque table de données personnelles, et on attribue l'existant au premier
-- administrateur (plus petit id) pour ne rien perdre.

-- Tables « racines » qui portent directement un propriétaire.
ALTER TABLE accounts     ADD COLUMN IF NOT EXISTS owner_id BIGINT REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE statements   ADD COLUMN IF NOT EXISTS owner_id BIGINT REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS owner_id BIGINT REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE domains      ADD COLUMN IF NOT EXISTS owner_id BIGINT REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE rules        ADD COLUMN IF NOT EXISTS owner_id BIGINT REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE tags         ADD COLUMN IF NOT EXISTS owner_id BIGINT REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE budgets      ADD COLUMN IF NOT EXISTS owner_id BIGINT REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE credits      ADD COLUMN IF NOT EXISTS owner_id BIGINT REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE taxes        ADD COLUMN IF NOT EXISTS owner_id BIGINT REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE analyses     ADD COLUMN IF NOT EXISTS owner_id BIGINT REFERENCES users(id) ON DELETE CASCADE;

-- Attribuer TOUTES les données existantes au premier admin (à défaut, au plus
-- petit user id). Sans utilisateur, la base est vierge : rien à attribuer.
DO $$
DECLARE
    first_admin BIGINT;
BEGIN
    SELECT id INTO first_admin FROM users WHERE role = 'admin' ORDER BY id LIMIT 1;
    IF first_admin IS NULL THEN
        SELECT id INTO first_admin FROM users ORDER BY id LIMIT 1;
    END IF;

    IF first_admin IS NOT NULL THEN
        UPDATE accounts     SET owner_id = first_admin WHERE owner_id IS NULL;
        UPDATE statements   SET owner_id = first_admin WHERE owner_id IS NULL;
        UPDATE transactions SET owner_id = first_admin WHERE owner_id IS NULL;
        UPDATE domains      SET owner_id = first_admin WHERE owner_id IS NULL;
        UPDATE rules        SET owner_id = first_admin WHERE owner_id IS NULL;
        UPDATE tags         SET owner_id = first_admin WHERE owner_id IS NULL;
        UPDATE budgets      SET owner_id = first_admin WHERE owner_id IS NULL;
        UPDATE credits      SET owner_id = first_admin WHERE owner_id IS NULL;
        UPDATE taxes        SET owner_id = first_admin WHERE owner_id IS NULL;
        UPDATE analyses     SET owner_id = first_admin WHERE owner_id IS NULL;
    END IF;
END $$;

-- Index pour des filtres par propriétaire rapides.
CREATE INDEX IF NOT EXISTS idx_transactions_owner ON transactions(owner_id);
CREATE INDEX IF NOT EXISTS idx_statements_owner   ON statements(owner_id);
CREATE INDEX IF NOT EXISTS idx_accounts_owner     ON accounts(owner_id);
CREATE INDEX IF NOT EXISTS idx_domains_owner      ON domains(owner_id);
CREATE INDEX IF NOT EXISTS idx_rules_owner        ON rules(owner_id);
CREATE INDEX IF NOT EXISTS idx_tags_owner         ON tags(owner_id);

-- L'index unique global sur les règles devient unique PAR utilisateur
-- (deux personnes peuvent avoir chacune une règle « CARREFOUR »).
DROP INDEX IF EXISTS idx_rules_pattern_ci;
CREATE UNIQUE INDEX IF NOT EXISTS idx_rules_owner_pattern_ci ON rules (owner_id, lower(pattern));

-- De même, l'unicité des catégories racines devient par utilisateur.
DROP INDEX IF EXISTS idx_domains_root_name;
CREATE UNIQUE INDEX IF NOT EXISTS idx_domains_owner_root_name ON domains (owner_id, name) WHERE parent_id IS NULL;

-- La déduplication par hash doit être PAR utilisateur : deux personnes peuvent
-- légitimement avoir une transaction identique (même hash). On retire la
-- contrainte globale et on la remplace par une unicité (owner_id, hash).
ALTER TABLE transactions DROP CONSTRAINT IF EXISTS transactions_hash_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_transactions_owner_hash ON transactions (owner_id, hash);

-- Les tags deviennent propres à chaque utilisateur (unicité par propriétaire).
ALTER TABLE tags DROP CONSTRAINT IF EXISTS tags_name_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_tags_owner_name ON tags (owner_id, name);

-- Le nom de compte devient unique PAR utilisateur (deux personnes peuvent
-- chacune avoir un « Compte courant »).
ALTER TABLE accounts DROP CONSTRAINT IF EXISTS accounts_name_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_accounts_owner_name ON accounts (owner_id, name);

-- L'unicité des catégories (name, parent_id) était globale : elle devient
-- par utilisateur. On retire l'ancienne contrainte nommée.
ALTER TABLE domains DROP CONSTRAINT IF EXISTS domains_name_parent_id_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_domains_owner_name_parent ON domains (owner_id, name, COALESCE(parent_id, 0));

-- Les analyses IA mensuelles deviennent propres à chaque utilisateur.
ALTER TABLE analyses DROP CONSTRAINT IF EXISTS analyses_month_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_analyses_owner_month ON analyses (owner_id, month);

-- Réglages : distinguer les réglages GLOBAUX (ex. config IA gérée par l'admin,
-- owner_id NULL) des préférences PAR UTILISATEUR (ex. layout du tableau de bord).
-- On ajoute owner_id ; la clé seule n'est plus unique, c'est (owner_id, key).
ALTER TABLE settings ADD COLUMN IF NOT EXISTS owner_id BIGINT REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE settings DROP CONSTRAINT IF EXISTS settings_pkey;
ALTER TABLE settings DROP CONSTRAINT IF EXISTS settings_key_key;
DROP INDEX IF EXISTS settings_key_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_settings_owner_key ON settings (COALESCE(owner_id, 0), key);
