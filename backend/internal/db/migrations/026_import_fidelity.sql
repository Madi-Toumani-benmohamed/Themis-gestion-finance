-- Additive provenance only. Historical transactions and their business hashes
-- remain untouched; an old statement needs an explicit, reviewed repair.
ALTER TABLE statements ADD COLUMN source_sha256 text;
ALTER TABLE statements ADD COLUMN source_parse_hash text;
ALTER TABLE statements ADD COLUMN import_report jsonb;
CREATE UNIQUE INDEX idx_statement_source_document ON statements(owner_id,account_id,source_sha256) WHERE source_sha256 IS NOT NULL;
ALTER TABLE transactions ADD COLUMN source_business_hash text;
ALTER TABLE transactions ADD COLUMN source_index integer;
ALTER TABLE transactions ADD COLUMN source_page integer;
ALTER TABLE transactions ADD COLUMN source_line integer;
ALTER TABLE transactions ADD COLUMN value_date date;
CREATE INDEX idx_transaction_source_business ON transactions(owner_id,source_business_hash) WHERE source_business_hash IS NOT NULL;
ALTER TABLE transactions ADD CONSTRAINT source_location_positive CHECK (
 (source_index IS NULL OR source_index>=0) AND (source_page IS NULL OR source_page>0) AND (source_line IS NULL OR source_line>0));
