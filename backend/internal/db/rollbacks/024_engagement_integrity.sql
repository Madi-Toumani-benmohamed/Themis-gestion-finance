-- Retour au schéma 023 UNIQUEMENT avant toute utilisation des nouveaux champs.
-- Un seul bloc atomique ; refus explicite plutôt que perte de paiements/preuves.
DO $$
BEGIN
 IF EXISTS(SELECT 1 FROM credit_history)
 OR EXISTS(SELECT 1 FROM credits WHERE opening_paid_cents IS NOT NULL OR closed_at IS NOT NULL OR archived_at IS NOT NULL)
 OR EXISTS(SELECT 1 FROM credit_payments WHERE idempotency_key IS NOT NULL OR voided_at IS NOT NULL OR void_note<>'')
 OR EXISTS(SELECT 1 FROM taxes WHERE paid_amount_cents IS NOT NULL OR overdue_confirmed OR archived_at IS NOT NULL)
 OR EXISTS(SELECT 1 FROM tax_history WHERE event_type<>'legacy_status' OR before_state IS NOT NULL OR after_state IS NOT NULL OR idempotency_key IS NOT NULL OR payment_date IS NOT NULL OR payment_amount_cents IS NOT NULL OR note<>'' OR tax_id IS NULL OR changed_by IS NULL)
 THEN RAISE EXCEPTION 'Rollback refusé : des déclarations ou preuves nouvelles existent. Conserver le schéma et préparer une restauration vérifiée.';
 END IF;
 DROP TABLE credit_history;
 ALTER TABLE credit_payments DROP CONSTRAINT credit_payments_credit_owner_fk;
 ALTER TABLE credit_payments ADD CONSTRAINT credit_payments_credit_id_fkey FOREIGN KEY(credit_id) REFERENCES credits(id) ON DELETE CASCADE;
 DROP INDEX idx_credit_payments_submission;
 DROP INDEX idx_credit_payments_owner_credit;
 ALTER TABLE credit_payments DROP COLUMN idempotency_key, DROP COLUMN voided_at, DROP COLUMN void_note;
 ALTER TABLE credits DROP CONSTRAINT credits_opening_paid_valid;
 DROP INDEX idx_credits_identity_owner;
 ALTER TABLE credits DROP COLUMN opening_paid_cents, DROP COLUMN closed_at, DROP COLUMN archived_at;
 ALTER TABLE tax_history DROP CONSTRAINT tax_history_tax_owner_fk, DROP CONSTRAINT tax_history_owner_required;
 ALTER TABLE tax_history ALTER COLUMN tax_id SET NOT NULL;
 ALTER TABLE tax_history DROP CONSTRAINT tax_history_changed_by_fkey;
 ALTER TABLE tax_history ALTER COLUMN changed_by SET NOT NULL;
 ALTER TABLE tax_history ADD CONSTRAINT tax_history_changed_by_fkey FOREIGN KEY(changed_by) REFERENCES users(id) ON DELETE CASCADE;
 ALTER TABLE tax_history ADD CONSTRAINT tax_history_tax_id_fkey FOREIGN KEY(tax_id) REFERENCES taxes(id) ON DELETE CASCADE;
 DROP INDEX idx_tax_history_owner_resource;
 DROP INDEX idx_tax_history_submission;
 ALTER TABLE tax_history DROP COLUMN owner_id, DROP COLUMN resource_id, DROP COLUMN event_type, DROP COLUMN before_state, DROP COLUMN after_state, DROP COLUMN idempotency_key, DROP COLUMN payment_date, DROP COLUMN payment_amount_cents, DROP COLUMN note;
 ALTER TABLE taxes DROP CONSTRAINT taxes_paid_amount_valid;
 DROP INDEX idx_taxes_identity_owner;
 ALTER TABLE taxes DROP COLUMN paid_amount_cents, DROP COLUMN overdue_confirmed, DROP COLUMN archived_at;
 DELETE FROM schema_migrations WHERE name='024_engagement_integrity.sql';
END $$;
