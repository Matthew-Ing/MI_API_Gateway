package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Matthew-Ing/MI_API_Gateway/internal/circuitbreaker"
	"github.com/Matthew-Ing/MI_API_Gateway/internal/config"
)

func testCfg(upstream string) *config.Config {
	return &config.Config{
		Upstreams: map[string]config.Upstream{"orders": {URL: upstream}},
		Routes:    []config.Route{{Path: "/orders", Upstream: "orders"}},
	}
}

func TestMatch(t *testing.T) {
	cfg := testCfg("http://example")
	if Match(cfg, "/orders/1") == nil {
		t.Fatal("prefix")
	}
	if Match(cfg, "/nope") != nil {
		t.Fatal("unknown")
	}
}

func TestCircuitOpens(t *testing.T) {
	hits := 0
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		http.Error(w, "fail", http.StatusInternalServerError)
	}))
	t.Cleanup(up.Close)

	cfg := testCfg(up.URL)
	cb := circuitbreaker.NewRegistry(3, time.Hour)
	h := New(cfg, cb)

	var last int
	for i := 0; i < 4; i++ {
		req := httptest.NewRequest(http.MethodGet, "/orders", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		last = rec.Code
	}
	if last != http.StatusServiceUnavailable {
		t.Fatalf("got %d", last)
	}
	if hits != 3 {
		t.Fatalf("upstream hits %d, want 3", hits)
	}
}

func TestUnknownPath404(t *testing.T) {
	h := New(testCfg("http://127.0.0.1:1"), circuitbreaker.NewRegistry(5, time.Second))
	req := httptest.NewRequest(http.MethodGet, "/nope", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatal(rec.Code)
	}
}