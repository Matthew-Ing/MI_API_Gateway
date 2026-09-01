package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/redis/go-redis/v9"

	internalauth "github.com/Matthew-Ing/apigateway/internal/auth"
	"github.com/Matthew-Ing/apigateway/internal/config"
	"github.com/Matthew-Ing/apigateway/internal/middleware"
	"github.com/Matthew-Ing/apigateway/internal/proxy"
	"github.com/Matthew-Ing/apigateway/internal/ratelimit"
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

	if os.Getenv("JWT_SECRET") == "" {
		log.Fatal("JWT_SECRET is not set")
	}

	h := middleware.Chain(
		middleware.RequestID,
		middleware.RequestLogger,
		middleware.RecoverPanic,
		internalauth.New(rdb),
		ratelimit.New(rdb),
	)(proxy.New(cfg))

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("status ok"))
	})
	mux.Handle("/", h)
	log.Fatal(http.ListenAndServe(cfg.Listen, mux))
}
