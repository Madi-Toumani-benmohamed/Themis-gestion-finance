-- Catégories de « transfert interne » : les virements entre ses propres comptes
-- (courant ↔ Livret A, etc.) ne sont ni des dépenses ni des revenus. On marque
-- la ou les catégories concernées d'un drapeau, et tous les calculs de flux
-- (dépenses, revenus, budgets, pouls, journal, reste à vivre) les excluent.
ALTER TABLE domains ADD COLUMN IF NOT EXISTS is_transfer BOOLEAN NOT NULL DEFAULT FALSE;
