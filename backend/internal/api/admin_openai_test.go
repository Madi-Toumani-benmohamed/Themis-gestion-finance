package api

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"themis/internal/config"
	"themis/internal/llm"
)

// Persistance en mémoire pour exercer save/reload sans service externe.
type aiConfigStore struct{ raw []byte }

func (s *aiConfigStore) Connect(context.Context) (driver.Conn, error) { return &aiConfigConn{s}, nil }
func (s *aiConfigStore) Driver() driver.Driver                        { return aiConfigDriver{s} }

type aiConfigDriver struct{ s *aiConfigStore }

func (d aiConfigDriver) Open(string) (driver.Conn, error) { return &aiConfigConn{d.s}, nil }

type aiConfigConn struct{ s *aiConfigStore }

func (c *aiConfigConn) Prepare(string) (driver.Stmt, error) { return nil, driver.ErrSkip }
func (c *aiConfigConn) Close() error                        { return nil }
func (c *aiConfigConn) Begin() (driver.Tx, error)           { return nil, driver.ErrSkip }
func (c *aiConfigConn) ExecContext(_ context.Context, _ string, args []driver.NamedValue) (driver.Result, error) {
	c.s.raw = append([]byte(nil), args[0].Value.([]byte)...)
	return driver.RowsAffected(1), nil
}
func (c *aiConfigConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	return &aiConfigRows{raw: c.s.raw}, nil
}

type aiConfigRows struct {
	raw  []byte
	done bool
}

func (*aiConfigRows) Columns() []string { return []string{"value"} }
func (*aiConfigRows) Close() error      { return nil }
func (r *aiConfigRows) Next(dst []driver.Value) error {
	if r.done || len(r.raw) == 0 {
		return io.EOF
	}
	r.done = true
	dst[0] = r.raw
	return nil
}

func TestAdminOpenAISaveReloadAndMask(t *testing.T) {
	store := &aiConfigStore{}
	db := sql.OpenDB(store)
	defer db.Close()
	srv := &Server{DB: db, Cfg: config.Config{OpenAIModel: llm.DefaultOpenAIModel}}
	save := func(body string) *httptest.ResponseRecorder {
		t.Helper()
		w := httptest.NewRecorder()
		srv.adminPutConfig(w, httptest.NewRequest("PUT", "/api/admin/config", strings.NewReader(body)))
		if w.Code != http.StatusOK {
			t.Fatalf("save: %d %s", w.Code, w.Body.String())
		}
		if strings.Contains(w.Body.String(), "test-secret") || strings.Contains(w.Body.String(), `"openai_key":`) {
			t.Fatal("secret exposed")
		}
		return w
	}
	save(`{"provider":"openai","openai_key":"test-secret","openai_model":"custom-model"}`)
	save(`{"provider":"openai","openai_model":"another-model"}`)
	if srv.getAIConfig().OpenAIKey != "test-secret" {
		t.Fatal("blank input erased saved key")
	}
	// Simule le redémarrage du serveur à partir du JSON persisté.
	reloaded := &Server{DB: db, Cfg: srv.Cfg}
	reloaded.InitAI()
	client, ok := reloaded.getLLM().(*llm.OpenAI)
	if !ok || client.APIKey != "test-secret" || client.Model != "another-model" {
		t.Fatal("saved config not restored")
	}
	masked := reloaded.maskedConfig()
	raw, _ := json.Marshal(masked)
	if masked["openai_key_set"] != true || strings.Contains(string(raw), "test-secret") {
		t.Fatal("masking incorrect")
	}
	save(`{"provider":"none"}`)
	save(`{"provider":"openai"}`)
	if srv.getAIConfig().OpenAIKey != "test-secret" {
		t.Fatal("provider switch erased key")
	}
}

func TestAdminOpenAIEnvironmentAndValidation(t *testing.T) {
	srv := &Server{Cfg: config.Config{OpenAIKey: "env-secret", OpenAIModel: "env-model"}}
	c, ok := srv.buildClient(AIConfig{Provider: "openai"}).(*llm.OpenAI)
	if !ok || c.APIKey != "env-secret" || c.Model != "env-model" {
		t.Fatal("environment fallback failed")
	}
	srv.ai.config = AIConfig{OpenAIKey: "saved-secret"}
	c = srv.buildClient(srv.withOpenAIDefaults(AIConfig{Provider: "openai"})).(*llm.OpenAI)
	if c.APIKey != "saved-secret" {
		t.Fatal("test configuration should use saved key before environment")
	}
	empty := &Server{}
	w := httptest.NewRecorder()
	empty.adminPutConfig(w, httptest.NewRequest("PUT", "/api/admin/config", strings.NewReader(`{"provider":"openai"}`)))
	if w.Code != http.StatusBadRequest {
		t.Fatal("missing key should fail before database access")
	}
	if empty.buildClient(AIConfig{Provider: "openai"}) != nil {
		t.Fatal("missing key must disable client")
	}
}

type openAIAdminTransport func(*http.Request) (*http.Response, error)

func (f openAIAdminTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestAdminOpenAITestUsesSavedKeyAndSyntheticPrompt(t *testing.T) {
	previous := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = previous })
	calls := 0
	http.DefaultTransport = openAIAdminTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.String() != "https://api.openai.com/v1/responses" || r.Header.Get("Authorization") != "Bearer saved-secret" {
			t.Error("saved key or OpenAI endpoint incorrect")
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			return nil, err
		}
		if payload["input"] != "ping" || payload["store"] != false || payload["model"] != "chosen-model" || payload["max_output_tokens"] != float64(256) {
			t.Errorf("unexpected test payload: %v", payload)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"OK"}]}]}`)), Header: make(http.Header)}, nil
	})
	srv := &Server{}
	srv.ai.config = AIConfig{Provider: "openai", OpenAIKey: "saved-secret"}
	w := httptest.NewRecorder()
	srv.adminTestAI(w, httptest.NewRequest("POST", "/api/admin/ai/test", strings.NewReader(`{"provider":"openai","openai_model":"chosen-model"}`)))
	if calls != 1 || !strings.Contains(w.Body.String(), `"ok":true`) || strings.Contains(w.Body.String(), "saved-secret") {
		t.Fatalf("test failed: %s", w.Body.String())
	}
}
