package api

// notTransferClause est la condition SQL qui exclut les transactions de
// transfert interne (virements entre comptes personnels) des calculs de flux.
// Elle doit être ajoutée à toute requête qui somme des dépenses ou des revenus,
// sinon un virement courant→Livret A serait compté comme une dépense d'un côté
// et un revenu de l'autre, faussant les statistiques.
//
// Usage : concaténer dans le WHERE d'une requête sur `transactions` (alias `t`
// ou table nue selon le cas — deux variantes fournies). La sous-requête est
// volontairement simple ; PostgreSQL la met en cache par requête.
const (
	// Pour une table transactions sans alias (colonne domain_id directe).
	notTransfer = ` AND (domain_id IS NULL OR domain_id NOT IN (SELECT id FROM domains WHERE is_transfer)) `
	// Pour une table transactions aliasée `t`.
	notTransferT = ` AND (t.domain_id IS NULL OR t.domain_id NOT IN (SELECT id FROM domains WHERE is_transfer)) `
)
