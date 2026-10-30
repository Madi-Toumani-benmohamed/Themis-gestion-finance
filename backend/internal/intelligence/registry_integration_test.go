package intelligence_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/lib/pq"
	"themis/internal/db"
	"themis/internal/intelligence"
)

func TestRegistryOwnerIsolationAndIdempotentHistoryPostgres(t *testing.T) {
	dsn := os.Getenv("THEMIS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("THEMIS_TEST_DATABASE_URL non défini")
	}
	admin, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := fmt.Sprintf("intelligence_registry_%d", time.Now().UnixNano())
	if _, err = admin.Exec(`CREATE SCHEMA ` + pq.QuoteIdentifier(schema)); err != nil {
		t.Fatal(err)
	}
	defer admin.Exec(`DROP SCHEMA ` + pq.QuoteIdentifier(schema) + ` CASCADE`)
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	query := u.Query()
	query.Set("search_path", schema)
	u.RawQuery = query.Encode()
	store, err := sql.Open("postgres", u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	store.SetMaxOpenConns(4)
	if err = db.Migrate(store); err != nil {
		t.Fatal(err)
	}
	if _, err = store.Exec(`INSERT INTO users(id,username,password_hash,role) VALUES(1,'TEST INTEL 1','TEST','user'),(2,'TEST INTEL 2','TEST','user')`); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err = intelligence.Enqueue(ctx, store, 1, "statement_import"); err != nil {
		t.Fatal(err)
	}
	if err = intelligence.Enqueue(ctx, store, 1, "category_change"); err != nil {
		t.Fatal(err)
	}
	job, err := intelligence.Claim(ctx, store)
	if err != nil || job == nil || job.OwnerID != 1 || job.Trigger != "category_change" || job.Attempts != 1 {
		t.Fatalf("coalesced job claim mismatch: %+v %v", job, err)
	}
	if err = intelligence.Enqueue(ctx, store, 1, "tag_change"); err != nil {
		t.Fatal(err)
	}
	if err = intelligence.CompleteJob(ctx, store, 1); err != nil {
		t.Fatal(err)
	}
	job, err = intelligence.Claim(ctx, store)
	if err != nil || job == nil || job.Trigger != "tag_change" {
		t.Fatalf("mutation during run did not schedule one rerun: %+v %v", job, err)
	}
	if err = intelligence.CompleteJob(ctx, store, 1); err != nil {
		t.Fatal(err)
	}
	jobStatus, err := intelligence.GetJobStatus(ctx, store, 1)
	if err != nil || jobStatus.Status != "done" || jobStatus.LastSuccessAt == nil {
		t.Fatalf("job status mismatch: %+v %v", jobStatus, err)
	}
	if other, err := intelligence.GetJobStatus(ctx, store, 2); err != nil || other.Status != "idle" {
		t.Fatalf("job status crossed owner boundary: %+v %v", other, err)
	}
	discovery := intelligence.Discovery{
		OwnerID: 1, Kind: "recurring", SubjectKey: "merchant:synthetic-shop", Period: "2026-09",
		Title: "Hausse observée", Summary: "Quatre périodes complètes comparées.", Confidence: 80,
		RelevanceScore: 71, ImpactScore: 63, Coverage: intelligence.Coverage{Status: "complete", Periods: 4},
	}
	first, err := intelligence.Record(ctx, store, discovery, "test-v1")
	if err != nil {
		t.Fatal(err)
	}
	replay, err := intelligence.Record(ctx, store, discovery, "test-v1")
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != replay.ID || first.ObservationID != replay.ObservationID || first.RuleVersion != "test-v1" {
		t.Fatal("identical recompute changed stable identities")
	}
	secondVersion := discovery
	secondVersion.Summary = "Une observation distincte, toujours synthétique."
	second, err := intelligence.Record(ctx, store, secondVersion, "test-v1")
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != first.ID || second.ObservationID == first.ObservationID {
		t.Fatal("new observation did not retain phenomenon identity and version evidence")
	}
	feed, err := intelligence.Feed(ctx, store, 1, 3)
	if err != nil || len(feed) != 1 || feed[0].ID != first.ID || feed[0].Summary != secondVersion.Summary {
		t.Fatalf("owner feed mismatch: %#v, %v", feed, err)
	}
	history, err := intelligence.History(ctx, store, 1, first.ID, 100)
	if err != nil || len(history) != 2 {
		t.Fatalf("observation history mismatch: %#v, %v", history, err)
	}
	stale := discovery
	stale.SubjectKey = "merchant:stale-synthetic-shop"
	stale.Title = "Ancienne observation"
	staleRecord, err := intelligence.Record(ctx, store, stale, "test-v1")
	if err != nil {
		t.Fatal(err)
	}
	if err = intelligence.ReconcileActive(ctx, store, 1, "recurring", []string{first.ID}); err != nil {
		t.Fatal(err)
	}
	resolved, err := intelligence.Get(ctx, store, 1, staleRecord.ID)
	if err != nil || resolved.State != "resolved" {
		t.Fatalf("stale discovery state = %q, %v", resolved.State, err)
	}
	if _, err = store.Exec(`UPDATE intelligence_discoveries SET state='hidden' WHERE owner_id=1 AND id=$1`, first.ID); err != nil {
		t.Fatal(err)
	}
	if err = intelligence.ReconcileActive(ctx, store, 1, "recurring", []string{first.ID}); err != nil {
		t.Fatal(err)
	}
	itemAfterReconcile, err := intelligence.Get(ctx, store, 1, first.ID)
	if err != nil || itemAfterReconcile.State != "hidden" {
		t.Fatalf("hidden discovery reappeared: state=%q, err=%v", itemAfterReconcile.State, err)
	}
	if state, err := intelligence.Feedback(ctx, store, 1, first.ID, "interesting"); err != nil || state != "hidden" {
		t.Fatalf("non-lifecycle feedback changed state: %q, %v", state, err)
	}
	preferences, err := intelligence.Preferences(ctx, store, 1, 100)
	if err != nil || len(preferences) != 1 || preferences[0].Action != "interesting" {
		t.Fatalf("feedback preference was not persisted: %+v, %v", preferences, err)
	}
	if err = intelligence.SetPreference(ctx, store, 1, first.ID, "irrelevant", "synthetic exception"); err != nil {
		t.Fatal(err)
	}
	preferences, err = intelligence.Preferences(ctx, store, 1, 100)
	if err != nil || len(preferences) != 1 || preferences[0].Action != "irrelevant" || preferences[0].Note != "synthetic exception" {
		t.Fatalf("preference edit was not persisted: %+v, %v", preferences, err)
	}
	personalized, err := intelligence.ApplyPreference(ctx, store, discovery)
	if err != nil || personalized.RelevanceScore != discovery.RelevanceScore-12 || personalized.Confidence != discovery.Confidence {
		t.Fatalf("explicit feedback changed more than priority: %+v, %v", personalized, err)
	}
	peer := discovery
	peer.SubjectKey = "merchant:other-synthetic-shop"
	peerScore, err := intelligence.ApplyPreference(ctx, store, peer)
	if err != nil || peerScore.RelevanceScore != discovery.RelevanceScore-3 || peerScore.Confidence != discovery.Confidence {
		t.Fatalf("similar signal preference penalty mismatch: %+v, %v", peerScore, err)
	}
	if err = intelligence.ClearPreference(ctx, store, 1, first.ID); err != nil {
		t.Fatal(err)
	}
	preferences, err = intelligence.Preferences(ctx, store, 1, 100)
	if err != nil || len(preferences) != 0 {
		t.Fatalf("cleared preference remained visible: %+v, %v", preferences, err)
	}
	if state, err := intelligence.Feedback(ctx, store, 2, first.ID, "hide"); err != intelligence.ErrNotFound {
		t.Fatalf("cross-owner feedback result = %q, %v", state, err)
	}
	if state, err := intelligence.Feedback(ctx, store, 1, first.ID, "unhide"); err != nil || state != "resolved" {
		t.Fatalf("unhide should await evidence recomputation: %q, %v", state, err)
	}
	if err = intelligence.ReconcileActive(ctx, store, 1, "recurring", []string{first.ID}); err != nil {
		t.Fatal(err)
	}
	itemAfterUnhide, err := intelligence.Get(ctx, store, 1, first.ID)
	if err != nil || itemAfterUnhide.State != "active" {
		t.Fatalf("current evidence did not reactivate unhidden discovery: state=%q, err=%v", itemAfterUnhide.State, err)
	}
	otherOwner, err := intelligence.Feed(ctx, store, 2, 3)
	if err != nil || len(otherOwner) != 0 {
		t.Fatalf("cross-owner feed leak: %#v, %v", otherOwner, err)
	}
	if _, err = intelligence.Get(ctx, store, 2, first.ID); err != intelligence.ErrNotFound {
		t.Fatalf("cross-owner detail result = %v, want not found", err)
	}
	var statement1, statement2 int64
	if err = store.QueryRow(`INSERT INTO statements(owner_id,filename) VALUES(1,'SYNTHETIC 1') RETURNING id`).Scan(&statement1); err != nil {
		t.Fatal(err)
	}
	if err = store.QueryRow(`INSERT INTO statements(owner_id,filename) VALUES(2,'SYNTHETIC 2') RETURNING id`).Scan(&statement2); err != nil {
		t.Fatal(err)
	}
	var transaction1, transaction2 int64
	if err = store.QueryRow(`INSERT INTO transactions(statement_id,owner_id,tx_date,label,amount,hash) VALUES($1,1,'2026-09-01','SYNTHETIC 1',-1.00,'intel-test-1') RETURNING id`, statement1).Scan(&transaction1); err != nil {
		t.Fatal(err)
	}
	if err = store.QueryRow(`INSERT INTO transactions(statement_id,owner_id,tx_date,label,amount,hash) VALUES($1,2,'2026-09-01','SYNTHETIC 2',-1.00,'intel-test-2') RETURNING id`, statement2).Scan(&transaction2); err != nil {
		t.Fatal(err)
	}
	withEvidence := discovery
	withEvidence.SubjectKey = "merchant:evidence-owner-check"
	withEvidence.Evidence = intelligence.Evidence{TransactionIDs: []int64{transaction1, transaction2}, Periods: []string{"2026-09"}, Source: "test"}
	if _, err = intelligence.Record(ctx, store, withEvidence, "test-v1"); err == nil {
		t.Fatal("cross-owner evidence was accepted")
	}
	withEvidence.Evidence.TransactionIDs = []int64{transaction1}
	if _, err = intelligence.Record(ctx, store, withEvidence, "test-v1"); err != nil {
		t.Fatalf("owner evidence was rejected: %v", err)
	}
	release, acquired, err := intelligence.AcquireRecomputeLock(ctx, store, 1)
	if err != nil || !acquired {
		t.Fatalf("first recompute lock = %v, %v", acquired, err)
	}
	_, acquired, err = intelligence.AcquireRecomputeLock(ctx, store, 1)
	if err != nil || acquired {
		t.Fatalf("concurrent recompute lock was not rejected: %v, %v", acquired, err)
	}
	release()
	releaseAgain, acquired, err := intelligence.AcquireRecomputeLock(ctx, store, 1)
	if err != nil || !acquired {
		t.Fatalf("recompute lock was not released: %v, %v", acquired, err)
	}
	releaseAgain()
}
