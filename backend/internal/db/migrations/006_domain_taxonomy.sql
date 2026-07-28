-- ============================================================================
-- Bascule : le DOMAINE devient la taxonomie unique. La catégorie disparaît.
-- Les règles automatiques et les budgets pointent désormais vers un domaine.
-- ============================================================================

-- 1. S'assurer que les domaines de vie existent (idempotent avec 005)
INSERT INTO domains (name, parent_id, color) VALUES
    ('Logement',       NULL, '#175C46'),
    ('Vie pro',        NULL, '#3D6B99'),
    ('Vie perso',      NULL, '#8C6D2F'),
    ('Vie amoureuse',  NULL, '#A63A28'),
    ('Famille',        NULL, '#6E4E8C'),
    ('Loisirs',        NULL, '#4E8578')
ON CONFLICT (name, parent_id) DO NOTHING;

-- 2. Règles : ajouter domain_id, migrer les motifs vers des domaines de vie
ALTER TABLE rules ADD COLUMN IF NOT EXISTS domain_id BIGINT REFERENCES domains(id) ON DELETE CASCADE;

-- Reclassement des règles existantes vers les domaines de vie (mapping net).
-- Les motifs sans domaine de vie évident (transport, abonnements…) tombent en
-- "Vie perso" par défaut — l'utilisateur affinera.
UPDATE rules r SET domain_id = d.id
FROM categories c, domains d
WHERE r.category_id = c.id AND d.parent_id IS NULL AND d.name = CASE c.name
    WHEN 'Logement'        THEN 'Logement'
    WHEN 'Salaire'         THEN 'Vie pro'
    WHEN 'Restaurants'     THEN 'Loisirs'
    WHEN 'Abonnements'     THEN 'Loisirs'
    WHEN 'Santé'           THEN 'Vie perso'
    WHEN 'Courses'         THEN 'Vie perso'
    WHEN 'Transport'       THEN 'Vie perso'
    WHEN 'Virements'       THEN 'Vie perso'
    WHEN 'Frais bancaires' THEN 'Vie perso'
    ELSE 'Vie perso'
END;

-- Les règles qui n'ont pas pu être mappées (aucune) sont supprimées ; on garde
-- celles avec un domaine. On abandonne l'ancienne colonne category_id.
DELETE FROM rules WHERE domain_id IS NULL;
ALTER TABLE rules DROP COLUMN IF EXISTS category_id;
ALTER TABLE rules ALTER COLUMN domain_id SET NOT NULL;

-- 3. Transactions : le domaine hérite de la catégorie quand il est absent,
-- via les règles n'ayant pas encore tourné — on fait un mapping direct simple.
UPDATE transactions t SET domain_id = d.id
FROM categories c, domains d
WHERE t.domain_id IS NULL AND t.category_id = c.id AND d.parent_id IS NULL AND d.name = CASE c.name
    WHEN 'Logement'        THEN 'Logement'
    WHEN 'Salaire'         THEN 'Vie pro'
    WHEN 'Restaurants'     THEN 'Loisirs'
    WHEN 'Abonnements'     THEN 'Loisirs'
    WHEN 'Santé'           THEN 'Vie perso'
    WHEN 'Courses'         THEN 'Vie perso'
    WHEN 'Transport'       THEN 'Vie perso'
    WHEN 'Virements'       THEN 'Vie perso'
    WHEN 'Frais bancaires' THEN 'Vie perso'
    ELSE 'Vie perso'
END;

-- 4. Budgets : basculer sur le domaine
ALTER TABLE budgets ADD COLUMN IF NOT EXISTS domain_id BIGINT REFERENCES domains(id) ON DELETE CASCADE;
UPDATE budgets b SET domain_id = d.id
FROM categories c, domains d
WHERE b.category_id = c.id AND d.parent_id IS NULL AND d.name = CASE c.name
    WHEN 'Logement'        THEN 'Logement'
    WHEN 'Salaire'         THEN 'Vie pro'
    WHEN 'Restaurants'     THEN 'Loisirs'
    WHEN 'Abonnements'     THEN 'Loisirs'
    WHEN 'Santé'           THEN 'Vie perso'
    WHEN 'Courses'         THEN 'Vie perso'
    WHEN 'Transport'       THEN 'Vie perso'
    ELSE 'Vie perso'
END;
DELETE FROM budgets WHERE domain_id IS NULL;
ALTER TABLE budgets DROP COLUMN IF EXISTS category_id;
-- un seul budget par domaine
ALTER TABLE budgets ADD CONSTRAINT budgets_domain_unique UNIQUE (domain_id);

-- 5. Transactions : abandonner la catégorie
DROP INDEX IF EXISTS idx_tx_category;
ALTER TABLE transactions DROP COLUMN IF EXISTS category_id;

-- 6. La table categories n'est plus utilisée
DROP TABLE IF EXISTS categories CASCADE;
