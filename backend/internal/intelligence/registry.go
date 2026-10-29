package intelligence

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/lib/pq"
)

const maxEvidenceReferences = 500

var ErrNotFound = errors.New("discovery not found")
var ErrRecomputeBusy = errors.New("intelligence recompute already running")

// AcquireRecomputeLock serializes a manual recalculation per owner across API
// processes. The session lock is always released before returning the
// connection to database/sql's pool.
func AcquireRecomputeLock(ctx context.Context, db *sql.DB, ownerID int64) (func(), bool, error) {
	if db == nil || ownerID <= 0 {
		return nil, false, errors.New("invalid recompute lock owner")
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, false, err
	}
	var acquired bool
	if err = conn.QueryRowContext(ctx, `SELECT pg_try_advisory_lock(hashtextextended('themis-intelligence-recompute:'||$1::text,0))`, ownerID).Scan(&acquired); err != nil {
		_ = conn.Close()
		return nil, false, err
	}
	if !acquired {
		_ = conn.Close()
		return nil, false, nil
	}
	release := func() {
		releaseCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_, _ = conn.ExecContext(releaseCtx, `SELECT pg_advisory_unlock(hashtextextended('themis-intelligence-recompute:'||$1::text,0))`, ownerID)
		_ = conn.Close()
	}
	return release, true, nil
}

// StableID identifies a phenomenon for one owner. Raw subject keys never
// become public identifiers and cannot collide across owners.
func StableID(ownerID int64, kind, subjectKey string) string {
	return digest(fmt.Sprintf("%d\x00%s\x00%s", ownerID, kind, subjectKey))
}

type observationFingerprint struct {
	Period      string
	Title       string
	Summary     string
	Severity    string
	Confidence  int
	Relevance   int
	Impact      int
	Metrics     []Metric
	Evidence    Evidence
	Coverage    Coverage
	Limitations []string
	RuleVersion string
}

