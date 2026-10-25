package memory

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestMemoryHealthStatesAndRedirection(t *testing.T) {
	var calls atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/health" {
			t.Error("health queried graph facts")
		}
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()
	client := New(Config{Enabled: false, URL: server.URL, RequestTimeout: time.Second})
	if client.IsReady(context.Background()) || calls.Load() != 0 {
		t.Fatal("disabled memory queried")
	}
	client = New(Config{Enabled: true, URL: server.URL, RequestTimeout: time.Second})
	if !client.IsReady(context.Background()) || calls.Load() != 1 {
		t.Fatal("healthy memory unavailable")
	}
	client = New(Config{Enabled: true, URL: server.URL + "/", RequestTimeout: time.Second})
	if !client.IsReady(context.Background()) || calls.Load() != 2 {
		t.Fatal("trailing slash broke the health endpoint")
	}
	invalid := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`<html>private-token</html>`)) }))
	defer invalid.Close()
	client = New(Config{Enabled: true, URL: invalid.URL, RequestTimeout: time.Second})
	if client.IsReady(context.Background()) {
		t.Fatal("HTTP 200 HTML is not memory health")
	}
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, server.URL, 302) }))
	defer redirect.Close()
	client = New(Config{Enabled: true, URL: redirect.URL, RequestTimeout: time.Second})
	if client.IsReady(context.Background()) || calls.Load() != 2 {
		t.Fatal("memory health followed redirect")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	client = New(Config{Enabled: true, URL: server.URL, RequestTimeout: time.Second})
	if client.IsReady(ctx) {
		t.Fatal("cancelled health succeeded")
	}
}
