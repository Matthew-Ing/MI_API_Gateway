package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestTooManyRequests(t *testing.T) {
	t.Setenv("RATE_LIMIT_MAX", "2")
	t.Setenv("RATE_LIMIT_WINDOW_SEC", "60")
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mr.Close)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	h := New(rdb)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	var last int
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-API-Key", "k")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		last = rec.Code
	}
	if last != http.StatusTooManyRequests {
		t.Fatalf("got %d", last)
	}
}