// Record appends an immutable observation and upserts its stable phenomenon
// identity in one transaction. Replaying identical detector output is safe.
func Record(ctx context.Context, db *sql.DB, d Discovery, ruleVersion string) (Discovery, error) {
	if db == nil || d.OwnerID <= 0 || strings.TrimSpace(d.Kind) == "" || len(d.Kind) > 80 || strings.TrimSpace(d.SubjectKey) == "" || strings.TrimSpace(d.Title) == "" {
		return Discovery{}, errors.New("invalid discovery identity")
	}
	if len(d.Title) > 500 || len(d.Summary) > 4000 || len(d.SubjectKey) > 500 || len(d.Severity) > 80 || len(d.Period) > 30 || len(d.Metrics) > 50 || len(d.Evidence.Source) > 120 || len(d.Evidence.TransactionIDs) > maxEvidenceReferences || len(d.Evidence.Periods) > maxEvidenceReferences {
		return Discovery{}, errors.New("discovery exceeds registry limits")
	}
	for _, metric := range d.Metrics {
		if strings.TrimSpace(metric.Name) == "" || len(metric.Name) > 100 || (metric.Kind != MetricMoneyCents && metric.Kind != MetricCount && metric.Kind != MetricPercent && metric.Kind != MetricDuration) {
			return Discovery{}, errors.New("invalid discovery metric")
		}
	}
	if d.Coverage.Status != "complete" && d.Coverage.Status != "partial" && d.Coverage.Status != "unknown" {
		return Discovery{}, errors.New("invalid discovery coverage")
	}
	if d.Confidence < 0 || d.Confidence > 100 || d.RelevanceScore < 0 || d.RelevanceScore > 100 || d.ImpactScore < 0 || d.ImpactScore > 100 {
		return Discovery{}, errors.New("invalid discovery score")
	}
	if d.State == "" {
		d.State = "active"
	}
	if d.State != "active" && d.State != "hidden" && d.State != "resolved" {
		return Discovery{}, errors.New("invalid discovery state")
	}
	d.Evidence.TransactionIDs = uniqueSortedIDs(d.Evidence.TransactionIDs)
	if (len(d.Evidence.TransactionIDs) == 0 && (len(d.Evidence.Periods) == 0 || d.Evidence.Source == "")) || d.Coverage.Status != "complete" {
		if d.RelevanceScore > 49 {
			d.RelevanceScore = 49
		}
	}
	if d.Metrics == nil {
		d.Metrics = []Metric{}
	}
	if d.Limitations == nil {
		d.Limitations = []string{}
	}
	if ruleVersion == "" {
		ruleVersion = "1"
	}
	d.RuleVersion = ruleVersion
	d.ID = StableID(d.OwnerID, d.Kind, d.SubjectKey)
	now := time.Now().UTC()
	if d.DetectedAt.IsZero() {
		d.DetectedAt = now
	}
	d.UpdatedAt = now
	fingerprint, err := json.Marshal(observationFingerprint{
		Period: d.Period, Title: d.Title, Summary: d.Summary, Severity: d.Severity,
		Confidence: d.Confidence, Relevance: d.RelevanceScore, Impact: d.ImpactScore,
		Metrics: d.Metrics, Evidence: d.Evidence, Coverage: d.Coverage,
		Limitations: d.Limitations, RuleVersion: ruleVersion,
	})
	if err != nil {
		return Discovery{}, fmt.Errorf("encode discovery: %w", err)
	}
	d.ObservationID = digest(string(fingerprint))
	metrics, err := json.Marshal(d.Metrics)
	if err != nil {
		return Discovery{}, err
	}
	evidence, err := json.Marshal(d.Evidence)
	if err != nil {
		return Discovery{}, err
	}
	coverage, err := json.Marshal(d.Coverage)
	if err != nil {
		return Discovery{}, err
	}
	limitations, err := json.Marshal(d.Limitations)
	if err != nil {
		return Discovery{}, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return Discovery{}, err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO intelligence_discoveries(owner_id,id,kind,subject_key,state,detected_at,updated_at)
 VALUES($1,$2,$3,$4,$5,$6,$7)
	 ON CONFLICT(owner_id,kind,subject_key) DO NOTHING`,
		d.OwnerID, d.ID, d.Kind, d.SubjectKey, d.State, d.DetectedAt, d.UpdatedAt)
	if err != nil {
		return Discovery{}, fmt.Errorf("upsert discovery: %w", err)
	}
	if len(d.Evidence.TransactionIDs) > 0 {
		var owned int
		if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM transactions WHERE owner_id=$1 AND id=ANY($2)`, d.OwnerID, pq.Array(d.Evidence.TransactionIDs)).Scan(&owned); err != nil {
			return Discovery{}, fmt.Errorf("verify discovery evidence: %w", err)
		}
		if owned != len(d.Evidence.TransactionIDs) {
			return Discovery{}, errors.New("discovery evidence references unavailable transactions")
		}
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO intelligence_observations(owner_id,id,discovery_id,period,title,summary,severity,confidence,relevance_score,impact_score,metrics,evidence,coverage,limitations,rule_version,detected_at,updated_at)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11::jsonb,$12::jsonb,$13::jsonb,$14::jsonb,$15,$16,$17)
 ON CONFLICT(owner_id,discovery_id,id) DO NOTHING`, d.OwnerID, d.ObservationID, d.ID, d.Period, d.Title, d.Summary, d.Severity, d.Confidence, d.RelevanceScore, d.ImpactScore, metrics, evidence, coverage, limitations, ruleVersion, d.DetectedAt, d.UpdatedAt)
	if err != nil {
		return Discovery{}, fmt.Errorf("append observation: %w", err)
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return Discovery{}, err
	}
	if inserted > 0 {
		if _, err = tx.ExecContext(ctx, `UPDATE intelligence_discoveries SET updated_at=$3 WHERE owner_id=$1 AND id=$2`, d.OwnerID, d.ID, d.UpdatedAt); err != nil {
			return Discovery{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return Discovery{}, err
	}
	return d, nil
}

// Feed returns the latest active observation for an owner, with a strict
// upper bound suitable for a dashboard response.
func Feed(ctx context.Context, db *sql.DB, ownerID int64, limit int) ([]Discovery, error) {
	if ownerID <= 0 || limit < 1 || limit > 50 {
		return nil, errors.New("invalid discovery feed parameters")
	}
	rows, err := db.QueryContext(ctx, `SELECT d.id,d.owner_id,d.kind,d.subject_key,d.state,d.detected_at,d.updated_at,
 o.id,o.period,o.title,o.summary,o.severity,o.confidence,o.relevance_score,o.impact_score,
 o.metrics,o.evidence,o.coverage,o.limitations,o.rule_version,o.detected_at,o.updated_at
 FROM intelligence_discoveries d
 JOIN LATERAL (SELECT * FROM intelligence_observations x
  WHERE x.owner_id=d.owner_id AND x.discovery_id=d.id
  ORDER BY x.updated_at DESC,x.id DESC LIMIT 1) o ON true
	 WHERE d.owner_id=$1 AND d.state='active'
 ORDER BY o.relevance_score DESC,o.confidence DESC,o.updated_at DESC,d.id
 LIMIT $2`, ownerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]Discovery, 0, limit)
	for rows.Next() {
		item, err := scanDiscovery(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func Get(ctx context.Context, db *sql.DB, ownerID int64, id string) (Discovery, error) {
	if ownerID <= 0 || len(id) != 64 {
		return Discovery{}, ErrNotFound
	}
	row := db.QueryRowContext(ctx, `SELECT d.id,d.owner_id,d.kind,d.subject_key,d.state,d.detected_at,d.updated_at,
 o.id,o.period,o.title,o.summary,o.severity,o.confidence,o.relevance_score,o.impact_score,
 o.metrics,o.evidence,o.coverage,o.limitations,o.rule_version,o.detected_at,o.updated_at
 FROM intelligence_discoveries d
 JOIN LATERAL (SELECT * FROM intelligence_observations x
  WHERE x.owner_id=d.owner_id AND x.discovery_id=d.id
  ORDER BY x.updated_at DESC,x.id DESC LIMIT 1) o ON true
 WHERE d.owner_id=$1 AND d.id=$2`, ownerID, id)
	item, err := scanDiscovery(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Discovery{}, ErrNotFound
	}
	return item, err
}

// History returns immutable versions for one owner-scoped phenomenon.
func History(ctx context.Context, db *sql.DB, ownerID int64, id string, limit int) ([]Discovery, error) {
	if ownerID <= 0 || len(id) != 64 || limit < 1 || limit > 100 {
		return nil, ErrNotFound
	}
	rows, err := db.QueryContext(ctx, `SELECT d.id,d.owner_id,d.kind,d.subject_key,d.state,o.detected_at,o.updated_at,
 o.id,o.period,o.title,o.summary,o.severity,o.confidence,o.relevance_score,o.impact_score,
 o.metrics,o.evidence,o.coverage,o.limitations,o.rule_version,o.detected_at,o.updated_at
 FROM intelligence_discoveries d
 JOIN intelligence_observations o ON o.owner_id=d.owner_id AND o.discovery_id=d.id
 WHERE d.owner_id=$1 AND d.id=$2
 ORDER BY o.updated_at DESC,o.id DESC LIMIT $3`, ownerID, id, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]Discovery, 0, limit)
	for rows.Next() {
		item, err := scanDiscovery(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, ErrNotFound
	}
	return items, nil
}

// ReconcileActive retires disappeared observations for one detector and
// reactivates its current findings, while preserving user-hidden phenomena.
func ReconcileActive(ctx context.Context, db *sql.DB, ownerID int64, kind string, currentIDs []string) error {
	if ownerID <= 0 || strings.TrimSpace(kind) == "" || len(kind) > 80 || len(currentIDs) > 500 {
		return errors.New("invalid discovery reconciliation")
	}
	for _, id := range currentIDs {
		if len(id) != 64 {
			return errors.New("invalid discovery identity")
		}
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `UPDATE intelligence_discoveries SET state='resolved',updated_at=now()
 WHERE owner_id=$1 AND kind=$2 AND state='active' AND id<>ALL($3)`, ownerID, kind, pq.Array(currentIDs)); err != nil {
		return err
	}
	if len(currentIDs) > 0 {
		if _, err = tx.ExecContext(ctx, `UPDATE intelligence_discoveries SET state='active',updated_at=now()
 WHERE owner_id=$1 AND kind=$2 AND state='resolved' AND id=ANY($3)`, ownerID, kind, pq.Array(currentIDs)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// Feedback appends the user's explicit response and applies only the lifecycle
// actions that are safe in this lot. Other feedback remains durable for later
// relevance tuning.
func Feedback(ctx context.Context, db *sql.DB, ownerID int64, id, action string) (string, error) {
	allowed := map[string]bool{"interesting": true, "already_knew": true, "irrelevant": true, "normal": true, "remind_me": true, "hide": true, "unhide": true}
	if ownerID <= 0 || len(id) != 64 || !allowed[action] {
		return "", errors.New("invalid discovery feedback")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	var state string
	err = tx.QueryRowContext(ctx, `SELECT state FROM intelligence_discoveries WHERE owner_id=$1 AND id=$2 FOR UPDATE`, ownerID, id).Scan(&state)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO intelligence_feedback(owner_id,discovery_id,action) VALUES($1,$2,$3)`, ownerID, id, action); err != nil {
		return "", err
	}
	if action != "hide" && action != "unhide" {
		if _, err = tx.ExecContext(ctx, `INSERT INTO intelligence_preferences(owner_id,discovery_id,action) VALUES($1,$2,$3)
 ON CONFLICT(owner_id,discovery_id) DO UPDATE SET action=EXCLUDED.action,note='',updated_at=now()`, ownerID, id, action); err != nil {
			return "", err
		}
	}
	switch action {
	case "hide":
		state = "hidden"
	case "unhide":
		// A later recompute reactivates this only if its evidence still exists.
		state = "resolved"
	}
	if action == "hide" || action == "unhide" {
		if _, err = tx.ExecContext(ctx, `UPDATE intelligence_discoveries SET state=$3,updated_at=now() WHERE owner_id=$1 AND id=$2`, ownerID, id, state); err != nil {
			return "", err
		}
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return state, nil
}

