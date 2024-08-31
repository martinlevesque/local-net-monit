package httpTooling

import (
	"io"
	"log"
	"net/http"
)

func Get(baseUrl string, path string) (string, string) {
	var requesthToHost string

	if baseUrl == "" {
		requesthToHost = "http://localhost:8080"
	} else {
		requesthToHost = baseUrl
	}

	requesthTo := requesthToHost + path

	resp, err := http.Get(requesthTo)

	if err != nil {
		log.Fatalf("Failed to perform HTTP request: %v", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		log.Fatalf("Failed to read response body: %v", err)
	}

	return resp.Status, string(body)
}
