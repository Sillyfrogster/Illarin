package main

import (
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/config"
)

// newServer leaves total timeouts unset so healthy large uploads can finish.
func newServer(addr string, handler http.Handler, t config.ServerTimeouts) *http.Server {
	return &http.Server{
		Addr:    addr,
		Handler: handler,

		ReadHeaderTimeout: t.ReadHeader,
		IdleTimeout:       t.Idle,
	}
}
