-- Latest explicit user preference per discovery. Feedback history remains in
-- intelligence_feedback; this table is the editable source for personalization.
CREATE TABLE intelligence_preferences (
 owner_id BIGINT NOT NULL,
 discovery_id TEXT NOT NULL,
 action TEXT NOT NULL CHECK(action IN ('interesting','already_knew','irrelevant','normal','remind_me')),
 note TEXT NOT NULL DEFAULT '' CHECK(length(note)<=500),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 PRIMARY KEY(owner_id,discovery_id),
 FOREIGN KEY(owner_id,discovery_id)
  REFERENCES intelligence_discoveries(owner_id,id) ON DELETE CASCADE
);
CREATE INDEX intelligence_preferences_kind_action_idx
 ON intelligence_preferences(owner_id,action,updated_at DESC);
