package dispatch

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	Scheme = "https"
	Port   = "443"
)

const MaxAddressBytes = 300

var ErrNotPublic = errors.New("the host does not resolve into public address space")

type Address struct {
	URL  *url.URL
	Host string
}

func (a Address) String() string { return a.URL.String() }

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

type Limits struct {
	Connect   time.Duration
	Request   time.Duration
	ReadBytes int64
}

func DefaultLimits() Limits {
	return Limits{Connect: 5 * time.Second, Request: 10 * time.Second, ReadBytes: 8 << 10}
}

const MaxRetryAfter = 24 * time.Hour

type Answer struct {
	Status     int
	Body       []byte
	Took       time.Duration
	RetryAfter time.Duration
}

type Reaching struct {
	Resolve func(ctx context.Context, host string) ([]netip.Addr, error)
}

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
		if !Public(at) {
			return "", ErrNotPublic
		}
	}
	if len(found) == 0 {
		return "", ErrNotPublic
	}
	return net.JoinHostPort(found[0].String(), port), nil
}

func lookup(ctx context.Context, host string) ([]netip.Addr, error) {
	return net.DefaultResolver.LookupNetIP(ctx, "ip", host)
}

type Caller struct {
	Limits Limits
	Reach  func(ctx context.Context, host, port string) (string, error)

	transport *http.Transport
	client    *http.Client
	rates     sync.Mutex
	buckets   map[string]string
	nextSend  map[string]time.Time
}

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

func (c *Caller) Check(address string) (string, error) {
	checked, err := Check(address)
	if err != nil {
		return "", err
	}
	return checked.Host, nil
}

func (c *Caller) Post(
	ctx context.Context,
	address string,
	headers map[string]string,
	body []byte,
) (Answer, error) {
	return c.send(ctx, http.MethodPost, address, headers, body)
}

func (c *Caller) Get(ctx context.Context, address string) (Answer, error) {
	return c.send(ctx, http.MethodGet, address, nil, nil)
}

func (c *Caller) Request(ctx context.Context, method, address string, body []byte) (Answer, error) {
	return c.send(ctx, method, address, nil, body)
}

func (c *Caller) send(
	ctx context.Context,
	method, address string,
	headers map[string]string,
	body []byte,
) (Answer, error) {
	checked, err := Check(address)
	if err != nil {
		return Answer{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, c.Limits.Request)
	defer cancel()
	route := ""
	if checked.Host == "discord.com" {
		if _, rest, found := strings.Cut(checked.URL.Path, "/webhooks/"); found {
			id, _, _ := strings.Cut(rest, "/")
			route = method + "/" + id
			if delay := c.discordDelay(route); delay > 0 {
				return Answer{Status: http.StatusTooManyRequests, RetryAfter: delay}, nil
			}
		}
	}
	var carried io.Reader
	if body != nil {
		carried = bytes.NewReader(body)
	}
	request, err := http.NewRequestWithContext(ctx, method, checked.String(), carried)
	if err != nil {
		return Answer{}, fmt.Errorf("build the request: %w", err)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
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
	if route != "" {
		c.recordDiscordRate(route, response.Header)
	}
	read, err := io.ReadAll(io.LimitReader(response.Body, c.Limits.ReadBytes))
	if err != nil {
		return Answer{}, fmt.Errorf("read what %s said back: %w", checked.Host, err)
	}
	return Answer{
		Status: response.StatusCode, Body: read, Took: time.Since(started),
		RetryAfter: RetryAfter(response.Header.Get("Retry-After"), started),
	}, nil
}

func RetryAfter(header string, from time.Time) time.Duration {
	header = strings.TrimSpace(header)
	if header == "" {
		return 0
	}
	wait := time.Duration(0)
	if seconds, err := strconv.ParseFloat(header, 64); err == nil && !math.IsNaN(seconds) && !math.IsInf(seconds, 0) {
		wait = time.Duration(min(max(seconds, 0), MaxRetryAfter.Seconds()) * float64(time.Second))
	} else if at, err := http.ParseTime(header); err == nil {
		wait = at.Sub(from)
	}
	if wait < 0 {
		return 0
	}
	return min(wait, MaxRetryAfter)
}

func (c *Caller) discordDelay(route string) time.Duration {
	c.rates.Lock()
	next := c.nextSend[c.buckets[route]]
	global := c.nextSend["global"]
	c.rates.Unlock()
	if global.After(next) {
		next = global
	}
	return max(0, time.Until(next))
}

func (c *Caller) recordDiscordRate(route string, headers http.Header) {
	c.rates.Lock()
	defer c.rates.Unlock()
	if c.buckets == nil {
		c.buckets = make(map[string]string)
		c.nextSend = make(map[string]time.Time)
	}
	_, id, _ := strings.Cut(route, "/")
	bucket := c.buckets[route]
	if name := headers.Get("X-RateLimit-Bucket"); name != "" {
		bucket = name + "/" + id
	}
	if bucket == "" {
		bucket = route
	}
	c.buckets[route] = bucket
	now := time.Now()
	delay := RetryAfter(headers.Get("Retry-After"), now)
	if headers.Get("X-RateLimit-Remaining") == "0" {
		delay = max(delay, RetryAfter(headers.Get("X-RateLimit-Reset-After"), now))
	}
	if headers.Get("X-RateLimit-Global") == "true" {
		bucket = "global"
	}
	if until := now.Add(delay); until.After(c.nextSend[bucket]) {
		c.nextSend[bucket] = until
	}
}

func (c *Caller) trust(config *tls.Config) { c.transport.TLSClientConfig = config }
