package llm

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestEndpointPolicyURLAndAddressValidation(t *testing.T) {
	p := EndpointPolicy{}
	for _, raw := range []string{"file:///etc/passwd", "http://127.0.0.1:11434", "https://127.0.0.1", "https://[::1]", "https://[::ffff:127.0.0.1]", "https://169.254.169.254", "https://10.0.0.1", "https://user:secret@models.example", "https://models.example?key=secret", "https://models.example/#secret", "https://models.example:0", "https://models.example:99999", "https://models.example\\evil", "https://[fe80::1%25eth0]"} {
		if p.Validate(raw) == nil {
			t.Fatalf("unsafe endpoint accepted: %s", raw)
		}
	}
	for _, raw := range []string{"https://models.example", "https://models.example/v1", "https://93.184.216.34"} {
		if err := p.Validate(raw); err != nil {
			t.Fatal(raw, err)
		}
	}
	p.TrustedEndpoints = []string{"http://127.0.0.1:11434/ollama", "http://10.0.0.2:11434", "http://[::1]:11434", "http://169.254.169.254"}
	for _, raw := range p.TrustedEndpoints[:3] {
		if err := p.Validate(raw); err != nil {
			t.Fatal("trusted server refused", raw, err)
		}
	}
	for _, raw := range []string{"http://127.0.0.1:11434/other", "http://10.0.0.2:8080", "http://169.254.169.254"} {
		if p.Validate(raw) == nil {
			t.Fatal("allowlist escaped", raw)
		}
	}
	for _, raw := range []string{"0.0.0.0", "127.0.0.2", "::1", "::ffff:10.1.2.3", "169.254.169.254", "100.64.0.1", "198.18.1.1", "224.0.0.1", "255.255.255.255", "fd00::1", "fe80::1", "2001:db8::1", "64:ff9b::7f00:1"} {
		if permittedIP(net.ParseIP(raw), false) {
			t.Fatal("nonpublic address allowed", raw)
		}
	}
}
func TestEndpointPolicyDNSPinningAndMixedAnswers(t *testing.T) {
	lookups, dials := 0, 0
	p := EndpointPolicy{lookup: func(context.Context, string) ([]net.IPAddr, error) {
		lookups++
		return []net.IPAddr{{IP: net.ParseIP("93.184.216.34")}}, nil
	}, dial: func(_ context.Context, _, address string) (net.Conn, error) {
		dials++
		if address != "93.184.216.34:443" {
			t.Errorf("hostname resolved again: %s", address)
		}
		return nil, errors.New("mock unavailable")
	}}
	client, err := p.Client("https://models.example")
	if err != nil {
		t.Fatal(err)
	}
	transport := client.Transport.(*http.Transport)
	if transport.ResponseHeaderTimeout != 0 {
		t.Fatal("network guard shortened model generation timeout")
	}
	if transport.Proxy != nil {
		t.Fatal("ambient proxy bypasses network policy")
	}
	_, _ = transport.DialContext(context.Background(), "tcp", "models.example:443")
	if lookups != 1 || dials != 1 {
		t.Fatal("unverified dial", lookups, dials)
	}
	p.lookup = func(context.Context, string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP("93.184.216.34")}, {IP: net.ParseIP("127.0.0.1")}}, nil
	}
	client, _ = p.Client("https://models.example")
	_, err = client.Transport.(*http.Transport).DialContext(context.Background(), "tcp", "models.example:443")
	if err == nil || dials != 1 {
		t.Fatal("mixed public/private DNS response accepted")
	}
	_, err = transport.DialContext(context.Background(), "tcp", "other.example:443")
	if err == nil || dials != 1 {
		t.Fatal("client reused for another server")
	}
}
func TestEndpointPolicyBlocksRedirects(t *testing.T) {
	calls := 0
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(200) }))
	defer destination.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, destination.URL, 302) }))
	defer source.Close()
	p := EndpointPolicy{TrustedEndpoints: []string{source.URL}}
	client, err := p.Client(source.URL)
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Get(source.URL + "/api/tags")
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 302 || calls != 0 {
		t.Fatal("redirect followed", calls)
	}
}
func TestProviderErrorsNeverReturnRemoteSecrets(t *testing.T) {
	previous := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = previous })
	for _, status := range []int{401, 403, 429, 500, 200} {
		http.DefaultTransport = providerTestTransport(func(r *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"error":{"message":"private-token PRIVATE BANK DATA"}}`))}, nil
		})
		for _, client := range []Client{&Anthropic{APIKey: "private-token", Model: "test"}, &Mistral{APIKey: "private-token", Model: "test"}, &FreeLLM{APIKey: "private-token", Model: "test", BaseURL: "https://models.example"}} {
			_, err := client.Complete(context.Background(), "PRIVATE PROMPT", "PRIVATE BANK DATA", 4, false)
			if err == nil || strings.Contains(err.Error(), "private-token") || strings.Contains(err.Error(), "PRIVATE") {
				t.Fatal("unsafe provider error", status, err)
			}
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	<-ctx.Done()
	if !errors.Is(providerNetworkError(ctx, errors.New("PRIVATE URL key=secret")), context.DeadlineExceeded) {
		t.Fatal("timeout not retained")
	}
}

type providerTestTransport func(*http.Request) (*http.Response, error)

func (f providerTestTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
