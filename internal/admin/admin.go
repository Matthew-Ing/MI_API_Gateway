package admin

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	internalauth "github.com/Matthew-Ing/MI_API_Gateway/internal/auth"
	"github.com/Matthew-Ing/MI_API_Gateway/internal/circuitbreaker"
	"github.com/Matthew-Ing/MI_API_Gateway/internal/config"
	"github.com/redis/go-redis/v9"
)

type Server struct {
	rdb *redis.Client
	cfg *config.Config
	cb  *circuitbreaker.Registry
}

func New(rdb *redis.Client, cfg *config.Config, cb *circuitbreaker.Registry) *Server {
	return &Server{rdb: rdb, cfg: cfg, cb: cb}
}

func (s *Server) Login(w http.ResponseWriter, r *http.Request) {
	password := r.URL.Query().Get("password")
	if password == "" {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		var body struct {
			Password string `json:"password"`
		}
		if json.Unmarshal(raw, &body) == nil {
			password = body.Password
		}
		if password == "" {
			if vals, err := url.ParseQuery(string(raw)); err == nil {
				password = vals.Get("password")
			}
		}
	}
	if password == "" || password != os.Getenv("ADMIN_PASSWORD") {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	tok, err := internalauth.GenerateAdminToken()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"token": tok})
}

func (s *Server) CreateKey(w http.ResponseWriter, r *http.Request) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	raw := hex.EncodeToString(b)
	hash := internalauth.HashKey(raw)
	if err := s.rdb.Set(r.Context(), "apikey:"+hash, "1", 0).Err(); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"key": raw, "hash": hash})
}

func (s *Server) RevokeKey(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	n, err := s.rdb.Del(r.Context(), "apikey:"+hash).Result()
	if err != nil || n == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) Health(w http.ResponseWriter, r *http.Request) {
	client := &http.Client{Timeout: 1 * time.Second}
	type up struct {
		Name    string `json:"name"`
		URL     string `json:"url"`
		OK      bool   `json:"ok"`
		Breaker string `json:"breaker"`
	}
	out := make([]up, 0, len(s.cfg.Upstreams))
	for name, u := range s.cfg.Upstreams {
		ok := false
		resp, err := client.Get(u.URL + "/healthz")
		if err == nil {
			ok = resp.StatusCode < 500
			resp.Body.Close()
		}
		out = append(out, up{
			Name:    name,
			URL:     u.URL,
			OK:      ok,
			Breaker: s.cb.For(name).Current().String(),
		})
	}
	json.NewEncoder(w).Encode(out)
}