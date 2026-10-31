-- Operational evidence only. No synthetic facts and no backfilled history.
CREATE UNIQUE INDEX idx_statements_identity_owner ON statements(id,owner_id);
CREATE UNIQUE INDEX idx_chat_conversations_identity_owner ON chat_conversations(id,owner_id);
CREATE TABLE knowledge_activity (
 id BIGSERIAL PRIMARY KEY,
 owner_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 kind TEXT NOT NULL CHECK(kind IN ('learn','search','chat')),
 source TEXT NOT NULL CHECK(source IN ('statement','manual','diagnostic','assistant')),
 statement_id BIGINT,
 conversation_id BIGINT,
 status TEXT NOT NULL CHECK(status IN ('running','success','empty','error','uncertain','partial','disabled','not_configured','cancelled','skipped')),
 fact_count INTEGER CHECK(fact_count>=0),
 processed_count INTEGER CHECK(processed_count>=0),
 entities_processed INTEGER CHECK(entities_processed>=0),
 edges_processed INTEGER CHECK(edges_processed>=0),
 result_count INTEGER CHECK(result_count BETWEEN 0 AND 5),
 context_included BOOLEAN NOT NULL DEFAULT false,
 facts JSONB NOT NULL DEFAULT '[]' CHECK(jsonb_typeof(facts)='array' AND jsonb_array_length(facts)<=5 AND octet_length(facts::text)<=65536),
 error_code TEXT NOT NULL DEFAULT '' CHECK(length(error_code)<=80),
 duration_ms BIGINT CHECK(duration_ms>=0),
 started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 finished_at TIMESTAMPTZ,
 FOREIGN KEY(statement_id,owner_id) REFERENCES statements(id,owner_id) ON DELETE CASCADE,
 FOREIGN KEY(conversation_id,owner_id) REFERENCES chat_conversations(id,owner_id) ON DELETE CASCADE
);
CREATE INDEX idx_knowledge_activity_owner_started ON knowledge_activity(owner_id,id DESC);

CREATE INDEX idx_knowledge_activity_last_kind ON knowledge_activity(owner_id,kind,id DESC);
CREATE INDEX idx_knowledge_activity_statement ON knowledge_activity(statement_id) WHERE statement_id IS NOT NULL;
CREATE INDEX idx_knowledge_activity_conversation ON knowledge_activity(conversation_id) WHERE conversation_id IS NOT NULL;
