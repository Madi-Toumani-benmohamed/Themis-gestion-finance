-- Le type métier existant initialise le suivi, sans modifier les soldes.
ALTER TABLE accounts ADD COLUMN tracking_mode text;
UPDATE accounts SET tracking_mode=CASE WHEN kind='courant' THEN 'statements' ELSE 'manual' END;
ALTER TABLE accounts ALTER COLUMN tracking_mode SET NOT NULL;
ALTER TABLE accounts ALTER COLUMN tracking_mode SET DEFAULT 'statements';
ALTER TABLE accounts ADD CONSTRAINT accounts_tracking_mode CHECK(tracking_mode IN ('statements','manual'));

CREATE TABLE statement_validation_history (
 id bigserial PRIMARY KEY,
 statement_id integer REFERENCES statements(id) ON DELETE SET NULL,
 resource_id bigint NOT NULL,
 owner_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 actor_id bigint REFERENCES users(id) ON DELETE SET NULL,
 action text NOT NULL CHECK(action IN ('validate','reopen','delete')),
 previous_validated boolean NOT NULL,
 note text NOT NULL CHECK(length(note) BETWEEN 10 AND 1000),
 snapshot jsonb NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_statement_history_owner ON statement_validation_history(owner_id,resource_id,id DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_statement_owner ON transactions(statement_id,owner_id);
