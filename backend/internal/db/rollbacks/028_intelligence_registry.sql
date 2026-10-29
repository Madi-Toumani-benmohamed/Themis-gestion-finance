-- Refuser de supprimer des découvertes, observations ou retours utilisateur.
DO $$
BEGIN
 IF EXISTS(SELECT 1 FROM intelligence_feedback)
 OR EXISTS(SELECT 1 FROM intelligence_observations)
 OR EXISTS(SELECT 1 FROM intelligence_discoveries)
 THEN
  RAISE EXCEPTION 'Rollback refusé : le registre intelligence contient des données.';
 END IF;
 DROP TABLE intelligence_feedback;
 DROP TABLE intelligence_observations;
 DROP TABLE intelligence_discoveries;
 DELETE FROM schema_migrations WHERE name='028_intelligence_registry.sql';
END $$;
