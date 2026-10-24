-- Explicit rollback: removes review decisions only, never bank transactions.
BEGIN;
DROP TABLE IF EXISTS recurring_reviews;
DELETE FROM schema_migrations WHERE name = '020_recurring_reviews.sql';
COMMIT;
