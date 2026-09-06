package outbound

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
)

func TestAPlainPublicAddressIsAccepted(t *testing.T) {
	for _, raw := range []string{
		"https://hooks.example.com/publication",
		"https://hooks.example.com:443/publication?source=illarin",
		"https://93.184.216.34/publication",
	} {
		if _, err := Check(raw); err != nil {
			t.Errorf("Check(%q) = %v, want it accepted", raw, err)
		}
	}
}

func TestAnAddressOutsideThePolicyIsRefused(t *testing.T) {
	cases := map[string]string{
		"plain http":      "http://hooks.example.com/publication",
		"another scheme":  "ftp://hooks.example.com/publication",
		"another port":    "https://hooks.example.com:8443/publication",
		"credentials":     "https://user:pass@hooks.example.com/publication",
		"a user":          "https://user@hooks.example.com/publication",
		"a fragment":      "https://hooks.example.com/publication#part",
		"no host":         "https:///publication",
		"not an address":  "hooks.example.com/publication",
		"a loopback host": "https://127.0.0.1/publication",
		"a private host":  "https://10.0.0.7/publication",
		"the metadata ip": "https://169.254.169.254/publication",
		"a bare name":     "https://localhost/publication",
	}
	for name, raw := range cases {
		if _, err := Check(raw); err == nil {
			t.Errorf("%s: Check(%q) was accepted", name, raw)
		}
	}
}

func TestPublicSpaceRefusesEveryReservedRange(t *testing.T) {
	refused := []string{
		"0.0.0.0", "127.0.0.1", "10.1.2.3", "172.16.9.9", "192.168.1.1",
		"169.254.169.254", "100.64.0.1", "192.0.2.5", "198.18.0.1",
		"198.51.100.5", "203.0.113.5", "224.0.0.1", "255.255.255.255",
		"::1", "::", "fe80::1", "fc00::1", "fd00::abcd", "ff02::1",
		"::ffff:127.0.0.1", "::ffff:10.0.0.1", "2001:db8::1", "64:ff9b::7f00:1",
	}
	for _, raw := range refused {
		at := netip.MustParseAddr(raw)
		if Public(at) {
			t.Errorf("Public(%s) = true, want false", raw)
		}
	}
	for _, raw := range []string{"93.184.216.34", "1.1.1.1", "2606:4700:4700::1111"} {
		at := netip.MustParseAddr(raw)
		if !Public(at) {
			t.Errorf("Public(%s) = false, want true", raw)
		}
	}
}

func TestACallReachesTheReceiverItResolvedTo(t *testing.T) {
	receiver := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != "example.com" {
			t.Errorf("Host = %q, want the original host", r.Host)
		}
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("thanks"))
	}))
	defer receiver.Close()
	caller := receiverCaller(t, receiver)

	answer, err := caller.Post(
		context.Background(), "https://example.com/publication", nil, []byte(`{}`),
	)

	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if answer.Status != http.StatusAccepted {
		t.Errorf("status = %d, want 202", answer.Status)
	}
	if string(answer.Body) != "thanks" {
		t.Errorf("body = %q, want %q", answer.Body, "thanks")
	}
}

func TestARedirectIsReportedAndNeverFollowed(t *testing.T) {
	var reached int
	receiver := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached++
		http.Redirect(w, r, "https://hooks.example.com/moved", http.StatusFound)
	}))
	defer receiver.Close()
	caller := receiverCaller(t, receiver)

	answer, err := caller.Post(
		context.Background(), "https://example.com/publication", nil, []byte(`{}`),
	)

	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if answer.Status != http.StatusFound {
		t.Errorf("status = %d, want 302", answer.Status)
	}
	if reached != 1 {
		t.Errorf("the receiver was reached %d times, want 1", reached)
	}
}

func TestAHostResolvingOutsidePublicSpaceIsRefused(t *testing.T) {
	reaching := Reaching{Resolve: func(context.Context, string) ([]netip.Addr, error) {
		return []netip.Addr{
			netip.MustParseAddr("10.0.0.7"),
			netip.MustParseAddr("::1"),
		}, nil
	}}

	_, err := reaching.At(context.Background(), "hooks.example.com", "443")

	if !errors.Is(err, ErrNotPublic) {
		t.Errorf("error = %v, want it to name the address policy", err)
	}
}

func TestAHostResolvingNowhereIsRefused(t *testing.T) {
	reaching := Reaching{Resolve: func(context.Context, string) ([]netip.Addr, error) {
		return nil, nil
	}}

	_, err := reaching.At(context.Background(), "hooks.example.com", "443")

	if err == nil {
		t.Fatal("a host that resolved to nothing was allowed")
	}
}

func TestOnePublicResultAmongPrivateOnesIsDialed(t *testing.T) {
	reaching := Reaching{Resolve: func(context.Context, string) ([]netip.Addr, error) {
		return []netip.Addr{
			netip.MustParseAddr("127.0.0.1"),
			netip.MustParseAddr("93.184.216.34"),
		}, nil
	}}

	at, err := reaching.At(context.Background(), "hooks.example.com", "443")

	if err != nil {
		t.Fatalf("at: %v", err)
	}
	if at != "93.184.216.34:443" {
		t.Errorf("at = %q, want the public result", at)
	}
}

func TestAResolverFailureIsRefused(t *testing.T) {
	reaching := Reaching{Resolve: func(context.Context, string) ([]netip.Addr, error) {
		return nil, errors.New("no such host")
	}}

	if _, err := reaching.At(context.Background(), "hooks.example.com", "443"); err == nil {
		t.Fatal("a failed lookup was allowed")
	}
}

func TestAnOversizedAnswerIsCutOff(t *testing.T) {
	receiver := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(strings.Repeat("x", 4096)))
	}))
	defer receiver.Close()
	caller := receiverCaller(t, receiver)
	caller.Limits.ReadBytes = 64

	answer, err := caller.Post(
		context.Background(), "https://example.com/publication", nil, []byte(`{}`),
	)

	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if len(answer.Body) != 64 {
		t.Errorf("read %d bytes, want 64", len(answer.Body))
	}
}

func TestAnUnreachableReceiverIsAnError(t *testing.T) {
	receiver := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	caller := receiverCaller(t, receiver)
	receiver.Close()

	_, err := caller.Post(
		context.Background(), "https://example.com/publication", nil, []byte(`{}`),
	)

	if err == nil {
		t.Fatal("a closed receiver reported success")
	}
}

// receiverCaller points a caller at one test receiver, which lives on loopback
// and would otherwise be refused.
func receiverCaller(t *testing.T, receiver *httptest.Server) *Caller {
	t.Helper()
	at := netip.MustParseAddrPort(strings.TrimPrefix(receiver.URL, "https://"))
	caller := NewCaller(DefaultLimits())
	caller.Reach = func(context.Context, string, string) (string, error) {
		return at.String(), nil
	}
	caller.trust(receiver.Client().Transport.(*http.Transport).TLSClientConfig)
	return caller
}
