package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	LIMIT_PER_SECOND             = 100
	LIMIT_QUERY_HOSTS_PER_MINUTE = 1
)

var (
	rateLimiters     = make(map[string]*RateLimiter)
	hostRateLimiters = make(map[string]*HostRateLimiter)
	mu               sync.Mutex
	hostMu           sync.Mutex
)

type RateLimiter struct {
	requests    int
	lastRequest time.Time
}

type HostRateLimiter struct {
	hosts []string

	lastRequest time.Time
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		requests:    0,
		lastRequest: time.Now(),
	}
}

func NewHostRateLimiter() *HostRateLimiter {
	return &HostRateLimiter{
		hosts:       make([]string, 0),
		lastRequest: time.Now(),
	}
}

func (rl *RateLimiter) Allow(limit int, interval time.Duration) bool {
	now := time.Now()

	if now.Sub(rl.lastRequest) > interval {
		rl.requests = 0
		rl.lastRequest = now
	}

	rl.requests++
	return rl.requests <= limit
}

func (hrl *HostRateLimiter) Allow(host string, interval time.Duration) bool {
	now := time.Now()

	if now.Sub(hrl.lastRequest) > interval {
		hrl.hosts = make([]string, 0)
		hrl.lastRequest = now
	}

	// Check if the host is already in the list in one line
	hostExists := false

	for _, h := range hrl.hosts {
		if h == host {
			hostExists = true
		}
	}

	if !hostExists {
		hrl.hosts = append(hrl.hosts, host)
	}

	return len(hrl.hosts) <= LIMIT_QUERY_HOSTS_PER_MINUTE
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

		if !limiter.Allow(LIMIT_PER_SECOND, time.Second) {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func hostRateLimit(fromIp string, host string) bool {
	hostMu.Lock()
	defer hostMu.Unlock()

	limiter, exists := hostRateLimiters[fromIp]

	if !exists {
		limiter = NewHostRateLimiter()
		hostRateLimiters[fromIp] = limiter
	}

	return limiter.Allow(host, time.Minute)
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

		// Apply host rate limiter
		fromIp, _, err := net.SplitHostPort(r.RemoteAddr)

		if err != nil {
			http.Error(w, "Unable to determine IP address", http.StatusInternalServerError)
			return
		}

		cleanedHost := ""

		cleanedHost = strings.TrimSpace(strings.ToLower(request.Host))

		if !hostRateLimit(fromIp, cleanedHost) {
			http.Error(w, "Rate limit for this host exceeded", http.StatusTooManyRequests)
			return
		}

		// Check if the host is reachable
		timeout := 1000 * time.Millisecond
		_, err = net.DialTimeout("tcp", fmt.Sprintf("%s:%d", request.Host, request.Port), timeout)

		w.Header().Set("Content-Type", "application/json")

		response := Response{
			Status: "unreachable",
		}

		if err != nil {
			response = Response{
				Status: "unreachable",
			}
		}

		// Marshal the struct to JSON
		json.NewEncoder(w).Encode(response)
	})

	server := &http.Server{
		Addr:    serverAddress,
		Handler: rateLimitMiddleware(mux),
	}

	log.Println("Starting server at", serverAddress)
	server.ListenAndServe()
}
