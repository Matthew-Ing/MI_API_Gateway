package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
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
	log.Fatal(http.ListenAndServe(":8082", mux))
}
func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Order Service is running"))
}
func listOrders(w http.ResponseWriter, r *http.Request) {
	orders := []Order{
		{ID: 1, Name: "Order 1"},
		{ID: 2, Name: "Order 2"},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}
func getOrderByID(w http.ResponseWriter, r *http.Request) {
	orders := []Order{
		{ID: 1, Name: "Order 1"},
		{ID: 2, Name: "Order 2"},
	}
	orderID, _ := strconv.Atoi(r.PathValue("id"))
	for _, order := range orders {
		if order.ID == orderID {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(order)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("Order not found"))
}
