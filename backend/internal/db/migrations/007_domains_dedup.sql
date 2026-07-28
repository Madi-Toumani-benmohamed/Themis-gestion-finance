-- La contrainte UNIQUE (name, parent_id) ne protège PAS les domaines racines :
-- en SQL, deux NULL ne sont jamais "égaux", donc (name, NULL) pouvait exister
-- en double. On déduplique puis on pose un index unique partiel correct.

-- 1. Réaffecter les transactions/règles/budgets vers le plus petit id de chaque nom
UPDATE transactions t SET domain_id = keep.id
FROM domains d JOIN (SELECT name, min(id) AS id FROM domains WHERE parent_id IS NULL GROUP BY name) keep
     ON keep.name = d.name
WHERE t.domain_id = d.id AND d.parent_id IS NULL AND d.id <> keep.id;

UPDATE rules r SET domain_id = keep.id
FROM domains d JOIN (SELECT name, min(id) AS id FROM domains WHERE parent_id IS NULL GROUP BY name) keep
     ON keep.name = d.name
WHERE r.domain_id = d.id AND d.parent_id IS NULL AND d.id <> keep.id;

UPDATE budgets b SET domain_id = keep.id
FROM domains d JOIN (SELECT name, min(id) AS id FROM domains WHERE parent_id IS NULL GROUP BY name) keep
     ON keep.name = d.name
WHERE b.domain_id = d.id AND d.parent_id IS NULL AND d.id <> keep.id
  AND NOT EXISTS (SELECT 1 FROM budgets b2 WHERE b2.domain_id = keep.id);

-- 2. Supprimer les doublons (on garde le plus petit id par nom)
DELETE FROM domains d USING (
    SELECT name, min(id) AS id FROM domains WHERE parent_id IS NULL GROUP BY name
) keep
WHERE d.parent_id IS NULL AND d.name = keep.name AND d.id <> keep.id;

-- 3. Index unique partiel : un seul domaine racine par nom, désormais garanti
CREATE UNIQUE INDEX IF NOT EXISTS idx_domains_root_name
    ON domains (name) WHERE parent_id IS NULL;
