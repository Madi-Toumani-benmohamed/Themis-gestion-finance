-- Independent user documents; no relation to bank transactions or obligations.
CREATE TABLE laboratory_documents (
 id BIGSERIAL PRIMARY KEY,
 owner_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 client_key UUID NOT NULL,
 title TEXT NOT NULL CHECK(length(btrim(title)) BETWEEN 1 AND 120),
 type TEXT NOT NULL CHECK(type IN ('scratchpad','workbook')),
 content JSONB NOT NULL CHECK(jsonb_typeof(content)='object' AND octet_length(content::text)<=262144),
 schema_version INTEGER NOT NULL CHECK(schema_version=1),
 revision BIGINT NOT NULL DEFAULT 1 CHECK(revision BETWEEN 1 AND 9007199254740991),
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 UNIQUE(owner_id,client_key)
);
CREATE INDEX idx_laboratory_owner_updated ON laboratory_documents(owner_id,updated_at DESC,id DESC);
