-- Bascule des « domaines de vie » (contexte) vers des catégories de dépense
-- classiques. On préserve les transactions déjà rangées en remappant les
-- anciens domaines vers les nouveaux au plus proche, puis on ajoute les
-- catégories manquantes et retire les anciens domaines devenus inutiles.

-- 1. Insérer les nouvelles catégories (si pas déjà présentes).
INSERT INTO domains (name, parent_id, color) VALUES
    ('Logement',                 NULL, '#4C6FBF'),
    ('Transport',                NULL, '#3F8F7B'),
    ('Alimentation',             NULL, '#C77D2E'),
    ('Santé et Bien-être',       NULL, '#D6455D'),
    ('Divertissement',           NULL, '#8A5CD1'),
    ('Épargne et Investissements', NULL, '#2E8B57'),
    ('Vie Professionnelle',      NULL, '#3D6B99'),
    ('Famille',                  NULL, '#B5548A'),
    ('Autres',                   NULL, '#8A8A99')
ON CONFLICT (name) WHERE parent_id IS NULL DO NOTHING;

-- 2. Remapper les transactions des anciens domaines vers les nouvelles catégories.
--    Correspondances directes quand elles existent, sinon « Autres ».
DO $$
DECLARE
    autres_id BIGINT;
BEGIN
    SELECT id INTO autres_id FROM domains WHERE name = 'Autres' AND parent_id IS NULL;

    -- Vie pro -> Vie Professionnelle
    UPDATE transactions SET domain_id = (SELECT id FROM domains WHERE name = 'Vie Professionnelle' AND parent_id IS NULL)
    WHERE domain_id IN (SELECT id FROM domains WHERE name = 'Vie pro' AND parent_id IS NULL);
    UPDATE rules SET domain_id = (SELECT id FROM domains WHERE name = 'Vie Professionnelle' AND parent_id IS NULL)
    WHERE domain_id IN (SELECT id FROM domains WHERE name = 'Vie pro' AND parent_id IS NULL);

    -- Loisirs -> Divertissement
    UPDATE transactions SET domain_id = (SELECT id FROM domains WHERE name = 'Divertissement' AND parent_id IS NULL)
    WHERE domain_id IN (SELECT id FROM domains WHERE name = 'Loisirs' AND parent_id IS NULL);
    UPDATE rules SET domain_id = (SELECT id FROM domains WHERE name = 'Divertissement' AND parent_id IS NULL)
    WHERE domain_id IN (SELECT id FROM domains WHERE name = 'Loisirs' AND parent_id IS NULL);

    -- Vie perso, Vie amoureuse -> Autres (pas d'équivalent direct)
    UPDATE transactions SET domain_id = autres_id
    WHERE domain_id IN (SELECT id FROM domains WHERE name IN ('Vie perso', 'Vie amoureuse') AND parent_id IS NULL);
    UPDATE rules SET domain_id = autres_id
    WHERE domain_id IN (SELECT id FROM domains WHERE name IN ('Vie perso', 'Vie amoureuse') AND parent_id IS NULL);

    -- Budgets : remapper vers Autres uniquement s'il n'y a pas déjà un budget dessus
    UPDATE budgets SET domain_id = autres_id
    WHERE domain_id IN (SELECT id FROM domains WHERE name IN ('Vie perso', 'Vie amoureuse', 'Loisirs') AND parent_id IS NULL)
      AND NOT EXISTS (SELECT 1 FROM budgets b2 WHERE b2.domain_id = autres_id);
END $$;

-- 3. Supprimer les budgets orphelins restants sur les anciens domaines.
DELETE FROM budgets WHERE domain_id IN (
    SELECT id FROM domains WHERE name IN ('Vie pro', 'Vie perso', 'Vie amoureuse', 'Loisirs') AND parent_id IS NULL
);

-- 4. Retirer les anciens domaines de vie désormais vides.
DELETE FROM domains WHERE parent_id IS NULL
  AND name IN ('Vie pro', 'Vie perso', 'Vie amoureuse', 'Loisirs');

-- 5. Mettre à jour la couleur de Logement / Famille au nouveau nuancier.
UPDATE domains SET color = '#4C6FBF' WHERE name = 'Logement' AND parent_id IS NULL;
UPDATE domains SET color = '#B5548A' WHERE name = 'Famille' AND parent_id IS NULL;
