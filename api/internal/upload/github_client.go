package upload

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/format/extension"
)

var repositoryPart = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,99}$`)

func githubRepository(raw string) (string, error) {
	address, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || address.Scheme != "https" || !strings.EqualFold(address.Hostname(), "github.com") || address.Port() != "" || address.User != nil || address.RawQuery != "" || address.Fragment != "" {
		return "", errors.New("Enter a GitHub repository address.")
	}
	parts := strings.Split(strings.Trim(address.Path, "/"), "/")
	if len(parts) != 2 {
		return "", errors.New("Enter the repository's main address.")
	}
	parts[1] = strings.TrimSuffix(parts[1], ".git")
	if !repositoryPart.MatchString(parts[0]) || !repositoryPart.MatchString(parts[1]) || parts[0] == "." || parts[1] == "." {
		return "", errors.New("Enter a valid GitHub repository address.")
	}
	return strings.ToLower(strings.Join(parts, "/")), nil
}

type githubRelease struct {
	ID          int64     `json:"id"`
	Tag         string    `json:"tag_name"`
	Draft       bool      `json:"draft"`
	Prerelease  bool      `json:"prerelease"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	} `json:"assets"`
}

type githubClient struct{ http *http.Client }

func newGitHubClient() githubClient {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
		if err != nil {
			return nil, err
		}
		for _, ip := range ips {
			if publicGitHubIP(ip) {
				return (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
			}
		}
		return nil, errors.New("GitHub resolved to a private address")
	}
	return githubClient{http: &http.Client{Timeout: 30 * time.Second, Transport: transport, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 || !githubHost(req.URL) {
			return errors.New("GitHub redirected to an unsafe address")
		}
		req.Header.Del("Authorization")
		return nil
	}}}
}

func publicGitHubIP(ip netip.Addr) bool {
	return ip.IsValid() && ip.IsGlobalUnicast() && !ip.IsPrivate() && !netip.MustParsePrefix("100.64.0.0/10").Contains(ip.Unmap())
}

func githubHost(u *url.URL) bool {
	if u.Scheme != "https" || u.User != nil || u.Port() != "" {
		return false
	}
	switch strings.ToLower(u.Hostname()) {
	case "api.github.com", "github.com", "codeload.github.com", "release-assets.githubusercontent.com", "objects.githubusercontent.com":
		return true
	}
	return false
}

func (g githubClient) get(ctx context.Context, path, accept string, limit int64) ([]byte, error) {
	address := "https://api.github.com" + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", accept)
	req.Header.Set("User-Agent", "Illarin-extension-import")
	response, err := g.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub returned %s", response.Status)
	}
	if response.ContentLength > limit {
		return nil, errors.New("GitHub's response is too large")
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, errors.New("GitHub's response is too large")
	}
	return data, nil
}

func (g githubClient) proof(ctx context.Context, repository string) (string, error) {
	body, err := g.get(ctx, "/repos/"+repository+"/contents/.illarin-proof", "application/vnd.github+json", 16<<10)
	if err != nil {
		return "", err
	}
	var file struct{ Type, Content, Encoding string }
	if err := json.Unmarshal(body, &file); err != nil {
		return "", err
	}
	if file.Type != "file" || file.Encoding != "base64" {
		return "", errors.New("the proof must be a file in the repository")
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(file.Content, "\n", ""))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(decoded)), nil
}

func (g githubClient) releases(ctx context.Context, repository string) ([]githubRelease, error) {
	all := make([]githubRelease, 0)
	for page := 1; page <= 100; page++ {
		body, err := g.get(ctx, fmt.Sprintf("/repos/%s/releases?per_page=100&page=%d", repository, page), "application/vnd.github+json", 2<<20)
		if err != nil {
			return nil, err
		}
		var batch []githubRelease
		if err := json.Unmarshal(body, &batch); err != nil {
			return nil, err
		}
		all = append(all, batch...)
		if len(batch) < 100 {
			return all, nil
		}
	}
	return nil, errors.New("the repository has too many releases for one check")
}

func (g githubClient) archive(ctx context.Context, repository, tag string, assetID int64) ([]byte, string, error) {
	path, filename := "/repos/"+repository+"/zipball/"+url.PathEscape(tag), tag+".zip"
	if assetID != 0 {
		path = fmt.Sprintf("/repos/%s/releases/assets/%d", repository, assetID)
		filename = "release.zip"
	}
	body, err := g.get(ctx, path, "application/octet-stream", extension.MaxArchiveBytes)
	return body, filename, err
}
