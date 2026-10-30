DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM laboratory_documents) THEN
  RAISE EXCEPTION 'Rollback refusé : documents du Laboratoire existants.';
 END IF;
 DROP TABLE laboratory_documents;
 DELETE FROM schema_migrations WHERE name='032_laboratory.sql';
END $$;
