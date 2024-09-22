package httpTooling

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
)

func Get(baseUrl string, path string) (string, string) {
	var requesToHost string

	if baseUrl == "" {
		requesToHost = "http://localhost:8080"
	} else {
		requesToHost = baseUrl
	}

	requesthTo := requesToHost + path

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

func Post(baseUrl string, path string, body map[string]interface{}) (string, string, error) {
	var requestToHost string

	if baseUrl == "" {
		requestToHost = "http://localhost:8080"
	} else {
		requestToHost = baseUrl
	}

	requestTo := requestToHost + path

	json_data, err := json.Marshal(body)

	if err != nil {
		return "", "", err
	}

	resp, err := http.Post(requestTo, "application/json",
		bytes.NewBuffer(json_data))

	if err != nil {
		return "", "", err
	}

	defer resp.Body.Close()

	resBody, err := io.ReadAll(resp.Body)

	if err != nil {
		return "", "", err
	}

	return resp.Status, string(resBody), nil
}
