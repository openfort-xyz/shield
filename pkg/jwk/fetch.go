package jwk

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	keyfunc "github.com/MicahParks/keyfunc/v3"
)

// AllowedHostsEnv lists hosts (host or host:port, comma separated) whose JWK
// sets tenants may point at even though they are internal or plain HTTP, such
// as an auth service on the same private network.
const AllowedHostsEnv = "JWK_ALLOWED_HOSTS"

// ErrDisallowedJWKURL means a tenant-supplied JWK URL points somewhere Shield
// must not fetch from: not HTTPS, or a non-public address.
var ErrDisallowedJWKURL = errors.New("JWK URL not allowed")

const (
	fetchTimeout = 10 * time.Second
	maxRedirects = 3
	maxJWKSBytes = 1 << 20
	// ponytail: the cache resets when full; an LRU would keep hot entries if
	// one Shield ever serves more distinct JWK URLs than this.
	maxCachedSets = 256
)

// Addresses that are never public even though netip calls them global unicast.
var nonPublicPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("64:ff9b::/96"),
	netip.MustParsePrefix("64:ff9b:1::/48"),
	netip.MustParsePrefix("2002::/16"),
}

func isPublic(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsGlobalUnicast() || ip.IsPrivate() {
		return false
	}
	for _, p := range nonPublicPrefixes {
		if p.Contains(ip) {
			return false
		}
	}
	return true
}

// hostAllowed reports whether addr (host or host:port) is on the operator's
// allowlist, matching either the bare host or host:port.
func hostAllowed(addr string) bool {
	host := addr
	if h, _, err := net.SplitHostPort(addr); err == nil {
		host = h
	}
	for _, entry := range strings.Split(os.Getenv(AllowedHostsEnv), ",") {
		entry = strings.TrimSpace(entry)
		if entry != "" && (strings.EqualFold(entry, addr) || strings.EqualFold(entry, host)) {
			return true
		}
	}
	return false
}

// CheckTenantURL validates a tenant-supplied JWK URL before it is stored:
// HTTPS only, no credentials, and no literal non-public address, unless the
// host is allowlisted. Hostnames are resolved and checked again at fetch time.
func CheckTenantURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.Hostname() == "" {
		return fmt.Errorf("%w: not an absolute URL", ErrDisallowedJWKURL)
	}
	if u.User != nil {
		return fmt.Errorf("%w: credentials in URL", ErrDisallowedJWKURL)
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return fmt.Errorf("%w: scheme %q", ErrDisallowedJWKURL, u.Scheme)
	}
	if hostAllowed(u.Host) {
		return nil
	}
	if u.Scheme != "https" {
		return fmt.Errorf("%w: https required", ErrDisallowedJWKURL)
	}
	if ip, err := netip.ParseAddr(u.Hostname()); err == nil && !isPublic(ip) {
		return fmt.Errorf("%w: non-public address", ErrDisallowedJWKURL)
	}
	return nil
}

// dialPublic resolves addr itself and dials only a public address it checked,
// so DNS answers (including rebinding between check and use) cannot steer the
// request to an internal service.
func dialPublic(ctx context.Context, network, addr string) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: fetchTimeout}
	if hostAllowed(addr) {
		return dialer.DialContext(ctx, network, addr)
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return nil, err
	}
	for _, ip := range ips {
		if !isPublic(ip) {
			return nil, fmt.Errorf("%w: %s resolves to a non-public address", ErrDisallowedJWKURL, host)
		}
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("%w: %s does not resolve", ErrDisallowedJWKURL, host)
	}
	return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
}

type limitedBody struct {
	io.Reader
	io.Closer
}

// sizeLimit caps JWK set responses so a tenant URL cannot stream unbounded data.
type sizeLimit struct{ next http.RoundTripper }

func (s sizeLimit) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := s.next.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	resp.Body = limitedBody{Reader: io.LimitReader(resp.Body, maxJWKSBytes), Closer: resp.Body}
	return resp, nil
}

func tenantHTTPClient() *http.Client {
	transport := &http.Transport{
		// No proxy from the environment: a proxy would dial for us and skip
		// the address check.
		Proxy:                 nil,
		DialContext:           dialPublic,
		TLSHandshakeTimeout:   fetchTimeout,
		ResponseHeaderTimeout: fetchTimeout,
		ForceAttemptHTTP2:     true,
	}
	return &http.Client{
		Transport: sizeLimit{next: transport},
		Timeout:   fetchTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return fmt.Errorf("%w: too many redirects", ErrDisallowedJWKURL)
			}
			return CheckTenantURL(req.URL.String())
		},
	}
}

// keySets caches one keyfunc per JWK URL set. Building a keyfunc fetches the
// set and starts a refresher; doing it per request refetched on every call
// and leaked a goroutine each time. keyfunc does not fail on a failed first
// fetch: the set stays empty until its hourly refresh or a refresh on an
// unknown key ID (at most every 5 minutes).
type keySets struct {
	client *http.Client // nil: keyfunc's default client
	mu     sync.Mutex
	sets   map[string]*keySet
}

// keySet is built once; callers for the same URLs wait on ready, while
// callers for other URLs never wait on it (a slow tenant endpoint must not
// block everyone else's authentication).
type keySet struct {
	ready  chan struct{}
	kf     keyfunc.Keyfunc
	err    error
	cancel context.CancelFunc
}

func (k *keySets) get(urls []string) (keyfunc.Keyfunc, error) {
	key := strings.Join(urls, "\n")
	k.mu.Lock()
	if set, ok := k.sets[key]; ok {
		k.mu.Unlock()
		<-set.ready
		return set.kf, set.err
	}
	if len(k.sets) >= maxCachedSets {
		for _, set := range k.sets {
			set.cancel()
		}
		k.sets = nil
	}
	if k.sets == nil {
		k.sets = make(map[string]*keySet)
	}
	ctx, cancel := context.WithCancel(context.Background())
	set := &keySet{ready: make(chan struct{}), cancel: cancel}
	k.sets[key] = set
	k.mu.Unlock()

	var override keyfunc.Override
	if k.client != nil {
		override.Client = k.client
		override.HTTPTimeout = fetchTimeout
	}
	set.kf, set.err = keyfunc.NewDefaultOverrideCtx(ctx, urls, override)
	if set.err != nil {
		cancel()
		k.mu.Lock()
		if k.sets[key] == set {
			delete(k.sets, key)
		}
		k.mu.Unlock()
	}
	close(set.ready)
	return set.kf, set.err
}

var (
	trustedSets = &keySets{}
	tenantSets  = &keySets{client: tenantHTTPClient()}
)
