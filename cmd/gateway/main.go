package main

import (
	"log"
	"net/http"
	"os"

	internalauth "github.com/Matthew-Ing/apigateway/internal/auth"
	"github.com/Matthew-Ing/apigateway/internal/config"
	"github.com/Matthew-Ing/apigateway/internal/middleware"
	"github.com/Matthew-Ing/apigateway/internal/proxy"
)

func main() {
	cfg, err := config.LoadConfig("configs/gateway.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	m := middleware.Chain(
		middleware.RequestID,
		middleware.RequestLogger,
		middleware.RecoverPanic,
		internalauth.JWT,
	)

	p := proxy.New(cfg)
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("status ok"))
	})
	mux.Handle("/", m(p))

	log.Println("Gateway is running on", cfg.Listen)
	log.Println("JWT_SECRET is", os.Getenv("JWT_SECRET"))
	genToken, err := internalauth.GenerateToken("123")
	if err != nil {
		log.Fatalf("Failed to generate token: %v", err)
	}
	log.Println("Generated token:", genToken)
	log.Fatal(http.ListenAndServe(cfg.Listen, mux))
}
