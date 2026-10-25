-- Refuser un rollback qui effacerait une qualification ou une preuve utilisateur.
DO $$
BEGIN
 IF EXISTS(SELECT 1 FROM reimbursement_history)
 OR EXISTS(SELECT 1 FROM reimbursement_allocations)
 OR EXISTS(SELECT 1 FROM reimbursement_cases)
 THEN
  RAISE EXCEPTION 'Rollback refusé : les remboursements ou leur historique existent.';
 END IF;
 DROP TRIGGER protect_active_reimbursement_transaction_delete ON transactions;
 DROP FUNCTION protect_active_reimbursement_transaction_delete();
 DROP TRIGGER verify_reimbursement_allocation_target ON reimbursement_allocations;
 DROP FUNCTION verify_reimbursement_allocation_target();
 DROP TRIGGER verify_reimbursement_case_credit ON reimbursement_cases;
 DROP FUNCTION verify_reimbursement_case_credit();
 DROP TABLE reimbursement_history;
 DROP TABLE reimbursement_allocations;
 DROP TABLE reimbursement_cases;
 DROP INDEX idx_transactions_identity_owner;
 DELETE FROM schema_migrations WHERE name='027_reimbursements.sql';
END $$;
