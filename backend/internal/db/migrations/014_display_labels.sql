-- Noms d'affichage : le libellé bancaire brut est fait pour les systèmes,
-- pas pour les humains. display_label est le nom lisible, merchant la clé de
-- regroupement (canonique pour les marchands connus). Le brut (label) reste
-- la source de vérité des règles de tri et de la déduplication.
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS display_label TEXT NOT NULL DEFAULT '';
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS merchant      TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_transactions_owner_merchant ON transactions(owner_id, merchant);
