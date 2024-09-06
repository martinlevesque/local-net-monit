package networking

// using an external service, check if a given port is opened on the public IP

import (
	"encoding/json"
	"github.com/martinlevesque/local-net-monit/internal/httpTooling"
)

func IsPublicPortOpen(host string, port int) bool {
	remote_port_checker_base_url := "http://localhost:8081"

	body := make(map[string]interface{})

	body["host"] = host
	body["port"] = port

	status, response, err := httpTooling.Post(remote_port_checker_base_url, "/query", body)

	if err != nil {
		return false
	}

	if status != "200 OK" {
		return false
	}

	responseResult := make(map[string]interface{})

	json.Unmarshal([]byte(response), &responseResult)

	if checkResult, ok := responseResult["status"]; ok {
		return checkResult == "reachable"
	}

	return false
}
