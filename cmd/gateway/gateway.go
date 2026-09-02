package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"

	internalauth "github.com/Matthew-Ing/MI_API_Gateway/internal/auth"
	circuitbreaker "github.com/Matthew-Ing/MI_API_Gateway/internal/circuitbreaker"
	"github.com/Matthew-Ing/MI_API_Gateway/internal/config"
	"github.com/Matthew-Ing/MI_API_Gateway/internal/middleware"
	"github.com/Matthew-Ing/MI_API_Gateway/internal/proxy"
	"github.com/Matthew-Ing/MI_API_Gateway/internal/ratelimit"
	"github.com/Matthew-Ing/MI_API_Gateway/internal/tracing"
)

func main() {
	cfg, err := config.LoadConfig("configs/gateway.yaml")
	if err != nil {
		log.Fatal(err)
	}
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		log.Fatal("REDIS_ADDR is not set")
	}

	rdb := redis.NewClient(&redis.Options{Addr: addr})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatal(err)
	}
	internalauth.SeedSampleKey(context.Background(), rdb)

	if os.Getenv("JWT_SECRET") == "" {
		log.Fatal("JWT_SECRET is not set")
	}
	ctx := context.Background()
shutdown, err := tracing.Init(ctx, "gateway")
if err != nil {
	log.Fatal(err)
}
defer shutdown(ctx)

	cb := circuitbreaker.NewRegistry(10, 10*time.Second)
	h := middleware.Chain(
		middleware.RequestID,
		middleware.RequestLogger,
		middleware.RecoverPanic,
		middleware.Tracing,
		middleware.Metrics,
		internalauth.New(rdb),
		ratelimit.New(rdb),
	)(proxy.New(cfg, cb))

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("status ok"))
	})


	mux.Handle("/", h)
	log.Fatal(http.ListenAndServe(cfg.Listen, mux))
}
