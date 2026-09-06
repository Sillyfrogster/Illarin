// Package outbound holds the rule for where Illarin's own requests may go. A
// webhook address is supplied by a person, so every request it produces is a
// request an outsider chose the target of. Nothing here follows a redirect, and
// nothing dials an address it has not just checked.
package outbound

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// The scheme and port an endpoint is required to use.
const (
	Scheme = "https"
	Port   = "443"
)

// MaxAddressBytes is the longest endpoint address Illarin will hold.
const MaxAddressBytes = 300

// ErrNotPublic says a host resolved somewhere Illarin refuses to connect.
var ErrNotPublic = errors.New("the host does not resolve into public address space")

// Address is one endpoint that has passed the policy.
type Address struct {
	URL  *url.URL
	Host string
}

// String answers the address as it will be sent to.
func (a Address) String() string { return a.URL.String() }

// Check answers the address behind a supplied value, or why it was refused.
func Check(raw string) (Address, error) {
	if len(raw) > MaxAddressBytes {
		return Address{}, fmt.Errorf("an endpoint address is at most %d characters", MaxAddressBytes)
	}
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return Address{}, fmt.Errorf("that is not an address Illarin can read")
	}
	if parsed.Scheme != Scheme {
		return Address{}, fmt.Errorf("an endpoint address starts with %s://", Scheme)
	}
	if parsed.User != nil {
		return Address{}, errors.New("an endpoint address carries no username or password")
	}
	if parsed.Fragment != "" || strings.Contains(raw, "#") {
		return Address{}, errors.New("an endpoint address carries no fragment")
	}
	if port := parsed.Port(); port != "" && port != Port {
		return Address{}, fmt.Errorf("an endpoint address answers on port %s", Port)
	}
	host := strings.TrimSuffix(parsed.Hostname(), ".")
	if host == "" {
		return Address{}, errors.New("an endpoint address names a host")
	}
	if at, err := netip.ParseAddr(host); err == nil {
		if !Public(at) {
			return Address{}, ErrNotPublic
		}
	} else if !strings.Contains(host, ".") {
		return Address{}, errors.New("an endpoint address names a host on the public internet")
	}
	return Address{URL: parsed, Host: host}, nil
}

// Public answers whether one resolved address is somewhere Illarin will connect.
func Public(at netip.Addr) bool {
	if !at.IsValid() || at.Zone() != "" {
		return false
	}
	at = at.Unmap()
	refused := refusedSix
	if at.Is4() {
		refused = refusedFour
	}
	for _, prefix := range refused {
		if prefix.Contains(at) {
			return false
		}
	}
	return true
}

var refusedFour = prefixes(
	"0.0.0.0/8", "10.0.0.0/8", "100.64.0.0/10", "127.0.0.0/8", "169.254.0.0/16",
	"172.16.0.0/12", "192.0.0.0/24", "192.0.2.0/24", "192.88.99.0/24",
	"192.168.0.0/16", "198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24",
	"224.0.0.0/4", "240.0.0.0/4",
)

var refusedSix = prefixes(
	"::/128", "::1/128", "64:ff9b::/96", "64:ff9b:1::/48", "100::/64",
	"2001::/23", "2001:db8::/32", "2002::/16", "fc00::/7", "fe80::/10", "ff00::/8",
)

func prefixes(raw ...string) []netip.Prefix {
	held := make([]netip.Prefix, 0, len(raw))
	for _, one := range raw {
		held = append(held, netip.MustParsePrefix(one))
	}
	return held
}

// Limits are what one outbound request may spend and read back.
type Limits struct {
	Connect   time.Duration
	Request   time.Duration
	ReadBytes int64
}

// DefaultLimits are what Illarin sends publication events under.
func DefaultLimits() Limits {
	return Limits{Connect: 5 * time.Second, Request: 10 * time.Second, ReadBytes: 8 << 10}
}

// MaxRetryAfter is the longest wait Illarin will take from an endpoint, so a
// receiver asking for a month cannot park work indefinitely.
const MaxRetryAfter = 24 * time.Hour

