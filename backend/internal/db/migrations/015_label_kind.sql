-- Nature sémantique des opérations interprétées par le normaliseur :
-- '' (aucune) | savings | tax | benefit | health. Permet à l'interface
-- d'afficher une icône discrète (💰 Versement PEL, 🏛️ Cotisation URSSAF)
-- et au SQL de filtrer par nature si besoin.
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS label_kind TEXT NOT NULL DEFAULT '';
