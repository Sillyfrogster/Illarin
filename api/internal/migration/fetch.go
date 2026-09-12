package migration

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var ErrHostNotAllowed = errors.New("the host is not on the migration allowlist")

type FetchedMedia struct {
	MediaType string
	Body      []byte
}

type Fetcher interface {
	Fetch(ctx context.Context, address string) (FetchedMedia, error)
}

type FetchLimits struct {
	Bytes     int64
	Timeout   time.Duration
	Redirects int
}

func DefaultFetchLimits() FetchLimits {
	return FetchLimits{Bytes: 8 << 20, Timeout: 20 * time.Second, Redirects: 3}
}

type AllowlistedFetcher struct {
	hosts  map[string]struct{}
	limits FetchLimits
	client *http.Client
}

func NewAllowlistedFetcher(hosts []string, limits FetchLimits) *AllowlistedFetcher {
	allowed := make(map[string]struct{}, len(hosts))
	for _, host := range hosts {
		allowed[strings.ToLower(strings.TrimSpace(host))] = struct{}{}
	}
	fetcher := &AllowlistedFetcher{hosts: allowed, limits: limits}
	fetcher.client = &http.Client{
		Timeout: limits.Timeout,
		CheckRedirect: func(request *http.Request, via []*http.Request) error {
			if len(via) >= limits.Redirects {
				return fmt.Errorf("more than %d redirects", limits.Redirects)
			}
			return fetcher.allow(request.URL)
		},
	}
	return fetcher
}

func (fetcher *AllowlistedFetcher) Allows(address string) bool {
	parsed, err := url.Parse(address)
	if err != nil {
		return false
	}
	return fetcher.allow(parsed) == nil
}

func (fetcher *AllowlistedFetcher) allow(address *url.URL) error {
	if address.Scheme != "https" {
		return fmt.Errorf("%w: %s is not https", ErrHostNotAllowed, address.Scheme)
	}
	if _, found := fetcher.hosts[strings.ToLower(address.Hostname())]; !found {
		return ErrHostNotAllowed
	}
	return nil
}

func (fetcher *AllowlistedFetcher) Fetch(ctx context.Context, address string) (FetchedMedia, error) {
	parsed, err := url.Parse(address)
	if err != nil {
		return FetchedMedia{}, fmt.Errorf("read the image address: %w", err)
	}
	if err := fetcher.allow(parsed); err != nil {
		return FetchedMedia{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, fetcher.limits.Timeout)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return FetchedMedia{}, fmt.Errorf("build the image request: %w", err)
	}
	response, err := fetcher.client.Do(request)
	if err != nil {
		return FetchedMedia{}, fmt.Errorf("fetch the image: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return FetchedMedia{}, fmt.Errorf("the host answered %d", response.StatusCode)
	}
	mediaType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || !strings.HasPrefix(mediaType, "image/") {
		return FetchedMedia{}, fmt.Errorf("the host answered with %q", response.Header.Get("Content-Type"))
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, fetcher.limits.Bytes+1))
	if err != nil {
		return FetchedMedia{}, fmt.Errorf("read the image: %w", err)
	}
	if int64(len(body)) > fetcher.limits.Bytes {
		return FetchedMedia{}, fmt.Errorf("the image is larger than %d bytes", fetcher.limits.Bytes)
	}
	return FetchedMedia{MediaType: mediaType, Body: body}, nil
}

func FetchHosts(addresses []string) []string {
	seen := make(map[string]struct{}, len(addresses))
	hosts := make([]string, 0, len(addresses))
	for _, address := range addresses {
		parsed, err := url.Parse(address)
		if err != nil || parsed.Hostname() == "" {
			continue
		}
		host := strings.ToLower(parsed.Hostname())
		if _, found := seen[host]; found {
			continue
		}
		seen[host] = struct{}{}
		hosts = append(hosts, host)
	}
	return hosts
}
