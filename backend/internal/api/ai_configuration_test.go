package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"

	"themis/internal/config"
	"themis/internal/memory"
)

func aiRequest(s *Server, handler http.HandlerFunc, method, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	handler(w, httptest.NewRequest(method, "/api/admin/config", strings.NewReader(body)))
	return w
}
func TestAIConfigAllProviderSecretsAndFailurePersistence(t *testing.T) {
	store := &aiConfigStore{}
	db := sql.OpenDB(store)
	defer db.Close()
	s := &Server{DB: db, Cfg: config.Config{OllamaURL: "http://ollama:11434", OllamaModel: "unit", OpenAIModel: "unit", AnthropicModel: "unit", MistralModel: "unit", FreeLLMURL: "https://models.example/v1", FreeLLMModel: "unit"}}
	var logs strings.Builder
	previous := log.Writer()
	log.SetOutput(&logs)
	defer log.SetOutput(previous)
	w := aiRequest(s, s.adminPutConfig, "PUT", `{"provider":"none","openai_key":"test-secret-openai","anthropic_key":"test-secret-claude","mistral_key":"test-secret-mistral","freellm_key":"test-secret-free","context":"Note déclarative","auto_categorize":true}`)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	for _, provider := range []string{"openai", "anthropic", "mistral", "freellm", "ollama", "none"} {
		w = aiRequest(s, s.adminPutConfig, "PUT", `{"provider":"`+provider+`"}`)
		if w.Code != 200 {
			t.Fatal(provider, w.Code, w.Body)
		}
		if strings.Contains(w.Body.String(), "test-secret") {
			t.Fatal("secret in response")
		}
		if s.getAIConfig().Context != "Note déclarative" || !s.getAIConfig().AutoCategorize {
			t.Fatal("provider switch erased behavior")
		}
	}
	for field, value := range s.aiKeyValues(s.getAIConfig()) {
		if !strings.HasPrefix(value, "test-secret") {
			t.Fatal("blank secret lost", field)
		}
	}
	reloaded := &Server{DB: db, Cfg: s.Cfg}
	reloaded.InitAI()
	if reloaded.getAIConfig() != s.getAIConfig() {
		t.Fatal("restart changed config")
	}
	before := s.getAIConfig()
	store.writeErr = errors.New("DB failure test-secret")
	w = aiRequest(s, s.adminPutConfig, "PUT", `{"provider":"openai","context":"NEW"}`)
	if w.Code != 500 || s.getAIConfig() != before || strings.Contains(w.Body.String(), "test-secret") {
		t.Fatal("failed save changed state or leaked", w.Code)
	}
	store.writeErr = nil
	w = aiRequest(s, s.adminPutConfig, "PUT", `{"provider":"none","clear_keys":["mistral_key"]}`)
	if w.Code != 200 || s.getAIConfig().MistralKey != "" {
		t.Fatal("explicit key removal failed")
	}
	w = aiRequest(s, s.adminPutConfig, "PUT", `{"provider":"mistral"}`)
	if w.Code != 400 {
		t.Fatal("missing key allowed")
	}
	s.Cfg.MistralKey = "env-secret"
	w = aiRequest(s, s.adminPutConfig, "PUT", `{"provider":"mistral"}`)
	if w.Code != 200 {
		t.Fatal("environment fallback failed")
	}
	revision := s.maskedConfig()["revision"].(string)
	w = aiRequest(s, s.adminPutConfig, "PUT", `{"provider":"none","expected_revision":"stale"}`)
	if w.Code != 409 || s.maskedConfig()["revision"] != revision {
		t.Fatal("concurrent edit not rejected")
	}
	if strings.Contains(logs.String(), "secret") || strings.Contains(logs.String(), "Note déclarative") {
		t.Fatal("secret/context in logs")
	}
}
func TestAIParameterValidationAndInvalidJSON(t *testing.T) {
	s := &Server{}
	for _, body := range []string{`{"provider":"unknown"}`, `{"provider":"openai","openai_key":"secret","openai_model":"bad model"}`, `{"provider":"openai","openai_model":"unit"}`, `{"provider":"ollama","ollama_url":"http://169.254.169.254","ollama_model":"unit"}`, `{"provider":"none","context":"` + strings.Repeat("é", 1001) + `"}`, `{"provider":null}`, `{"auto_categorize":"yes"}`, `{"private":"value"}`, `{} {}`, `null`, `[]`, `invalid`} {
		w := aiRequest(s, s.adminPutConfig, "PUT", body)
		if w.Code != 400 {
			t.Fatalf("invalid input status %d", w.Code)
		}
	}
	s.ApplyAIConfig(AIConfig{Provider: "openai", OpenAIKey: "stored-key", OpenAIModel: "unit"})
	before := s.getAIConfig()
	for _, body := range []string{`invalid`, `[]`, `null`, `{"provider":"unknown"}`} {
		w := aiRequest(s, s.adminTestAI, "POST", body)
		if w.Code != 400 || s.getAIConfig() != before {
			t.Fatal("invalid test fell back to active provider")
		}
	}
	text := sanitizeUserContext(strings.Repeat("é", 1100))
	if !utf8.ValidString(text) || len([]rune(text)) != 1000 {
		t.Fatal("context UTF-8 truncated incorrectly")
	}
}
func TestAIAdminGuardAndReservedSettings(t *testing.T) {
	s := &Server{}
	for _, user := range []*authUser{nil, {ID: 1, Role: "user"}, {ID: 1, Role: "admin"}, {ID: 0, Role: "admin"}} {
		r := httptest.NewRequest("GET", "/api/admin/config", nil)
		if user != nil {
			r = r.WithContext(context.WithValue(r.Context(), ctxUserKey, user))
		}
		w := httptest.NewRecorder()
		s.aiAdminOnly(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })(w, r)
		want := 401
		if user != nil && user.ID > 0 {
			want = 403
			if user.Role == "admin" {
				want = 204
			}
		}
		if w.Code != want {
			t.Fatal("admin guard", w.Code, want)
		}
	}
	for _, handler := range []http.HandlerFunc{s.getSetting, s.putSetting} {
		r := httptest.NewRequest("GET", "/api/settings/ai_config", strings.NewReader(`{}`))
		r.SetPathValue("key", "ai_config")
		w := httptest.NewRecorder()
		handler(w, r)
		if w.Code != 403 {
			t.Fatal("generic settings exposes administrative configuration")
		}
	}
}
func TestAIGlobalConfigurationAndRouterPostgres(t *testing.T) {
	db := qualityTestDB(t)
	_, err := db.Exec(`CREATE TEMP TABLE settings(key text,value jsonb,owner_id bigint,updated_at timestamptz DEFAULT now());CREATE UNIQUE INDEX ai_global_key ON settings(COALESCE(owner_id,0),key);CREATE TEMP TABLE users(id bigint);
 INSERT INTO settings VALUES('ai_config','{"provider":"ollama","ollama_url":"http://169.254.169.254","ollama_model":"private"}',1,now()),('ai_config','{"provider":"none","context":"GLOBAL"}',NULL,now())`)
	if err != nil {
		t.Fatal(err)
	}
	s := &Server{DB: db}
	s.InitAI()
	if s.getAIConfig().Provider != "none" || s.getAIConfig().Context != "GLOBAL" {
		t.Fatal("user configuration shadowed global")
	}
	w := aiRequest(s, s.adminPutConfig, "PUT", `{"provider":"openai","openai_key":"global-secret","openai_model":"unit"}`)
	if w.Code != 200 {
		t.Fatal("PostgreSQL save", w.Code, w.Body)
	}
	reloaded := &Server{DB: db}
	reloaded.InitAI()
	if reloaded.getAIConfig().OpenAIKey != "global-secret" || reloaded.getAIConfig().Context != "GLOBAL" {
		t.Fatal("PostgreSQL persistence failed")
	}
	var owned string
	if err = db.QueryRow(`SELECT value->>'provider' FROM settings WHERE owner_id=1`).Scan(&owned); err != nil || owned != "ollama" {
		t.Fatal("global save changed user settings")
	}
	router := s.Router()
	for _, route := range []string{"GET /api/admin/config", "PUT /api/admin/config", "POST /api/admin/ai/test", "GET /api/admin/ollama/models", "POST /api/admin/ai/memory/test"} {
		p := strings.SplitN(route, " ", 2)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(p[0], p[1], strings.NewReader(`{}`)))
		if w.Code != 401 {
			t.Fatal("bootstrap bypassed admin guard", route, w.Code)
		}
	}
}
func TestAIDiagnosticDraftPromotionAndSanitizedFailure(t *testing.T) {
	var status atomic.Int64
	status.Store(200)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(int(status.Load()))
		w.Write([]byte(`{"models":[{"name":"unit"}],"error":"SECRET PROMPT TOKEN"}`))
	}))
	defer server.Close()
	store := &aiConfigStore{}
	db := sql.OpenDB(store)
	defer db.Close()
	s := &Server{DB: db, Cfg: config.Config{OllamaURL: server.URL, OllamaModel: "unit"}}
	s.ApplyAIConfig(AIConfig{Provider: "none", OllamaURL: server.URL, OllamaModel: "unit"})
	body := `{"provider":"ollama","ollama_model":"unit","ollama_url":"` + server.URL + `"}`
	before := s.getAIConfig()
	w := aiRequest(s, s.adminTestAI, "POST", body)
	var result aiDiagnostic
	json.Unmarshal(w.Body.Bytes(), &result)
	if w.Code != 200 || !result.OK || result.Scope != "draft" || result.CheckedAt == nil || result.LatencyMS == nil || s.getAIConfig() != before || len(store.raw) != 0 {
		t.Fatal("test mutated saved config or lacks real metadata", w.Code, w.Body)
	}
	w = aiRequest(s, s.adminPutConfig, "PUT", body)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	raw, _ := json.Marshal(s.maskedConfig())
	if !strings.Contains(string(raw), `"status":"available"`) || strings.Contains(string(raw), `"scope":"draft"`) {
		t.Fatal("successful draft test not attached to saved connection")
	}
	status.Store(500)
	before = s.getAIConfig()
	w = aiRequest(s, s.adminTestAI, "POST", `{}`)
	json.Unmarshal(w.Body.Bytes(), &result)
	if result.OK || result.Status != "unavailable" || s.getAIConfig() != before || strings.Contains(w.Body.String(), "SECRET") {
		t.Fatal("provider failure falsely reported healthy or leaked", w.Body)
	}
	s.ApplyAIConfig(AIConfig{Provider: "none"})
	if s.maskedConfig()["diagnostic"].(*aiDiagnostic).Status != "not_configured" {
		t.Fatal("stale result retained for another provider")
	}
}
func TestAIDiagnosticTimeoutAndMemoryStates(t *testing.T) {
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer slow.Close()
	s := &Server{Cfg: config.Config{OllamaURL: slow.URL, OllamaModel: "unit"}}
	s.ApplyAIConfig(AIConfig{Provider: "ollama", OllamaURL: slow.URL, OllamaModel: "unit"})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	r := httptest.NewRequest("POST", "/api/admin/ai/test", strings.NewReader(`{}`)).WithContext(ctx)
	w := httptest.NewRecorder()
	s.adminTestAI(w, r)
	if !strings.Contains(w.Body.String(), `"code":"timeout"`) {
		t.Fatal("timeout not explicit", w.Body)
	}
	healthy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"status":"ok"}`)) }))
	defer healthy.Close()
	s.Memory = memory.New(memory.Config{Enabled: false, URL: healthy.URL, RequestTimeout: time.Second})
	w = aiRequest(s, s.adminTestMemory, "POST", `{}`)
	if !strings.Contains(w.Body.String(), `"status":"disabled"`) {
		t.Fatal("disabled memory queried or hidden")
	}
	s.Memory = memory.New(memory.Config{Enabled: true, URL: healthy.URL, RequestTimeout: time.Second})
	w = aiRequest(s, s.adminTestMemory, "POST", `{}`)
	if !strings.Contains(w.Body.String(), `"status":"available"`) {
		t.Fatal("memory health failed", w.Body)
	}
	s.Memory = memory.New(memory.Config{Enabled: true, URL: "", RequestTimeout: time.Second})
	w = aiRequest(s, s.adminTestMemory, "POST", `{}`)
	if !strings.Contains(w.Body.String(), `"status":"not_configured"`) {
		t.Fatal("unconfigured memory reported available")
	}
}

func TestAIOutOfOrderTestsCannotOverwriteNewerResult(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		w.Write([]byte(`{"models":[{"name":"unit"}]}`))
	}))
	defer slow.Close()
	fast := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"models":[{"name":"unit"}]}`)) }))
	defer fast.Close()
	store := &aiConfigStore{}
	db := sql.OpenDB(store)
	defer db.Close()
	s := &Server{DB: db, Cfg: config.Config{OllamaURL: slow.URL, OllamaModel: "unit", AITrustedEndpoints: []string{fast.URL}}}
	s.ApplyAIConfig(AIConfig{Provider: "none", OllamaModel: "unit"})
	body := func(endpoint string) string {
		return `{"provider":"ollama","ollama_model":"unit","ollama_url":"` + endpoint + `"}`
	}
	complete := make(chan *httptest.ResponseRecorder, 1)
	go func() { complete <- aiRequest(s, s.adminTestAI, "POST", body(slow.URL)) }()
	<-started
	w := aiRequest(s, s.adminTestAI, "POST", body(fast.URL))
	if w.Code != 200 {
		close(release)
		<-complete
		t.Fatal(w.Code)
	}
	close(release)
	old := <-complete
	if old.Code != 200 {
		t.Fatal(old.Code)
	}
	w = aiRequest(s, s.adminPutConfig, "PUT", body(fast.URL))
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	result := s.maskedConfig()["diagnostic"].(*aiDiagnostic)
	if !result.OK || result.Scope != "saved" {
		t.Fatal("stale test overwrote newest draft result")
	}
}
func TestAIModelDiscoveryUsesControlledEndpointAndStatus(t *testing.T) {
	var status atomic.Int64
	status.Store(200)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(int(status.Load()))
		w.Write([]byte(`{"models":[{"name":"unit"},{"name":"unit"},{"name":"bad name"}]}`))
	}))
	defer server.Close()
	s := &Server{Cfg: config.Config{OllamaURL: server.URL}}
	r := httptest.NewRequest("GET", "/api/admin/ollama/models", nil)
	w := httptest.NewRecorder()
	s.adminOllamaModels(w, r)
	if w.Code != 200 || w.Body.String() != "{\"models\":[\"unit\"]}\n" {
		t.Fatal("models invalid/duplicate names not filtered", w.Code, w.Body)
	}
	status.Store(500)
	w = httptest.NewRecorder()
	s.adminOllamaModels(w, r)
	if w.Code != 502 {
		t.Fatal("non-2xx discovery reported available")
	}
	r = httptest.NewRequest("GET", "/api/admin/ollama/models?url=http%3A%2F%2F169.254.169.254", nil)
	w = httptest.NewRecorder()
	s.adminOllamaModels(w, r)
	if w.Code != 400 {
		t.Fatal("unsafe discovery URL accepted")
	}
}
