package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/Matthew-Ing/MI_API_Gateway/internal/circuitbreaker"
	"github.com/Matthew-Ing/MI_API_Gateway/internal/config"
	"github.com/Matthew-Ing/MI_API_Gateway/internal/metrics"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
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

func Proxy(cfg *config.Config, cb *circuitbreaker.Registry, w http.ResponseWriter, r *http.Request) {
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

	br := cb.For(route.Upstream)
	if !br.Allow() {
		metrics.BreakerOpen.WithLabelValues(route.Upstream).Inc()
		http.Error(w, "circuit open", http.StatusServiceUnavailable)
		metrics.BreakerState.WithLabelValues(route.Upstream).Set(float64(br.Current()))
		return
	}

	p := httputil.NewSingleHostReverseProxy(target)
	base := p.Director
	p.Director = func(req *http.Request) {
	base(req)
	req.Host = target.Host
}
p.Transport = otelhttp.NewTransport(http.DefaultTransport)
	p.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		br.RecordFailure()
		metrics.BreakerState.WithLabelValues(route.Upstream).Set(float64(br.Current()))
		http.Error(w, "bad gateway", http.StatusBadGateway)
	}
	p.ModifyResponse = func(res *http.Response) error {
		if res.StatusCode >= 500 {
			br.RecordFailure()
			metrics.BreakerState.WithLabelValues(route.Upstream).Set(float64(br.Current()))
			} else {
				br.RecordSuccess()
				metrics.BreakerState.WithLabelValues(route.Upstream).Set(float64(br.Current()))
			}
		return nil
	}
	p.ServeHTTP(w, r)
}

func New(cfg *config.Config, cb *circuitbreaker.Registry) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Proxy(cfg, cb, w, r)
	})
}