type Preference struct {
	DiscoveryID string    `json:"discovery_id"`
	Kind        string    `json:"kind"`
	Title       string    `json:"title"`
	Action      string    `json:"action"`
	Note        string    `json:"note"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func Preferences(ctx context.Context, db *sql.DB, ownerID int64, limit int) ([]Preference, error) {
	if ownerID <= 0 || limit < 1 || limit > 100 {
		return nil, errors.New("invalid intelligence preference query")
	}
	rows, err := db.QueryContext(ctx, `SELECT p.discovery_id,d.kind,o.title,p.action,p.note,p.updated_at
 FROM intelligence_preferences p JOIN intelligence_discoveries d ON d.owner_id=p.owner_id AND d.id=p.discovery_id
 JOIN LATERAL (SELECT title FROM intelligence_observations x WHERE x.owner_id=p.owner_id AND x.discovery_id=p.discovery_id ORDER BY updated_at DESC,id DESC LIMIT 1) o ON true
 WHERE p.owner_id=$1 ORDER BY p.updated_at DESC,p.discovery_id LIMIT $2`, ownerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Preference{}
	for rows.Next() {
		var item Preference
		if err = rows.Scan(&item.DiscoveryID, &item.Kind, &item.Title, &item.Action, &item.Note, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func SetPreference(ctx context.Context, db *sql.DB, ownerID int64, id, action, note string) error {
	allowed := map[string]bool{"interesting": true, "already_knew": true, "irrelevant": true, "normal": true, "remind_me": true}
	if ownerID <= 0 || len(id) != 64 || len(note) > 500 || !allowed[action] {
		return errors.New("invalid intelligence preference")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `INSERT INTO intelligence_preferences(owner_id,discovery_id,action,note) VALUES($1,$2,$3,$4)
 ON CONFLICT(owner_id,discovery_id) DO UPDATE SET action=EXCLUDED.action,note=EXCLUDED.note,updated_at=now()`, ownerID, id, action, note); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO intelligence_feedback(owner_id,discovery_id,action) VALUES($1,$2,$3)`, ownerID, id, action); err != nil {
		return err
	}
	return tx.Commit()
}

