package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/Matthew-Ing/MI_API_Gateway/internal/tracing"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func main() {
	mux := http.NewServeMux()
	log.Println("User Service is running on port 8081")
	mux.HandleFunc("GET /healthz", healthCheck)
	mux.HandleFunc("GET /users", listUsers)
	mux.HandleFunc("GET /users/{id}", getUserByID)

	log.Println("User Service is running on port 8081")
	ctx := context.Background()
	shutdown, err := tracing.Init(ctx, "userservice")
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = shutdown(ctx) }()

	handler := otelhttp.NewHandler(mux, "userservice")
	log.Fatal(http.ListenAndServe(":8081", handler))
}
func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("User Service is running"))
}
func listUsers(w http.ResponseWriter, r *http.Request) {
	users := []User{
		{ID: 1, Name: "John Doe"},
		{ID: 2, Name: "Jane Doe"},
	}
	w.Header().Set("Content-Type", "application/json")

	_ = json.NewEncoder(w).Encode(users)

}
func getUserByID(w http.ResponseWriter, r *http.Request) {
	user := []User{{ID: 1, Name: "John Doe", Email: "john.doe@example.com"},
		{ID: 2, Name: "Jane Doe", Email: "jane.doe@example.com"}}
	userID, _ := strconv.Atoi(r.PathValue("id"))
	for _, user := range user {
		if user.ID == userID {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(user)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte("User not found"))

}
