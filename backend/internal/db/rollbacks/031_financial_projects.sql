DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM financial_projects) THEN
  RAISE EXCEPTION 'Rollback refusé : projets financiers existants.';
 END IF;
 DROP TRIGGER protect_project_transaction_source ON transactions;
 DROP FUNCTION protect_project_transaction_source();
 DROP TABLE project_history,project_transactions,project_rules,financial_projects;
 DROP INDEX idx_tags_identity_owner;
 DROP INDEX idx_domains_identity_owner;
 DELETE FROM schema_migrations WHERE name='031_financial_projects.sql';
END $$;
