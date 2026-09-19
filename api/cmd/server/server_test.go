package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/config"
)

func serve(t *testing.T, handler http.Handler, timeouts config.ServerTimeouts) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	server := newServer(listener.Addr().String(), handler, timeouts)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })

	return "http://" + listener.Addr().String()
}

func TestAConnectionThatNeverFinishesItsHeadersIsCutOff(t *testing.T) {
	t.Parallel()
	base := serve(t, http.NotFoundHandler(), config.ServerTimeouts{
		ReadHeader: 200 * time.Millisecond,
		Idle:       time.Minute,
	})

	conn, err := net.Dial("tcp", base[len("http://"):])
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	if _, err := fmt.Fprint(conn, "GET /v1/assets HTTP/1.1\r\nHost: illarin.test\r\n"); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}

	if _, err := io.ReadAll(conn); err != nil {
		t.Fatalf("the connection was still open after five seconds: %v", err)
	}
}

func TestAnUploadArrivingSlowlyRunsToCompletion(t *testing.T) {
	t.Parallel()
	r, session := harness.NewVerifiedRouterWith(t, 1<<20, api.Deadlines{
		JSON:     300 * time.Millisecond,
		Upload:   30 * time.Second,
		Download: 30 * time.Second,
		Collect:  30 * time.Second,
		Verify:   30 * time.Second,
	})
	base := serve(t, r, config.ServerTimeouts{ReadHeader: 300 * time.Millisecond, Idle: time.Minute})

	built := apitest.UploadRequest(t, apitest.ExampleMetadata("Trickle"), []byte("bytes that take their time"))
	form, err := io.ReadAll(built.Body)
	if err != nil {
		t.Fatalf("read form: %v", err)
	}

	body := &slowReader{rest: form, chunk: len(form)/5 + 1, pause: 200 * time.Millisecond}
	req, err := http.NewRequest(http.MethodPost, base+"/v1/assets", body)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", built.Header.Get("Content-Type"))
	apitest.Authorized(req, session)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		answer, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 202. body: %s", resp.StatusCode, answer)
	}
}

type slowReader struct {
	rest  []byte
	chunk int
	pause time.Duration
}

func (s *slowReader) Read(p []byte) (int, error) {
	if len(s.rest) == 0 {
		return 0, io.EOF
	}
	time.Sleep(s.pause)
	n := copy(p, s.rest[:min(s.chunk, len(s.rest))])
	s.rest = s.rest[n:]
	return n, nil
}
