-- Qualification explicite et allocations traçables. Aucune écriture bancaire
-- existante n'est mise à jour ou reclassée par cette migration.
CREATE UNIQUE INDEX IF NOT EXISTS idx_transactions_identity_owner
 ON transactions(id,owner_id);

CREATE TABLE reimbursement_cases (
 id BIGSERIAL PRIMARY KEY,
 owner_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 credit_transaction_id INTEGER,
 credit_snapshot JSONB NOT NULL,
 kind TEXT NOT NULL CHECK(kind IN ('personal_expense','merchant_refund')),
 status TEXT NOT NULL DEFAULT 'unmatched'
  CHECK(status IN ('unmatched','partial','allocated','needs_review','voided')),
 note TEXT NOT NULL DEFAULT '' CHECK(length(note)<=2000),
 created_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 UNIQUE(id,owner_id),
 CONSTRAINT reimbursement_cases_credit_owner_fk
  FOREIGN KEY(credit_transaction_id,owner_id)
  REFERENCES transactions(id,owner_id)
  ON DELETE SET NULL (credit_transaction_id)
);
CREATE UNIQUE INDEX idx_reimbursement_cases_credit_owner
 ON reimbursement_cases(owner_id,credit_transaction_id)
 WHERE credit_transaction_id IS NOT NULL;
CREATE INDEX idx_reimbursement_cases_owner_status
 ON reimbursement_cases(owner_id,status,updated_at DESC,id DESC);

CREATE TABLE reimbursement_allocations (
 id BIGSERIAL PRIMARY KEY,
 owner_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 case_id BIGINT NOT NULL,
 expense_transaction_id INTEGER,
 expense_snapshot JSONB NOT NULL,
 amount_cents BIGINT NOT NULL CHECK(amount_cents>0 AND amount_cents<=999999999999),
 idempotency_key TEXT NOT NULL CHECK(length(idempotency_key) BETWEEN 16 AND 100),
 created_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 voided_at TIMESTAMPTZ,
 voided_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
 void_reason TEXT NOT NULL DEFAULT '',
 CHECK(length(void_reason)<=1000),
 UNIQUE(id,owner_id),
 CONSTRAINT reimbursement_allocations_case_owner_fk
  FOREIGN KEY(case_id,owner_id)
  REFERENCES reimbursement_cases(id,owner_id) ON DELETE CASCADE,
 CONSTRAINT reimbursement_allocations_expense_owner_fk
  FOREIGN KEY(expense_transaction_id,owner_id)
  REFERENCES transactions(id,owner_id)
  ON DELETE SET NULL (expense_transaction_id)
);
CREATE UNIQUE INDEX idx_reimbursement_allocation_submission
 ON reimbursement_allocations(owner_id,idempotency_key);
CREATE INDEX idx_reimbursement_allocations_case_active
 ON reimbursement_allocations(owner_id,case_id,id) WHERE voided_at IS NULL;
CREATE INDEX idx_reimbursement_allocations_expense_active
 ON reimbursement_allocations(owner_id,expense_transaction_id,id)
 WHERE voided_at IS NULL AND expense_transaction_id IS NOT NULL;

CREATE TABLE reimbursement_history (
 id BIGSERIAL PRIMARY KEY,
 owner_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 case_id BIGINT NOT NULL,
 allocation_id BIGINT,
 event_type TEXT NOT NULL CHECK(event_type IN ('case_created','allocation_created','allocation_voided','case_voided')),
 before_state JSONB,
 after_state JSONB NOT NULL,
 actor_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
 note TEXT NOT NULL DEFAULT '' CHECK(length(note)<=1000),
 changed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 FOREIGN KEY(case_id,owner_id)
  REFERENCES reimbursement_cases(id,owner_id) ON DELETE CASCADE,
 FOREIGN KEY(allocation_id,owner_id)
  REFERENCES reimbursement_allocations(id,owner_id)
  ON DELETE SET NULL (allocation_id)
);
CREATE INDEX idx_reimbursement_history_owner_case
 ON reimbursement_history(owner_id,case_id,id DESC);

