-- Consolidation additive des 022/023 : ne réécrit aucun montant d'engagement.
-- NULL = remboursements historiques non documentés, jamais zéro implicite.
ALTER TABLE credits ADD COLUMN IF NOT EXISTS opening_paid_cents BIGINT;
ALTER TABLE credits ADD COLUMN IF NOT EXISTS closed_at TIMESTAMPTZ;
ALTER TABLE credits ADD COLUMN IF NOT EXISTS archived_at TIMESTAMPTZ;
ALTER TABLE credits ADD CONSTRAINT credits_opening_paid_valid CHECK (
 opening_paid_cents IS NULL OR (opening_paid_cents>=0 AND opening_paid_cents<=monthly_amount*100*months_total)
) NOT VALID;
CREATE UNIQUE INDEX IF NOT EXISTS idx_credits_identity_owner ON credits(id,owner_id);

ALTER TABLE credit_payments ADD COLUMN IF NOT EXISTS idempotency_key TEXT;
ALTER TABLE credit_payments ADD COLUMN IF NOT EXISTS voided_at TIMESTAMPTZ;
ALTER TABLE credit_payments ADD COLUMN IF NOT EXISTS void_note TEXT NOT NULL DEFAULT '';
ALTER TABLE credit_payments DROP CONSTRAINT IF EXISTS credit_payments_credit_id_fkey;
-- NOT VALID protège toutes les nouvelles écritures sans effacer d'anciennes
-- associations incohérentes. Leur lecture sera signalée et exclue des certitudes.
ALTER TABLE credit_payments ADD CONSTRAINT credit_payments_credit_owner_fk
 FOREIGN KEY(credit_id,owner_id) REFERENCES credits(id,owner_id) ON DELETE NO ACTION DEFERRABLE INITIALLY DEFERRED NOT VALID;
CREATE UNIQUE INDEX IF NOT EXISTS idx_credit_payments_submission
 ON credit_payments(owner_id,credit_id,idempotency_key) WHERE idempotency_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_credit_payments_owner_credit ON credit_payments(owner_id,credit_id,payment_date,id);

CREATE TABLE IF NOT EXISTS credit_history (
 id BIGSERIAL PRIMARY KEY,
 credit_id BIGINT,
 resource_id BIGINT NOT NULL,
 owner_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 event_type TEXT NOT NULL,
 before_state JSONB,
 after_state JSONB NOT NULL,
 changed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 FOREIGN KEY(credit_id,owner_id) REFERENCES credits(id,owner_id) ON DELETE SET NULL (credit_id)
);
CREATE INDEX IF NOT EXISTS idx_credit_history_owner_resource ON credit_history(owner_id,resource_id,id DESC);

ALTER TABLE taxes ADD COLUMN IF NOT EXISTS paid_amount_cents BIGINT;
ALTER TABLE taxes ADD COLUMN IF NOT EXISTS overdue_confirmed BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE taxes ADD COLUMN IF NOT EXISTS archived_at TIMESTAMPTZ;
ALTER TABLE taxes ADD CONSTRAINT taxes_paid_amount_valid CHECK (
 paid_amount_cents IS NULL OR (paid_amount_cents>=0 AND paid_amount_cents<=amount*100
 AND paid=(paid_amount_cents=amount*100))
) NOT VALID;
CREATE UNIQUE INDEX IF NOT EXISTS idx_taxes_identity_owner ON taxes(id,owner_id);

ALTER TABLE tax_history ADD COLUMN IF NOT EXISTS owner_id BIGINT REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE tax_history ADD COLUMN IF NOT EXISTS resource_id BIGINT;
-- Ces deux métadonnées dérivent d'identifiants existants, pas de montants passés.
UPDATE tax_history h SET owner_id=t.owner_id FROM taxes t WHERE t.id=h.tax_id AND h.owner_id IS NULL;
UPDATE tax_history SET resource_id=tax_id WHERE resource_id IS NULL;
ALTER TABLE tax_history ADD COLUMN IF NOT EXISTS event_type TEXT NOT NULL DEFAULT 'legacy_status';
ALTER TABLE tax_history ADD COLUMN IF NOT EXISTS before_state JSONB;
ALTER TABLE tax_history ADD COLUMN IF NOT EXISTS after_state JSONB;
ALTER TABLE tax_history ADD COLUMN IF NOT EXISTS idempotency_key TEXT;
ALTER TABLE tax_history ADD COLUMN IF NOT EXISTS payment_date DATE;
ALTER TABLE tax_history ADD COLUMN IF NOT EXISTS payment_amount_cents BIGINT;
ALTER TABLE tax_history ADD COLUMN IF NOT EXISTS note TEXT NOT NULL DEFAULT '';
ALTER TABLE tax_history ALTER COLUMN tax_id DROP NOT NULL;
ALTER TABLE tax_history ALTER COLUMN changed_by DROP NOT NULL;
ALTER TABLE tax_history DROP CONSTRAINT IF EXISTS tax_history_changed_by_fkey;
ALTER TABLE tax_history ADD CONSTRAINT tax_history_changed_by_fkey
 FOREIGN KEY(changed_by) REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE tax_history DROP CONSTRAINT IF EXISTS tax_history_tax_id_fkey;
ALTER TABLE tax_history ADD CONSTRAINT tax_history_tax_owner_fk
 FOREIGN KEY(tax_id,owner_id) REFERENCES taxes(id,owner_id) ON DELETE SET NULL (tax_id) NOT VALID;
ALTER TABLE tax_history ADD CONSTRAINT tax_history_owner_required CHECK(owner_id IS NOT NULL) NOT VALID;
CREATE INDEX IF NOT EXISTS idx_tax_history_owner_resource ON tax_history(owner_id,resource_id,id DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_tax_history_submission
 ON tax_history(owner_id,idempotency_key) WHERE idempotency_key IS NOT NULL;
