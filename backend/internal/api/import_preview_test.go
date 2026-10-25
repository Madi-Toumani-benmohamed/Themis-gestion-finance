package api

import (
	"encoding/json"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"themis/internal/models"
	"themis/internal/parser"
)

func TestDescribeImportExactCentDifference(t *testing.T) {
	opening, closing := 0.0, 0.3
	res := &models.ParseResult{OpeningBalance: &opening, ClosingBalance: &closing, Transactions: []models.Transaction{{Amount: 0.1}, {Amount: 0.2}}}
	if diff := describeImport(res, nil).BalanceDifference; diff == nil || *diff != 0 {
		t.Fatal("preview used inexact arithmetic", diff)
	}
	closing = 0.31
	if diff := describeImport(res, nil).BalanceDifference; diff == nil || *diff != 0.01 {
		t.Fatal("difference sign or cents incorrect", diff)
	}
}

func TestDescribeImportDuplicatesAndBalances(t *testing.T) {
	a := models.Transaction{Date: "2026-10-01", Label: "TEST A", Amount: -10}
	a.Hash = parser.TxHash(a)
	b := models.Transaction{Date: "2026-10-02", Label: "TEST B", Amount: 20}
	b.Hash = parser.TxHash(b)
	opening, closing := 100.0, 130.0
	res := &models.ParseResult{Transactions: []models.Transaction{a, b, b}, OpeningBalance: &opening, ClosingBalance: &closing}
	out := describeImport(res, map[string]bool{a.Hash: true})
	if out.NewCount != 1 || out.Duplicates != 2 || !out.Transactions[0].Duplicate || out.Transactions[1].Duplicate || !out.Transactions[2].Duplicate {
		t.Fatalf("incorrect duplicate counts: %+v", out)
	}
	if out.BalanceDifference == nil || math.Abs(*out.BalanceDifference) > 1e-9 {
		t.Fatal("balance check should use all extracted operations, including duplicate rows")
	}
	raw, _ := json.Marshal(out)
	if strings.Contains(string(raw), a.Hash) {
		t.Fatal("internal hash leaked")
	}
	res.OpeningBalance = nil
	if describeImport(res, nil).BalanceDifference != nil {
		t.Fatal("missing opening balance must not be invented")
	}
}

func TestImportPreviewOwnershipExpiryAndQuota(t *testing.T) {
	store := &importPreviewStore{}
	now := time.Now()
	pending := pendingImport{Owner: 1, Result: &models.ParseResult{}}
	token, err := store.put(pending, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := store.take(token, 2, now); ok {
		t.Fatal("another user consumed preview")
	}
	store.discard(token, 2)
	if _, ok := store.take(token, 1, now); !ok {
		t.Fatal("another user removed owner's preview")
	}
	if _, ok := store.take(token, 1, now); ok {
		t.Fatal("replayed preview accepted")
	}
	token, _ = store.put(pending, now)
	if _, ok := store.take(token, 1, now.Add(importPreviewTTL)); ok {
		t.Fatal("expired preview accepted")
	}
	for i := 0; i < 3; i++ {
		if _, err := store.put(pending, now); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.put(pending, now); err == nil {
		t.Fatal("per-owner quota not enforced")
	}
	if _, err := store.put(pending, now.Add(importPreviewTTL)); err != nil {
		t.Fatal("expired entries should release capacity")
	}
	store.mu.Lock()
	store.items["reserved"] = pendingImport{Owner: 2, Expires: now.Add(importPreviewTTL * 2), Size: importPreviewMaxBytes}
	store.mu.Unlock()
	if _, err := store.put(pending, now.Add(importPreviewTTL)); err == nil {
		t.Fatal("global memory budget not enforced")
	}
}

func TestImportPreviewConcurrentConfirmation(t *testing.T) {
	store := &importPreviewStore{}
	now := time.Now()
	token, err := store.put(pendingImport{Owner: 1, Result: &models.ParseResult{}}, now)
	if err != nil {
		t.Fatal(err)
	}
	var accepted atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, ok := store.take(token, 1, now); ok {
				accepted.Add(1)
			}
		}()
	}
	wg.Wait()
	if accepted.Load() != 1 {
		t.Fatalf("confirmed %d times", accepted.Load())
	}
}
