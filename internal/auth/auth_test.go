package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func ok() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestUserAuth(t *testing.T) {
	t.Setenv("JWT_SECRET", "user-secret")
	t.Setenv("ADMIN_JWT_SECRET", "admin-secret")

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mr.Close)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	raw := "demo-key"
	if err := rdb.Set(context.Background(), "apikey:"+HashKey(raw), "1", 0).Err(); err != nil {
		t.Fatal(err)
	}
	h := New(rdb)(ok())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("no creds: %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-API-Key", raw)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("api key: %d", rec.Code)
	}

	tok, err := GenerateToken("alice")
	if err != nil {
		t.Fatal(err)
	}
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("user jwt: %d", rec.Code)
	}

	adminTok, err := GenerateAdminToken()
	if err != nil {
		t.Fatal(err)
	}
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+adminTok)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("admin jwt on user auth: %d", rec.Code)
	}
}

func TestAdminAuth(t *testing.T) {
	t.Setenv("JWT_SECRET", "user-secret")
	t.Setenv("ADMIN_JWT_SECRET", "admin-secret")
	h := AdminAuth(ok())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatal(rec.Code)
	}

	userTok, _ := GenerateToken("alice")
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+userTok)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("user jwt on admin: %d", rec.Code)
	}

	adminTok, _ := GenerateAdminToken()
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+adminTok)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin jwt: %d", rec.Code)
	}
}