-- Les règles automatiques peuvent désormais poser des tags en plus de la
-- catégorie. Ex. « Bouygues Telecom » → catégorie Télécommunications + tags
-- {factures, abonnement, bouygues, mobile}. On stocke la liste des noms de tags
-- en texte (séparés par des virgules) : cohérent avec le reste du code qui
-- manipule les tags par nom, et pas de table de jointure supplémentaire à gérer.
ALTER TABLE rules ADD COLUMN IF NOT EXISTS tags TEXT NOT NULL DEFAULT '';
