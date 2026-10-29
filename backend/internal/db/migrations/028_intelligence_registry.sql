-- Durable discovery identity, immutable observations and explicit feedback.
CREATE TABLE intelligence_discoveries (
 owner_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 id TEXT NOT NULL CHECK(length(id) BETWEEN 16 AND 80),
 kind TEXT NOT NULL CHECK(length(kind) BETWEEN 1 AND 80),
 subject_key TEXT NOT NULL CHECK(length(subject_key) BETWEEN 1 AND 500),
 state TEXT NOT NULL DEFAULT 'active' CHECK(state IN ('active','hidden','resolved')),
 detected_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 PRIMARY KEY(owner_id,id),
 UNIQUE(owner_id,kind,subject_key)
);
CREATE INDEX intelligence_discoveries_feed_idx
 ON intelligence_discoveries(owner_id,state,updated_at DESC,id);

CREATE TABLE intelligence_observations (
 owner_id BIGINT NOT NULL,
 id TEXT NOT NULL CHECK(length(id) BETWEEN 16 AND 80),
 discovery_id TEXT NOT NULL,
 period TEXT NOT NULL DEFAULT '',
 title TEXT NOT NULL CHECK(length(title) BETWEEN 1 AND 500),
 summary TEXT NOT NULL CHECK(length(summary)<=4000),
 severity TEXT NOT NULL DEFAULT '',
 confidence SMALLINT NOT NULL CHECK(confidence BETWEEN 0 AND 100),
 relevance_score SMALLINT NOT NULL CHECK(relevance_score BETWEEN 0 AND 100),
 impact_score SMALLINT NOT NULL CHECK(impact_score BETWEEN 0 AND 100),
 metrics JSONB NOT NULL DEFAULT '[]'::jsonb CHECK(jsonb_typeof(metrics)='array'),
 evidence JSONB NOT NULL DEFAULT '{}'::jsonb CHECK(jsonb_typeof(evidence)='object'),
 coverage JSONB NOT NULL DEFAULT '{}'::jsonb CHECK(jsonb_typeof(coverage)='object'),
 limitations JSONB NOT NULL DEFAULT '[]'::jsonb CHECK(jsonb_typeof(limitations)='array'),
 rule_version TEXT NOT NULL DEFAULT '1',
 detected_at TIMESTAMPTZ NOT NULL,
 updated_at TIMESTAMPTZ NOT NULL,
 PRIMARY KEY(owner_id,id),
 UNIQUE(owner_id,discovery_id,id),
 FOREIGN KEY(owner_id,discovery_id)
  REFERENCES intelligence_discoveries(owner_id,id) ON DELETE CASCADE
);
CREATE INDEX intelligence_observations_latest_idx
 ON intelligence_observations(owner_id,discovery_id,updated_at DESC,id DESC);

CREATE TABLE intelligence_feedback (
 id BIGSERIAL PRIMARY KEY,
 owner_id BIGINT NOT NULL,
 discovery_id TEXT NOT NULL,
 action TEXT NOT NULL CHECK(action IN ('interesting','already_knew','irrelevant','normal','remind_me','hide','unhide')),
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 FOREIGN KEY(owner_id,discovery_id)
  REFERENCES intelligence_discoveries(owner_id,id) ON DELETE CASCADE
);
CREATE INDEX intelligence_feedback_owner_discovery_idx
 ON intelligence_feedback(owner_id,discovery_id,created_at DESC,id DESC);
