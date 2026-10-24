package api

// Financial expense eligibility is deterministic: category transfers (including
// a child of a transfer category) and normalized savings movements are not
// consumption. Other credits remain credits, never inferred reimbursements.
// These predicates are shared by budgets, monthly summaries and legacy tools.
const (
	transferDomainIDsSQL = `SELECT own_domain.id FROM domains own_domain
 LEFT JOIN domains parent_domain ON parent_domain.id=own_domain.parent_id AND parent_domain.owner_id=own_domain.owner_id
 WHERE own_domain.is_transfer OR COALESCE(parent_domain.is_transfer,false)`
	financialExpenseSQLT = `(COALESCE(t.label_kind,'')<>'savings' AND (t.domain_id IS NULL OR t.domain_id NOT IN (` + transferDomainIDsSQL + `)))`
	notTransfer          = ` AND (COALESCE(label_kind,'')<>'savings' AND (domain_id IS NULL OR domain_id NOT IN (` + transferDomainIDsSQL + `))) `
	notTransferT         = ` AND ` + financialExpenseSQLT + ` `
)
