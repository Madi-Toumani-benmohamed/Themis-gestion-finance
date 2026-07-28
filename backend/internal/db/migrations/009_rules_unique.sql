-- Empêcher les règles en double. Un même motif (insensible à la casse) ne doit
-- exister qu'une fois. On dédoublonne l'existant, puis on pose un index unique.

-- 1. Réaffecter les règles dupliquées vers la plus ancienne (plus petit id)
--    de chaque motif normalisé, puis supprimer les doublons.
DELETE FROM rules r USING (
    SELECT lower(pattern) AS lp, min(id) AS keep
    FROM rules GROUP BY lower(pattern)
) dup
WHERE lower(r.pattern) = dup.lp AND r.id <> dup.keep;

-- 2. Index unique insensible à la casse : un seul motif par règle désormais.
CREATE UNIQUE INDEX IF NOT EXISTS idx_rules_pattern_ci
    ON rules (lower(pattern));