// Answer is the safe part of what an endpoint said back. The body is bounded
// and is compared, never logged.
type Answer struct {
	Status     int
	Body       []byte
	Took       time.Duration
	RetryAfter time.Duration
}

// Reaching turns a host into the one address a request is dialed at.
type Reaching struct {
	Resolve func(ctx context.Context, host string) ([]netip.Addr, error)
}

// At answers the address to dial, having refused every non-public result.
func (r Reaching) At(ctx context.Context, host, port string) (string, error) {
	resolve := r.Resolve
	if resolve == nil {
		resolve = lookup
	}
	found, err := resolve(ctx, host)
	if err != nil {
		return "", fmt.Errorf("look up %s: %w", host, err)
	}
	for _, at := range found {
		if Public(at) {
			return net.JoinHostPort(at.String(), port), nil
		}
	}
	return "", ErrNotPublic
}

func lookup(ctx context.Context, host string) ([]netip.Addr, error) {
	return net.DefaultResolver.LookupNetIP(ctx, "ip", host)
}

// Caller sends one bounded request to a checked address. It never follows a
// redirect and never reuses a connection, so every attempt resolves again.
type Caller struct {
	Limits Limits
	Reach  func(ctx context.Context, host, port string) (string, error)

	transport *http.Transport
	client    *http.Client
}

// NewCaller builds the caller Illarin sends publication events with.
func NewCaller(limits Limits) *Caller {
	caller := &Caller{Limits: limits, Reach: Reaching{}.At}
	caller.transport = &http.Transport{
		DisableKeepAlives:   true,
		TLSHandshakeTimeout: limits.Connect,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			at, err := caller.Reach(ctx, host, port)
			if err != nil {
				return nil, err
			}
			return (&net.Dialer{Timeout: limits.Connect}).DialContext(ctx, network, at)
		},
	}
	caller.client = &http.Client{
		Transport: caller.transport,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return caller
}

// Check answers the host behind a supplied address, refusing anything the
// policy does not allow.
func (c *Caller) Check(address string) (string, error) {
	checked, err := Check(address)
	if err != nil {
		return "", err
	}
	return checked.Host, nil
}

// Post sends one JSON body and answers what came back, bounded.
func (c *Caller) Post(
	ctx context.Context,
	address string,
	headers map[string]string,
	body []byte,
) (Answer, error) {
	checked, err := Check(address)
	if err != nil {
		return Answer{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, c.Limits.Request)
	defer cancel()
	request, err := http.NewRequestWithContext(
		ctx, http.MethodPost, checked.String(), bytes.NewReader(body),
	)
	if err != nil {
		return Answer{}, fmt.Errorf("build the request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "Illarin-Publication/1")
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	started := time.Now()
	response, err := c.client.Do(request)
	if err != nil {
		return Answer{}, fmt.Errorf("reach %s: %w", checked.Host, err)
	}
	defer response.Body.Close()
	read, err := io.ReadAll(io.LimitReader(response.Body, c.Limits.ReadBytes))
	if err != nil {
		return Answer{}, fmt.Errorf("read what %s said back: %w", checked.Host, err)
	}
	return Answer{
		Status: response.StatusCode, Body: read, Took: time.Since(started),
		RetryAfter: RetryAfter(response.Header.Get("Retry-After"), started),
	}, nil
}

// RetryAfter reads how long an endpoint asked to be left alone, in either form
// the header takes, and holds the answer inside what Illarin will wait.
func RetryAfter(header string, from time.Time) time.Duration {
	header = strings.TrimSpace(header)
	if header == "" {
		return 0
	}
	wait := time.Duration(0)
	if seconds, err := strconv.Atoi(header); err == nil {
		wait = time.Duration(seconds) * time.Second
	} else if at, err := http.ParseTime(header); err == nil {
		wait = at.Sub(from)
	}
	if wait < 0 {
		return 0
	}
	return min(wait, MaxRetryAfter)
}

// trust lets a test point a caller at a receiver holding its own certificate.
func (c *Caller) trust(config *tls.Config) { c.transport.TLSClientConfig = config }
