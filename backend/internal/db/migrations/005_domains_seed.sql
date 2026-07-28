-- Les domaines gagnent une couleur (pastille dans l'UI)
ALTER TABLE domains ADD COLUMN IF NOT EXISTS color TEXT NOT NULL DEFAULT '#66736C';

-- Amorçage d'un petit ensemble de domaines de vie, simple et parlant.
-- L'utilisateur peut en ajouter/retirer, mais il n'arrive plus devant du vide.
INSERT INTO domains (name, parent_id, color) VALUES
    ('Logement',       NULL, '#175C46'),
    ('Vie pro',        NULL, '#3D6B99'),
    ('Vie perso',      NULL, '#8C6D2F'),
    ('Vie amoureuse',  NULL, '#A63A28'),
    ('Famille',        NULL, '#6E4E8C'),
    ('Loisirs',        NULL, '#4E8578')
ON CONFLICT (name, parent_id) DO NOTHING;