CREATE FUNCTION verify_reimbursement_case_credit()
RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
 valid_credit BOOLEAN;
BEGIN
 IF NEW.credit_transaction_id IS NULL THEN RETURN NEW; END IF;
 SELECT t.amount>0 AND COALESCE(t.label_kind,'')<>'savings' AND NOT EXISTS(
  SELECT 1 FROM domains own_domain
  LEFT JOIN domains parent_domain ON parent_domain.id=own_domain.parent_id AND parent_domain.owner_id=own_domain.owner_id
  WHERE own_domain.id=t.domain_id AND own_domain.owner_id=t.owner_id
  AND (own_domain.is_transfer OR COALESCE(parent_domain.is_transfer,false))
 ) INTO valid_credit
 FROM transactions t WHERE t.id=NEW.credit_transaction_id AND t.owner_id=NEW.owner_id;
 IF NOT COALESCE(valid_credit,false) THEN
  RAISE EXCEPTION 'source de remboursement non créditrice ou non éligible' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER verify_reimbursement_case_credit
 BEFORE INSERT OR UPDATE OF credit_transaction_id,owner_id ON reimbursement_cases
 FOR EACH ROW EXECUTE FUNCTION verify_reimbursement_case_credit();

CREATE FUNCTION verify_reimbursement_allocation_target()
RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
 valid_target BOOLEAN;
 case_status TEXT;
BEGIN
 IF NEW.expense_transaction_id IS NULL THEN RETURN NEW; END IF;
 SELECT status INTO case_status FROM reimbursement_cases WHERE id=NEW.case_id AND owner_id=NEW.owner_id;
 IF case_status IS NULL OR case_status='voided' THEN
  RAISE EXCEPTION 'dossier de remboursement fermé ou inaccessible' USING ERRCODE='23514';
 END IF;
 SELECT t.amount<0 AND COALESCE(t.label_kind,'')<>'savings' AND NOT EXISTS(
  SELECT 1 FROM domains own_domain
  LEFT JOIN domains parent_domain ON parent_domain.id=own_domain.parent_id AND parent_domain.owner_id=own_domain.owner_id
  WHERE own_domain.id=t.domain_id AND own_domain.owner_id=t.owner_id
  AND (own_domain.is_transfer OR COALESCE(parent_domain.is_transfer,false))
 ) INTO valid_target
 FROM transactions t WHERE t.id=NEW.expense_transaction_id AND t.owner_id=NEW.owner_id;
 IF NOT COALESCE(valid_target,false) THEN
  RAISE EXCEPTION 'dépense cible non débitrice ou non éligible' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER verify_reimbursement_allocation_target
 BEFORE INSERT OR UPDATE OF case_id,owner_id,expense_transaction_id,amount_cents ON reimbursement_allocations
 FOR EACH ROW EXECUTE FUNCTION verify_reimbursement_allocation_target();

CREATE FUNCTION protect_active_reimbursement_transaction_delete()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF EXISTS(
  SELECT 1 FROM reimbursement_cases c
  WHERE c.owner_id=OLD.owner_id AND c.credit_transaction_id=OLD.id AND c.status<>'voided'
 ) OR EXISTS(
  SELECT 1 FROM reimbursement_allocations a
  WHERE a.owner_id=OLD.owner_id AND a.expense_transaction_id=OLD.id AND a.voided_at IS NULL
 ) THEN
  RAISE EXCEPTION 'transaction liée à un remboursement actif' USING ERRCODE='23503';
 END IF;
 RETURN OLD;
END $$;
CREATE TRIGGER protect_active_reimbursement_transaction_delete
 BEFORE DELETE ON transactions
 FOR EACH ROW EXECUTE FUNCTION protect_active_reimbursement_transaction_delete();
