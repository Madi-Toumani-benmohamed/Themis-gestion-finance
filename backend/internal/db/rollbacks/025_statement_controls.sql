-- Refus de perdre une revue ou une configuration de suivi personnalisée.
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM statement_validation_history)
 OR EXISTS(SELECT 1 FROM accounts WHERE tracking_mode<>CASE WHEN kind='courant' THEN 'statements' ELSE 'manual' END)
 THEN RAISE EXCEPTION 'Rollback refusé : des revues ou modes de suivi personnalisés existent.';
 END IF;
 DROP TABLE statement_validation_history;
 DROP INDEX idx_transactions_statement_owner;
 ALTER TABLE accounts DROP COLUMN tracking_mode;
 DELETE FROM schema_migrations WHERE name='025_statement_controls.sql';
END $$;
