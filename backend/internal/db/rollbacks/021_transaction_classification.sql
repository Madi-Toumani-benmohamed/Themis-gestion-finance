-- Run with the prior application release. Removes only provenance/proposals,
-- retaining assigned categories, tags, amounts and original bank labels.
BEGIN;
DROP TABLE IF EXISTS transaction_suggestions;
DROP INDEX IF EXISTS idx_transactions_owner_date_id;
ALTER TABLE transactions DROP COLUMN IF EXISTS classification_source;
DELETE FROM schema_migrations WHERE name='021_transaction_classification.sql';
COMMIT;
