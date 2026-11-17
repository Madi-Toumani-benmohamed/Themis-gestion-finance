-- Additive assistant workspace; bank transactions and balances are untouched.
ALTER TABLE chat_conversations ADD COLUMN agent_context JSONB NOT NULL DEFAULT '{}';
ALTER TABLE chat_messages ADD COLUMN metadata JSONB NOT NULL DEFAULT '{}';
CREATE TABLE agent_missions (
 id BIGSERIAL PRIMARY KEY, owner_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 conversation_id BIGINT, title TEXT NOT NULL CHECK(length(title) BETWEEN 1 AND 160), objective TEXT NOT NULL CHECK(length(objective) BETWEEN 1 AND 2000),
 status TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active','paused','completed')),
 last_review_at TIMESTAMPTZ, created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 FOREIGN KEY(conversation_id,owner_id) REFERENCES chat_conversations(id,owner_id) ON DELETE CASCADE,
 UNIQUE(id,owner_id)
);
CREATE INDEX idx_agent_missions_owner ON agent_missions(owner_id,updated_at DESC,id DESC);
CREATE TABLE agent_memories (
 id BIGSERIAL PRIMARY KEY, owner_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 content TEXT NOT NULL CHECK(length(content) BETWEEN 1 AND 1000), created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_agent_memories_owner ON agent_memories(owner_id,id DESC);
CREATE TABLE agent_actions (
 id BIGSERIAL PRIMARY KEY, owner_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 conversation_id BIGINT NOT NULL, mission_id BIGINT,
 kind TEXT NOT NULL CHECK(kind IN ('categorize','tag','rule','simulation','memory')),
 status TEXT NOT NULL DEFAULT 'prepared' CHECK(status IN ('prepared','applied','dismissed','stale')),
 payload JSONB NOT NULL CHECK(octet_length(payload::text)<=32768), preview JSONB NOT NULL CHECK(octet_length(preview::text)<=131072),
 fingerprint TEXT NOT NULL DEFAULT '', result JSONB, created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 FOREIGN KEY(conversation_id,owner_id) REFERENCES chat_conversations(id,owner_id) ON DELETE CASCADE,
 FOREIGN KEY(mission_id,owner_id) REFERENCES agent_missions(id,owner_id) ON DELETE CASCADE
);
CREATE INDEX idx_agent_actions_owner ON agent_actions(owner_id,conversation_id,id DESC);
