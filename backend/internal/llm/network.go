package llm

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// EndpointPolicy controls configurable model servers. Trusted endpoints are
// complete base URLs from operator configuration, never from a browser request.
// DNS is checked at dial time; dialing a literal IP prevents DNS rebinding.
type EndpointPolicy struct {
	TrustedEndpoints []string
	lookup           func(context.Context, string) ([]net.IPAddr, error)
	dial             func(context.Context, string, string) (net.Conn, error)
}

func normalizedEndpoint(raw string) (*url.URL, error) {
	if len(raw) > 2048 || strings.TrimSpace(raw) != raw {
		return nil, fmt.Errorf("URL du serveur invalide")
	}
	u, err := url.Parse(raw)
	if err != nil || u.Opaque != "" || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		return nil, fmt.Errorf("URL HTTP(S) sans identifiants, paramètres ni fragment requise")
	}
	if strings.ContainsAny(u.Host, "\\%") || strings.ContainsAny(u.Path, "\\\x00\r\n") {
		return nil, fmt.Errorf("URL du serveur invalide")
	}
	if u.Port() != "" {
		n, err := strconv.Atoi(u.Port())
		if err != nil || n < 1 || n > 65535 {
			return nil, fmt.Errorf("port du serveur invalide")
		}
	}

	u.Host = strings.ToLower(u.Host)
	u.Path = strings.TrimRight(u.Path, "/")
	u.RawPath = ""
	return u, nil
}
func (p EndpointPolicy) trusted(u *url.URL) bool {
	for _, raw := range p.TrustedEndpoints {
		v, err := normalizedEndpoint(strings.TrimSpace(raw))
		if err == nil && v.String() == u.String() {
			return true
		}
	}
	return false
}

var blockedIPRanges = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"), netip.MustParsePrefix("100.64.0.0/10"), netip.MustParsePrefix("192.0.0.0/24"), netip.MustParsePrefix("192.0.2.0/24"), netip.MustParsePrefix("198.18.0.0/15"), netip.MustParsePrefix("198.51.100.0/24"), netip.MustParsePrefix("203.0.113.0/24"), netip.MustParsePrefix("240.0.0.0/4"), netip.MustParsePrefix("2001::/23"), netip.MustParsePrefix("2001:db8::/32"), netip.MustParsePrefix("2002::/16"),
}

func permittedIP(ip net.IP, trusted bool) bool {
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return false
	}
	addr = addr.Unmap()
	if addr.IsLoopback() {
		return trusted
	}
	// Metadata/link-local, unspecified and multicast destinations stay forbidden,
	// including on allowlisted endpoints.
	if !addr.IsGlobalUnicast() || addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() {
		return trusted && addr.IsLoopback()
	}
	if netip.MustParsePrefix("100.64.0.0/10").Contains(addr) {
		return trusted
	}
	if addr.IsPrivate() {
		return trusted
	}
	for _, prefix := range blockedIPRanges {
		if prefix.Contains(addr) {
			return false
		}
	}
	// Non-global IPv6 translation/site-local ranges cannot tunnel a private IPv4.
	if addr.Is6() && !netip.MustParsePrefix("2000::/3").Contains(addr) {
		return false
	}
	return true
}
func (p EndpointPolicy) Validate(raw string) error {
	u, err := normalizedEndpoint(raw)
	if err != nil {
		return err
	}
	trusted := p.trusted(u)
	if u.Scheme == "http" && !trusted {
		return fmt.Errorf("HTTPS requis, sauf serveur explicitement autorisé par l’administrateur système")
	}
	if ip := net.ParseIP(u.Hostname()); ip != nil && !permittedIP(ip, trusted) {
		return fmt.Errorf("destination réseau non autorisée ; vérifier AI_TRUSTED_ENDPOINTS côté serveur")
	}
	return nil
}
func (p EndpointPolicy) Client(raw string) (*http.Client, error) {
	if err := p.Validate(raw); err != nil {
		return nil, err
	}
	base, _ := normalizedEndpoint(raw)
	trusted := p.trusted(base)
	lookup := p.lookup
	if lookup == nil {
		lookup = net.DefaultResolver.LookupIPAddr
	}
	dial := p.dial
	if dial == nil {
		dial = (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext
	}
	transport := (&http.Transport{MaxIdleConns: 20, IdleConnTimeout: 60 * time.Second}).Clone()
	transport.Proxy = nil
	transport.TLSHandshakeTimeout = 5 * time.Second
	// Generation can legitimately exceed 30s. Operation contexts/client timeouts bound it.
	transport.ResponseHeaderTimeout = 0
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, fmt.Errorf("destination réseau invalide")
		}
		// A configured client cannot be reused for another origin.
		expectedPort := base.Port()
		if expectedPort == "" {
			expectedPort = "443"
			if base.Scheme == "http" {
				expectedPort = "80"
			}
		}
		if !strings.EqualFold(host, base.Hostname()) || port != expectedPort {
			return nil, fmt.Errorf("destination réseau différente du serveur configuré")
		}
		ips, err := lookup(ctx, host)
		if err != nil || len(ips) == 0 {
			return nil, providerNetworkError(ctx, err)
		}
		for _, ip := range ips {
			if !permittedIP(ip.IP, trusted) {
				return nil, fmt.Errorf("destination réseau non autorisée")
			}
		}
		var connection net.Conn
		for _, ip := range ips {
			connection, err = dial(ctx, network, net.JoinHostPort(ip.IP.String(), port))
			if err == nil {
				return connection, nil
			}
		}
		return nil, providerNetworkError(ctx, err)
	}
	return &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, nil
}
func providerHTTPClient(configured *http.Client, timeout time.Duration) *http.Client {
	if configured == nil {
		return &http.Client{Timeout: timeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	copy := *configured
	copy.Timeout = timeout
	copy.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &copy
}
