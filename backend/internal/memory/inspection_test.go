package memory

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func sampleResult(group string) Result {
	return Result{ID: "edge-" + group, GroupID: group, Source: "a-" + group, Target: "b-" + group, Type: "RECURRENT", Fact: "SYNTHETIC OWNER " + group, EpisodeIDs: []string{"ep-" + group}, Sources: []Source{{ID: "ep-" + group, GroupID: group, Name: "Import TEST", Source: "statement:101"}}}
}
func sampleGraph(group string) Graph {
	nodes, edges, episodes := int64(2), int64(1), int64(1)
	return Graph{GroupID: group, NodeCount: &nodes, EdgeCount: &edges, EpisodeCount: &episodes, Nodes: []Node{{ID: "a-" + group, GroupID: group, Name: "Merchant TEST"}, {ID: "b-" + group, GroupID: group, Name: "Habit TEST"}}, Edges: []Result{sampleResult(group)}, Sources: []Source{{ID: "ep-" + group, GroupID: group, Name: "Import TEST", Source: "statement:101"}}}
}
func TestMemoryStrictResultsFailuresAndUnknownCounts(t *testing.T) {
	var mode atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch mode.Load() {
		case 1:
			w.WriteHeader(500)
			w.Write([]byte(`{"detail":{"code":"model_missing","secret":"PRIVATE PROVIDER"}}`))
			return
		case 2:
			w.Write([]byte(`{"results":[],"count":0}`))
			return
		case 3:
			json.NewEncoder(w).Encode(map[string]any{"results": []Result{sampleResult("2")}, "count": 1})
			return
		case 4:
			w.Write([]byte(`{"group_id":"1","nodes":[],"edges":[],"sources":[]}`))
			return
		}
		if r.URL.Path == "/kg/inspect/1" {
			json.NewEncoder(w).Encode(sampleGraph("1"))
		} else if r.URL.Path == "/kg/query" {
			json.NewEncoder(w).Encode(map[string]any{"results": []Result{sampleResult("1")}, "count": 1})
		} else {
			w.Write([]byte(`{"entities_added":2,"edges_added":1}`))
		}
	}))
	defer server.Close()
	c := New(Config{Enabled: true, URL: server.URL, RequestTimeout: time.Second})
	graph, e := c.Inspect(context.Background(), "1")
	if e != nil || *graph.NodeCount != 2 {
		t.Fatal(graph, e)
	}
	facts, e := c.Query(context.Background(), "1", "test", 5)
	if e != nil || len(facts) != 1 || facts[0].Score != nil {
		t.Fatal(facts, e)
	}
	n, m, e := c.Learn(context.Background(), "1", "transaction_import", []string{"TEST"})
	if e != nil || n != 2 || m != 1 {
		t.Fatal(n, m, e)
	}
	mode.Store(1)
	_, e = c.Query(context.Background(), "1", "test", 5)
	if FailureCode(e) != "model_missing" || strings.Contains(e.Error(), "PRIVATE") {
		t.Fatal(e)
	}
	mode.Store(2)
	facts, e = c.Query(context.Background(), "1", "test", 5)
	if e != nil || facts == nil || len(facts) != 0 {
		t.Fatal("explicit empty result", facts, e)
	}
	mode.Store(3)
	_, e = c.Query(context.Background(), "1", "test", 5)
	if FailureCode(e) != "scope" {
		t.Fatal("foreign scope accepted", e)
	}
	mode.Store(4)
	graph, e = c.Inspect(context.Background(), "1")
	if e == nil || graph != nil {
		t.Fatal("missing counts fabricated as zero")
	}
}
func TestMemoryDisabledTimeoutProtocolAndRedirect(t *testing.T) {
	var calls atomic.Int64
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.Write([]byte(`{}`)) }))
	defer target.Close()
	c := New(Config{Enabled: false, URL: target.URL, RequestTimeout: time.Second})
	if _, e := c.Query(context.Background(), "1", "test", 5); FailureCode(e) != "disabled" || calls.Load() != 0 {
		t.Fatal(e, calls.Load())
	}
	c = New(Config{Enabled: true, URL: target.URL, RequestTimeout: time.Second})
	if _, e := c.Query(context.Background(), "1'", "test", 5); FailureCode(e) != "protocol" || calls.Load() != 0 {
		t.Fatal(e, calls.Load())
	}
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, 302) }))
	defer redirect.Close()
	c = New(Config{Enabled: true, URL: redirect.URL, RequestTimeout: time.Second})
	if _, e := c.Inspect(context.Background(), "1"); e == nil || calls.Load() != 0 {
		t.Fatal("redirect followed", e, calls.Load())
	}
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(100 * time.Millisecond):
		}
	}))
	defer slow.Close()
	c = New(Config{Enabled: true, URL: slow.URL, RequestTimeout: time.Second})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, _, e := c.Learn(ctx, "1", "transaction_import", []string{"TEST"}); FailureCode(e) != "timeout" {
		t.Fatal(e)
	}
}