func ClearPreference(ctx context.Context, db *sql.DB, ownerID int64, id string) error {
	if ownerID <= 0 || len(id) != 64 {
		return errors.New("invalid intelligence preference identity")
	}
	_, err := db.ExecContext(ctx, `DELETE FROM intelligence_preferences WHERE owner_id=$1 AND discovery_id=$2`, ownerID, id)
	return err
}

// ApplyPreference adjusts priority only. Confidence and evidence remain untouched.
func ApplyPreference(ctx context.Context, db *sql.DB, d Discovery) (Discovery, error) {
	d.ID = StableID(d.OwnerID, d.Kind, d.SubjectKey)
	var action sql.NullString
	var similarIrrelevant int
	err := db.QueryRowContext(ctx, `SELECT (SELECT p.action FROM intelligence_preferences p
  JOIN intelligence_discoveries d ON d.owner_id=p.owner_id AND d.id=p.discovery_id
  WHERE d.owner_id=$1 AND d.kind=$2 AND d.subject_key=$4),
 (SELECT count(*) FROM intelligence_preferences p2 JOIN intelligence_discoveries d2 ON d2.owner_id=p2.owner_id AND d2.id=p2.discovery_id
  WHERE p2.owner_id=$1 AND d2.kind=$2 AND p2.action='irrelevant' AND p2.discovery_id<>$3)
	`, d.OwnerID, d.Kind, d.ID, d.SubjectKey).Scan(&action, &similarIrrelevant)
	if err != nil {
		return Discovery{}, err
	}
	delta := 0
	if action.Valid {
		switch action.String {
		case "interesting":
			delta = 8
		case "already_knew":
			delta = -8
		case "irrelevant":
			delta = -12
		case "normal":
			delta = -6
		case "remind_me":
			delta = 5
		}
	}
	if similarIrrelevant > 3 {
		similarIrrelevant = 3
	}
	delta -= similarIrrelevant * 3
	d.RelevanceScore += delta
	if d.RelevanceScore < 0 {
		d.RelevanceScore = 0
	}
	if d.RelevanceScore > 100 {
		d.RelevanceScore = 100
	}
	return d, nil
}

