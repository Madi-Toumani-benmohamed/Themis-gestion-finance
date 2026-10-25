package api

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/lib/pq"

	"themis/internal/categorize"
	"themis/internal/models"
	"themis/internal/money"
	"themis/internal/normalize"
	"themis/internal/parser"
)

func parseFingerprint(res *models.ParseResult) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s|%s|", res.PeriodStart, res.PeriodEnd)
	for _, b := range []*float64{res.OpeningBalance, res.ClosingBalance} {
		if b == nil {
			fmt.Fprint(h, "null|")
		} else {
			c, _ := money.FromLegacy(*b)
			fmt.Fprintf(h, "%s|", c)
		}
	}
	for _, t := range res.Transactions {
		fmt.Fprintf(h, "%s|%s|", parser.TxHash(t), t.ValueDate)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// persistDocument is all-or-nothing. Only a byte-identical source with an
// unchanged parsed fingerprint is an established duplicate. Business similarity
// across different sources is advisory and never suppresses an occurrence.
func (s *Server) persistDocument(filename string, res *models.ParseResult, accountID, owner int64) (inserted, skipped int, statementID int64, err error) {
	if len(res.DocumentHash) != 64 || res.Report == nil {
		return 0, 0, 0, fmt.Errorf("provenance d'import absente")
	}
	var opening, closing any
	for i, b := range []*float64{res.OpeningBalance, res.ClosingBalance} {
		if b != nil {
			c, e := money.FromLegacy(*b)
			if e != nil {
				return 0, 0, 0, fmt.Errorf("solde invalide")
			}
			if i == 0 {
				opening = c.String()
			} else {
				closing = c.String()
			}
		}
	}
	parser.RefreshImportReport(res)
	if res.Report.Completeness == "failed" {
		return 0, 0, 0, fmt.Errorf("contrôles d'import en échec : aucune opération enregistrée ; consulter le rapport")
	}
	for _, t := range res.Transactions {
		if _, e := money.FromLegacy(t.Amount); e != nil {
			return 0, 0, 0, fmt.Errorf("montant invalide")
		}
	}
	if res.Validated && !parser.Validate(res) {
		return 0, 0, 0, fmt.Errorf("validation comptable incohérente")
	}
	eng, e := categorize.Load(s.DB, owner)
	if e != nil {
		return 0, 0, 0, e
	}
	tx, e := s.DB.Begin()
	if e != nil {
		return 0, 0, 0, e
	}
	defer tx.Rollback()
	if _, e = tx.Exec(`SELECT pg_advisory_xact_lock($1)`, owner); e != nil {
		return 0, 0, 0, e
	}
	var own bool
	if e = tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM accounts WHERE id=$1 AND owner_id=$2)`, accountID, owner).Scan(&own); e != nil || !own {
		return 0, 0, 0, fmt.Errorf("compte cible indisponible")
	}
	var fingerprint string
	var count int
	e = tx.QueryRow(`SELECT st.id,st.source_parse_hash,(SELECT count(*) FROM transactions t WHERE t.statement_id=st.id AND t.owner_id=st.owner_id) FROM statements st WHERE st.owner_id=$1 AND st.account_id=$2 AND st.source_sha256=$3`, owner, accountID, res.DocumentHash).Scan(&statementID, &fingerprint, &count)
	if e == nil {
		if fingerprint != parseFingerprint(res) || count != len(res.Transactions) {
			return 0, 0, 0, fmt.Errorf("ce document existe avec un résultat différent ; réimport contrôlé nécessaire pour préserver les corrections manuelles")
		}
		if e = tx.Commit(); e != nil {
			return 0, 0, 0, e
		}
		res.Report.Inserted = 0
		res.Report.Duplicates = count
		for i := range res.Report.Candidates {
			res.Report.Candidates[i].Status = "duplicate"
			res.Report.Candidates[i].Reason = "identical_source_document"
		}
		return 0, count, statementID, nil
	}
	if e != sql.ErrNoRows {
		return 0, 0, 0, e
	}
	if res.Parser != "csv" && res.PeriodStart != "" && res.PeriodEnd != "" {
		var overlap bool
		if e = tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM statements WHERE owner_id=$1 AND account_id=$2 AND period_start<$4::date AND period_end>$3::date)`, owner, accountID, res.PeriodStart, res.PeriodEnd).Scan(&overlap); e != nil {
			return 0, 0, 0, e
		}
		if overlap {
			return 0, 0, 0, fmt.Errorf("période recouvrant un relevé existant : comparaison et réimport contrôlé requis ; aucune écriture modifiée")
		}
	}
	hashes := make([]string, 0, len(res.Transactions))
	for _, t := range res.Transactions {
		hashes = append(hashes, parser.TxHash(t))
	}
	var similar int
	if e = tx.QueryRow(`SELECT count(*) FROM transactions t JOIN statements st ON st.id=t.statement_id AND st.owner_id=t.owner_id WHERE t.owner_id=$1 AND st.account_id=$2 AND (t.hash=ANY($3) OR t.source_business_hash=ANY($3))`, owner, accountID, pq.Array(hashes)).Scan(&similar); e != nil {
		return 0, 0, 0, e
	}
	// Keep a separate report snapshot until commit; a failed insert must not claim
	// that the candidates were imported.
	raw, _ := json.Marshal(res.Report)
	var report models.ImportReport
	if e = json.Unmarshal(raw, &report); e != nil {
		return 0, 0, 0, e
	}
	report.Inserted = len(res.Transactions)
	if similar > 0 {
		report.Warnings = appendUniqueReportWarning(report.Warnings, "similar_operations_in_other_import")
	}
	for i := range report.Candidates {
		if report.Candidates[i].Status == "parsed" {
			report.Candidates[i].Status = "imported"
		}
	}
	raw, e = json.Marshal(report)
	if e != nil {
		return 0, 0, 0, e
	}
	e = tx.QueryRow(`INSERT INTO statements(filename,period_start,period_end,opening_balance,closing_balance,parser,validated,account_id,owner_id,source_sha256,source_parse_hash,import_report) VALUES($1,NULLIF($2,'')::date,NULLIF($3,'')::date,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING id`, filename, res.PeriodStart, res.PeriodEnd, opening, closing, res.Parser, res.Validated, accountID, owner, res.DocumentHash, parseFingerprint(res), string(raw)).Scan(&statementID)
	if e != nil {
		return 0, 0, 0, e
	}
	for i, t := range res.Transactions {
		var domain *int64
		if id, ok := eng.Match(t.Label); ok {
			domain = &id
		}
		n := normalize.Label(t.Label, t.Amount)
		if alias, ok := s.aliasFor(owner, n.Merchant, t.Label); ok {
			n.Display = alias
		}
		amount, _ := money.FromLegacy(t.Amount)
		_, e = tx.Exec(`INSERT INTO transactions(statement_id,tx_date,label,amount,domain_id,hash,owner_id,display_label,merchant,label_kind,classification_source,source_business_hash,source_index,source_page,source_line,value_date) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,NULLIF($14,0),NULLIF($15,0),NULLIF($16,'')::date)`, statementID, t.Date, t.Label, amount.String(), domain, parser.SourceHash(res.DocumentHash, i, accountID), owner, n.Display, n.Merchant, n.Kind, classificationForRule(domain), parser.TxHash(t), i, t.SourcePage, t.SourceLine, t.ValueDate)
		if e != nil {
			return 0, 0, 0, fmt.Errorf("échec d'insertion : import entièrement annulé")
		}
	}
	if e = tx.Commit(); e != nil {
		return 0, 0, 0, e
	}
	res.Report = &report
	return len(res.Transactions), 0, statementID, nil
}

func appendUniqueReportWarning(warnings []string, value string) []string {
	for _, warning := range warnings {
		if warning == value {
			return warnings
		}
	}
	return append(warnings, value)
}
