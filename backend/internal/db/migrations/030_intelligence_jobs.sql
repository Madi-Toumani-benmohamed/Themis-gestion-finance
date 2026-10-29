-- One coalescing durable recalculation job per owner.
CREATE TABLE intelligence_jobs (
 owner_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
 status TEXT NOT NULL DEFAULT 'queued' CHECK(status IN ('queued','running','done','failed')),
 trigger TEXT NOT NULL DEFAULT 'manual' CHECK(length(trigger) BETWEEN 1 AND 80),
 requested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 available_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 lease_until TIMESTAMPTZ,
 attempts INTEGER NOT NULL DEFAULT 0 CHECK(attempts>=0),
 rerun_requested BOOLEAN NOT NULL DEFAULT false,
 last_success_at TIMESTAMPTZ,
 last_error TEXT NOT NULL DEFAULT '',
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX intelligence_jobs_claim_idx ON intelligence_jobs(status,available_at,requested_at);
