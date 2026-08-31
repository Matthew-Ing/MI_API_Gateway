package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/Matthew-Ing/apigateway/internal/config"
)

// Match — given *config.Config and r.URL.Path, find the first route whose path is a prefix (/users matches /users and /users/1).
func Match(config *config.Config, path string) *config.Route {
	for _, route := range config.Routes {
		if strings.HasPrefix(path, route.Path) {
			return &route
		}
	}
	return nil
}

// Resolve — route.Upstream → cfg.Upstreams[name].URL.
func Resolve(cfg *config.Config, route *config.Route) (string, bool) {
	u, ok := cfg.Upstreams[route.Upstream]
	if !ok || u.URL == "" {
		return "", false
	}
	return u.URL, true
}

// Proxy — httputil.NewSingleHostReverseProxy(targetURL) and ServeHTTP.

func Proxy(cfg *config.Config, w http.ResponseWriter, r *http.Request) {
	route := Match(cfg, r.URL.Path)
	if route == nil {
		http.NotFound(w, r)
		return
	}
	raw, ok := Resolve(cfg, route)
	if !ok {
		http.Error(w, "unknown upstream", http.StatusBadGateway)
		return
	}
	target, err := url.Parse(raw)
	if err != nil || target.Host == "" {
		http.Error(w, "bad upstream", http.StatusBadGateway)
		return
	}
	httputil.NewSingleHostReverseProxy(target).ServeHTTP(w, r)
}

func New(cfg *config.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Proxy(cfg, w, r)
	})
}
