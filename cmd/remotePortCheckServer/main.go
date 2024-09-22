package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

const (
	LIMIT_PER_SECOND = 100
)

var (
	rateLimiters = make(map[string]*RateLimiter)
	mu           sync.Mutex
)

type RateLimiter struct {
	requests    int
	lastRequest time.Time
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		requests:    0,
		lastRequest: time.Now(),
	}
}

func (rl *RateLimiter) Allow() bool {
	now := time.Now()
	if now.Sub(rl.lastRequest) > time.Second {
		rl.requests = 0
		rl.lastRequest = now
	}

	rl.requests++
	return rl.requests <= LIMIT_PER_SECOND
}

func rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			http.Error(w, "Unable to determine IP address", http.StatusInternalServerError)
			return
		}

		mu.Lock()
		limiter, exists := rateLimiters[ip]
		if !exists {
			limiter = NewRateLimiter()
			rateLimiters[ip] = limiter
		}
		mu.Unlock()

		if !limiter.Allow() {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	log.Println("Booting remotePortCheckServer")
	port := 8081
	serverAddress := fmt.Sprintf(":%d", port)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		type Response struct {
			Message string `json:"message"`
			Status  int    `json:"status"`
		}

		response := Response{
			Message: "Server is running",
			Status:  http.StatusOK,
		}

		// Marshal the struct to JSON
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	mux.HandleFunc("POST /query", func(w http.ResponseWriter, r *http.Request) {

		type Request struct {
			Host string `json:"host"`
			Port int    `json:"port"`
		}

		type Response struct {
			Status string `json:"status"`
		}

		// Decode the request body into a struct
		var request Request
		err := json.NewDecoder(r.Body).Decode(&request)

		if err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		log.Println("POST /query", request)

		// Check if the host is reachable
		timeout := 1000 * time.Millisecond
		_, err = net.DialTimeout("tcp", fmt.Sprintf("%s:%d", request.Host, request.Port), timeout)

		if err != nil {
			http.Error(w, "Host is not reachable", http.StatusServiceUnavailable)
			return
		}

		response := Response{
			Status: "reachable",
		}

		// Marshal the struct to JSON
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	server := &http.Server{
		Addr:    serverAddress,
		Handler: rateLimitMiddleware(mux),
	}

	log.Println("Starting server at", serverAddress)
	server.ListenAndServe()

}
