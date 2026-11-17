DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM agent_missions) OR EXISTS(SELECT 1 FROM agent_actions) OR EXISTS(SELECT 1 FROM agent_memories) OR EXISTS(SELECT 1 FROM chat_messages WHERE metadata<>'{}'::jsonb) OR EXISTS(SELECT 1 FROM chat_conversations WHERE agent_context<>'{}'::jsonb) THEN
 RAISE EXCEPTION 'Assistant data exists; rollback refused'; END IF;
END $$;
DROP TABLE agent_actions;
DROP TABLE agent_memories;
DROP TABLE agent_missions;
ALTER TABLE chat_messages DROP COLUMN metadata;
ALTER TABLE chat_conversations DROP COLUMN agent_context;
DELETE FROM schema_migrations WHERE name='034_agent_workspace.sql';
