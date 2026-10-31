DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM knowledge_activity) THEN
  RAISE EXCEPTION 'Rollback refusé : preuves d’activité mémoire existantes.';
 END IF;
 DROP TABLE knowledge_activity;
 DROP INDEX idx_statements_identity_owner;
 DROP INDEX idx_chat_conversations_identity_owner;
 DELETE FROM schema_migrations WHERE name='033_knowledge_activity.sql';
END $$;
