package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAIResponsesContract(t *testing.T) {
	for _, wantJSON := range []bool{false, true} {
		t.Run(map[bool]string{false: "text", true: "json"}[wantJSON], func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" || r.URL.Path != "/responses" || r.Header.Get("Authorization") != "Bearer test-key" {
					t.Error("incorrect endpoint or authentication")
				}
				var req map[string]any
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Fatal(err)
				}
				if req["store"] != false || req["input"] != "user" || req["model"] != "custom-model" || req["max_output_tokens"] != float64(16) {
					t.Errorf("incorrect request: %v", req)
				}
				if wantJSON {
					if !strings.Contains(req["instructions"].(string), "JSON") || req["text"].(map[string]any)["format"].(map[string]any)["type"] != "json_object" {
						t.Error("missing JSON configuration")
					}
				} else if req["text"] != nil {
					t.Error("text mode should not force JSON")
				}
				_, _ = w.Write([]byte(`{"status":"completed","error":null,"output":[{"type":"reasoning"},{"type":"message","content":[{"type":"output_text","text":"{\"ok\":true}"}]}]}`))
			}))
			defer ts.Close()
			client := &OpenAI{APIKey: "test-key", Model: "custom-model", endpoint: ts.URL + "/responses"}
			got, err := client.Complete(context.Background(), "system", "user", 4, wantJSON)
			if err != nil || got != `{"ok":true}` {
				t.Fatalf("got %q, %v", got, err)
			}
		})
	}
}

func TestOpenAIResponseFailures(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{"invalid key", 401, `{"error":{"message":"secret-key and private data"}}`},
		{"quota", 429, `{"error":{"message":"secret-key"}}`},
		{"server", 500, "private data"},
		{"invalid response", 200, "not JSON"},
		{"empty", 200, `{"status":"completed","output":[]}`},
		{"truncated", 200, `{"status":"incomplete","output":[{"type":"message","content":[{"type":"output_text","text":"partial"}]}]}`},
		{"refusal", 200, `{"status":"completed","output":[{"type":"message","content":[{"type":"refusal"}]}]}`},
		{"bad JSON output", 200, `{"status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"plain text"}]}]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer ts.Close()
			o := &OpenAI{APIKey: "secret-key", endpoint: ts.URL}
			got, err := o.Complete(context.Background(), "system", "user", 256, true)
			if err == nil || got != "" {
				t.Fatalf("expected failure, got %q, %v", got, err)
			}
			if strings.Contains(err.Error(), "secret-key") || strings.Contains(err.Error(), "private data") {
				t.Fatal("error leaks request or credential")
			}
		})
	}
}

func TestOpenAICancelledAndMissingKey(t *testing.T) {
	o := &OpenAI{}
	if _, err := o.Complete(context.Background(), "", "", 20, false); err == nil {
		t.Fatal("missing key accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	o.APIKey = "test-key"
	if _, err := o.Complete(ctx, "", "", 20, false); err == nil {
		t.Fatal("cancel ignored")
	}
}

func TestOpenAINoRedirect(t *testing.T) {
	redirected := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { redirected = true }))
	defer target.Close()
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer origin.Close()
	o := &OpenAI{APIKey: "secret-key", endpoint: origin.URL}
	if _, err := o.Complete(context.Background(), "", "", 20, false); err == nil {
		t.Fatal("redirect should fail")
	}
	if redirected {
		t.Fatal("request followed redirect with private data")
	}
}
