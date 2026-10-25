-- Rollback is safe only before any source-addressed import. Otherwise restore
-- the application version while preserving these additive columns and data.
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM statements WHERE source_sha256 IS NOT NULL)
 THEN RAISE EXCEPTION 'Source-addressed imports exist; preserve provenance and restore a compatible application'; END IF;
END $$;
DROP INDEX idx_transaction_source_business;
ALTER TABLE transactions DROP CONSTRAINT source_location_positive;
ALTER TABLE transactions DROP COLUMN source_business_hash, DROP COLUMN source_index, DROP COLUMN source_page, DROP COLUMN source_line, DROP COLUMN value_date;
DROP INDEX idx_statement_source_document;
ALTER TABLE statements DROP COLUMN source_sha256, DROP COLUMN source_parse_hash, DROP COLUMN import_report;
DELETE FROM schema_migrations WHERE name='026_import_fidelity.sql';