type scanner interface{ Scan(...any) error }

func scanDiscovery(row scanner) (Discovery, error) {
	var d Discovery
	var metrics, evidence, coverage, limitations []byte
	var observationDetected, observationUpdated time.Time
	err := row.Scan(&d.ID, &d.OwnerID, &d.Kind, &d.SubjectKey, &d.State, &d.DetectedAt, &d.UpdatedAt,
		&d.ObservationID, &d.Period, &d.Title, &d.Summary, &d.Severity, &d.Confidence, &d.RelevanceScore, &d.ImpactScore,
		&metrics, &evidence, &coverage, &limitations, &d.RuleVersion, &observationDetected, &observationUpdated)
	if err != nil {
		return Discovery{}, err
	}
	if err = json.Unmarshal(metrics, &d.Metrics); err != nil {
		return Discovery{}, err
	}
	if err = json.Unmarshal(evidence, &d.Evidence); err != nil {
		return Discovery{}, err
	}
	if err = json.Unmarshal(coverage, &d.Coverage); err != nil {
		return Discovery{}, err
	}
	if err = json.Unmarshal(limitations, &d.Limitations); err != nil {
		return Discovery{}, err
	}
	return d, nil
}

func uniqueSortedIDs(ids []int64) []int64 {
	if len(ids) < 2 {
		return ids
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	out := ids[:1]
	for _, id := range ids[1:] {
		if id != out[len(out)-1] {
			out = append(out, id)
		}
	}
	return out
}

func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
