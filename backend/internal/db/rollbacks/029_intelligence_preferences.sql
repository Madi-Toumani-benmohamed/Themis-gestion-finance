-- Refuser de supprimer les préférences explicites enregistrées.
DO $$
BEGIN
 IF EXISTS(SELECT 1 FROM intelligence_preferences) THEN
  RAISE EXCEPTION 'Rollback refusé : des préférences intelligence existent.';
 END IF;
 DROP TABLE intelligence_preferences;
 DELETE FROM schema_migrations WHERE name='029_intelligence_preferences.sql';
END $$;
