package http

import (
	"net/http"
	"time"
)

type Timeouts struct {
	ReadHeader time.Duration
	Idle       time.Duration
}

func DefaultTimeouts() Timeouts {
	return Timeouts{
		ReadHeader: 10 * time.Second,
		Idle:       2 * time.Minute,
	}
}

// NewServer leaves total timeouts unset so healthy large uploads can finish.
func NewServer(addr string, handler http.Handler, t Timeouts) *http.Server {
	return &http.Server{
		Addr:    addr,
		Handler: handler,

		ReadHeaderTimeout: t.ReadHeader,
		IdleTimeout:       t.Idle,
	}
}
