-- Additive: legacy tag folders and bank transactions are untouched.
CREATE UNIQUE INDEX IF NOT EXISTS idx_tags_identity_owner ON tags(id,owner_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_domains_identity_owner ON domains(id,owner_id);
CREATE TABLE financial_projects (
 id BIGSERIAL PRIMARY KEY,
 owner_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 name TEXT NOT NULL CHECK(length(btrim(name)) BETWEEN 1 AND 120),
 description TEXT NOT NULL DEFAULT '' CHECK(length(description)<=2000),
 budget_cents BIGINT CHECK(budget_cents BETWEEN 0 AND 999999999999),
 start_date DATE,
 end_date DATE,
 status TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active','completed','archived')),
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 CHECK(start_date IS NULL OR end_date IS NULL OR end_date>=start_date),
 UNIQUE(id,owner_id)
);
CREATE INDEX idx_financial_projects_owner ON financial_projects(owner_id,status,id);
CREATE TABLE project_rules (
 id BIGSERIAL PRIMARY KEY,
 owner_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 project_id BIGINT NOT NULL,
 kind TEXT NOT NULL CHECK(kind IN ('tag','category','merchant','label')),
 tag_id INTEGER,
 domain_id INTEGER,
 match_text TEXT NOT NULL DEFAULT '' CHECK(length(match_text)<=200),
 from_date DATE NOT NULL,
 to_date DATE NOT NULL,
 active BOOLEAN NOT NULL DEFAULT true,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 CHECK(to_date>=from_date AND to_date-from_date<=3660),
 CHECK((kind='tag' AND tag_id IS NOT NULL AND domain_id IS NULL AND match_text='') OR
       (kind='category' AND domain_id IS NOT NULL AND tag_id IS NULL AND match_text='') OR
       (kind IN ('merchant','label') AND tag_id IS NULL AND domain_id IS NULL AND length(btrim(match_text))>=3)),
 FOREIGN KEY(project_id,owner_id) REFERENCES financial_projects(id,owner_id) ON DELETE CASCADE,
 FOREIGN KEY(tag_id,owner_id) REFERENCES tags(id,owner_id) ON DELETE CASCADE,
 FOREIGN KEY(domain_id,owner_id) REFERENCES domains(id,owner_id) ON DELETE CASCADE,
 UNIQUE(id,owner_id)
);
CREATE UNIQUE INDEX idx_project_rules_criteria ON project_rules(project_id,kind,COALESCE(tag_id,0),COALESCE(domain_id,0),match_text,from_date,to_date) WHERE active;
CREATE INDEX idx_project_rules_owner ON project_rules(owner_id,project_id) WHERE active;
CREATE TABLE project_transactions (
 owner_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 project_id BIGINT NOT NULL,
 transaction_id INTEGER NOT NULL,
 state TEXT NOT NULL CHECK(state IN ('suggested','confirmed','rejected')),
 reason JSONB NOT NULL,
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 PRIMARY KEY(project_id,transaction_id),
 FOREIGN KEY(project_id,owner_id) REFERENCES financial_projects(id,owner_id) ON DELETE CASCADE,
 FOREIGN KEY(transaction_id,owner_id) REFERENCES transactions(id,owner_id) ON DELETE CASCADE
);
CREATE INDEX idx_project_transactions_owner_state ON project_transactions(owner_id,project_id,state,transaction_id);
CREATE TABLE project_history (
 id BIGSERIAL PRIMARY KEY,
 owner_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 project_id BIGINT NOT NULL,
 transaction_id INTEGER,
 event TEXT NOT NULL,
 before_state JSONB,
 after_state JSONB NOT NULL,
 changed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 FOREIGN KEY(project_id,owner_id) REFERENCES financial_projects(id,owner_id) ON DELETE CASCADE
);
CREATE INDEX idx_project_history_owner ON project_history(owner_id,project_id,id DESC);

-- Confirmation protects its source until explicitly withdrawn. Suggestions and
-- rejected links can disappear with a deleted statement; decisions stay in history.
CREATE FUNCTION protect_project_transaction_source() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 -- A full owner deletion already removes all their evidence by the existing
 -- explicit users cascade. Do not block it on an individual source guard.
 IF EXISTS(SELECT 1 FROM users WHERE id=OLD.owner_id) AND EXISTS(SELECT 1 FROM project_transactions pt WHERE pt.transaction_id=OLD.id AND pt.owner_id=OLD.owner_id AND pt.state='confirmed') THEN
  RAISE EXCEPTION 'Retirer les confirmations de projet avant suppression de la source.' USING ERRCODE='23514';
 END IF;
 RETURN OLD;
END $$;
CREATE TRIGGER protect_project_transaction_source BEFORE DELETE ON transactions FOR EACH ROW EXECUTE FUNCTION protect_project_transaction_source();
-- Reverse lookup for source protection and FK cascade checks.
CREATE INDEX idx_project_transaction_source ON project_transactions(transaction_id,owner_id);
CREATE INDEX idx_project_rule_tag ON project_rules(tag_id,owner_id) WHERE tag_id IS NOT NULL;
CREATE INDEX idx_project_rule_domain ON project_rules(domain_id,owner_id) WHERE domain_id IS NOT NULL;
