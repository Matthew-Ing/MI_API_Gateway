package ratelimit

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimit struct {
	rdb    *redis.Client
	max    int64
	window time.Duration
}

func New(rdb *redis.Client) func(http.Handler) http.Handler {
	max := int64(10)
	if v := os.Getenv("RATE_LIMIT_MAX"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			max = n
		}
	}
	window := 60 * time.Second
	if v := os.Getenv("RATE_LIMIT_WINDOW_SEC"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			window = time.Duration(n) * time.Second
		}
	}
	rl := &RateLimit{rdb: rdb, max: max, window: window}
	return rl.Handler
}

func identity(r *http.Request) string {
	if k := r.Header.Get("X-API-Key"); k != "" {
		sum := sha256.Sum256([]byte(k))
		return hex.EncodeToString(sum[:])
	}
	if a := r.Header.Get("Authorization"); strings.HasPrefix(a, "Bearer ") {
		sum := sha256.Sum256([]byte(a))
		return hex.EncodeToString(sum[:])
	}
	return r.RemoteAddr
}

func (rl *RateLimit) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := identity(r)
		key := "ratelimit:" + id

		n, err := rl.rdb.Incr(r.Context(), key).Result()
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if n == 1 {
			_ = rl.rdb.Expire(r.Context(), key, rl.window).Err()
		}
		if n > rl.max {
			w.Header().Set("Retry-After", strconv.Itoa(int(rl.window.Seconds())))
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests) // 429
			return
		}
		next.ServeHTTP(w, r)
	})
}
