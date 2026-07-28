-- Correctif : l'upsert des budgets utilise ON CONFLICT (owner_id, domain_id),
-- mais l'index unique correspondant n'avait pas été créé lors du passage à
-- l'isolation (migration 010). L'ancienne unicité portait sur (domain_id) seul,
-- ce qui, en plus, empêchait deux utilisateurs d'avoir chacun un budget sur la
-- même catégorie. On corrige les deux en une fois.
ALTER TABLE budgets DROP CONSTRAINT IF EXISTS budgets_domain_unique;
CREATE UNIQUE INDEX IF NOT EXISTS idx_budgets_owner_domain ON budgets (owner_id, domain_id);
