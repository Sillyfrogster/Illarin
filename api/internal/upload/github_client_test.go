package upload

import (
	"net/http"
	"net/netip"
	"net/url"
	"testing"
)

func TestGitHubFetchesStayOnPublicGitHubHosts(t *testing.T) {
	t.Parallel()
	for _, address := range []string{
		"http://api.github.com/repos/example/repo",
		"https://api.github.com.evil.test/",
		"https://127.0.0.1/",
		"https://github.com:8443/",
	} {
		parsed, _ := url.Parse(address)
		if githubHost(parsed) {
			t.Errorf("accepted redirect to %s", address)
		}
	}
	for _, address := range []string{"127.0.0.1", "10.1.2.3", "169.254.1.1", "100.64.1.1", "::1", "fd00::1"} {
		if publicGitHubIP(netip.MustParseAddr(address)) {
			t.Errorf("accepted private address %s", address)
		}
	}
	client := newGitHubClient().http
	bad, _ := http.NewRequest("GET", "https://example.com/", nil)
	if err := client.CheckRedirect(bad, []*http.Request{{}}); err == nil {
		t.Fatal("followed redirect away from GitHub")
	}
}
