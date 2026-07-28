-- Un alias peut être un motif : au lieu de matcher un marchand exact, il
-- s'applique à toute transaction dont le libellé CONTIENT la clé. Utile pour
-- les marchands à référence variable (« Acheel France Ref-xxxx » où xxxx change
-- à chaque transaction) : un seul motif « Acheel France » les renomme tous.
ALTER TABLE merchant_aliases ADD COLUMN IF NOT EXISTS is_pattern BOOLEAN NOT NULL DEFAULT FALSE;
