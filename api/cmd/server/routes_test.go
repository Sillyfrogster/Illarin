package main

import (
	"slices"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/apitest/full"
	"github.com/Sillyfrogster/Illarin/api/internal/connect"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/gin-gonic/gin"
)

func TestTheTestRouterServesTheSameRoutesAsTheServer(t *testing.T) {
	t.Parallel()
	assets, links := &work.Service{}, &connect.Apps{}

	served := routesOf(t, func(r *gin.Engine) error {
		return registerRoutes(
			r, services{Assets: assets, Links: links}, api.DefaultDeadlines(), apitest.Ready)
	})
	tested := routesOf(t, func(r *gin.Engine) error {
		return full.Register(
			r, apitest.Services{Assets: assets, Links: links}, api.DefaultDeadlines())
	})

	if !slices.Equal(served, tested) {
		t.Fatalf("the server and the test router disagree about the routes.\nserver: %v\ntests:  %v",
			missing(served, tested), missing(tested, served))
	}
}

func routesOf(t *testing.T, register func(*gin.Engine) error) []string {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	if err := register(r); err != nil {
		t.Fatalf("register: %v", err)
	}
	listed := make([]string, 0, len(r.Routes()))
	for _, route := range r.Routes() {
		listed = append(listed, route.Method+" "+route.Path)
	}
	slices.Sort(listed)
	return listed
}

func missing(from, other []string) []string {
	only := make([]string, 0)
	for _, route := range from {
		if !slices.Contains(other, route) {
			only = append(only, route)
		}
	}
	return only
}
