// Package financialsql centralizes financial scope predicates for deterministic
// SQL consumers. Credits remain credits; a refund is never inferred from a sign.
package financialsql

const (
	TransferDomainIDs = `SELECT own_domain.id FROM domains own_domain
 LEFT JOIN domains parent_domain ON parent_domain.id=own_domain.parent_id AND parent_domain.owner_id=own_domain.owner_id
 WHERE own_domain.is_transfer OR COALESCE(parent_domain.is_transfer,false)`
	Expense = `(COALESCE(transactions.label_kind,'')<>'savings' AND NOT EXISTS(
 SELECT 1 FROM domains own_domain LEFT JOIN domains parent_domain
 ON parent_domain.id=own_domain.parent_id AND parent_domain.owner_id=own_domain.owner_id
 WHERE own_domain.id=transactions.domain_id AND own_domain.owner_id=transactions.owner_id
 AND (own_domain.is_transfer OR COALESCE(parent_domain.is_transfer,false))))`
	ExpenseT = `(COALESCE(t.label_kind,'')<>'savings' AND NOT EXISTS(
 SELECT 1 FROM domains own_domain LEFT JOIN domains parent_domain
 ON parent_domain.id=own_domain.parent_id AND parent_domain.owner_id=own_domain.owner_id
 WHERE own_domain.id=t.domain_id AND own_domain.owner_id=t.owner_id
 AND (own_domain.is_transfer OR COALESCE(parent_domain.is_transfer,false))))`
)
