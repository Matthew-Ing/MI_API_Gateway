package main

import (
	"context"
	"encoding/json"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"strconv"

	"github.com/Matthew-Ing/MI_API_Gateway/internal/tracing"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type Order struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func main() {
	mux := http.NewServeMux()
	log.Println("Order Service is running on port 8082")
	mux.HandleFunc("GET /healthz", healthCheck)
	mux.HandleFunc("GET /orders", listOrders)
	mux.HandleFunc("GET /orders/{id}", getOrderByID)
	log.Println("Order Service is running on port 8082")
	ctx := context.Background()
	shutdown, err := tracing.Init(ctx, "orderservice")
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = shutdown(ctx) }()

	handler := otelhttp.NewHandler(mux, "orderservice")
	log.Fatal(http.ListenAndServe(":8082", handler))
}
func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Order Service is running"))
}
func listOrders(w http.ResponseWriter, r *http.Request) {
	if maybeFail(w) {
		return
	}
	orders := []Order{
		{ID: 1, Name: "Order 1"},
		{ID: 2, Name: "Order 2"},
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(orders)
}
func getOrderByID(w http.ResponseWriter, r *http.Request) {
	if maybeFail(w) {
		return
	}
	orders := []Order{
		{ID: 1, Name: "Order 1"},
		{ID: 2, Name: "Order 2"},
	}
	orderID, _ := strconv.Atoi(r.PathValue("id"))
	for _, order := range orders {
		if order.ID == orderID {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(order)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte("Order not found"))
}

func maybeFail(w http.ResponseWriter) bool {
	s := os.Getenv("FAIL_RATE")
	if s == "" {
		return false
	}
	rate, err := strconv.ParseFloat(s, 64)
	if err != nil || rate <= 0 {
		return false
	}
	if rand.Float64() < rate {
		http.Error(w, "injected failure", http.StatusInternalServerError)
		return true
	}
	return false
}
