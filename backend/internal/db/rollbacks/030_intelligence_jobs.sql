-- Enqueue state can be discarded only when no work is pending or in flight.
DO $$
BEGIN
 IF EXISTS(SELECT 1 FROM intelligence_jobs WHERE status IN ('queued','running')) THEN
  RAISE EXCEPTION 'Rollback refusé : des recalculs intelligence sont en attente.';
 END IF;
 DROP TABLE intelligence_jobs;
 DELETE FROM schema_migrations WHERE name='030_intelligence_jobs.sql';
END $$;
