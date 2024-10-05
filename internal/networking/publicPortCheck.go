package networking

// using an external service, check if a given port is opened on the public IP

import (
	"encoding/json"
	"fmt"
	"github.com/martinlevesque/local-net-monit/internal/env"
	"github.com/martinlevesque/local-net-monit/internal/httpTooling"
)

func IsPublicPortOpen(host string, port int) (bool, error) {
	remote_port_checker_base_url := env.EnvVar("REMOTE_PORT_CHECKER_BASE_URL", "https://remote-port-checker-server.fly.dev")

	body := make(map[string]interface{})

	body["host"] = host
	body["port"] = port

	status, response, err := httpTooling.Post(remote_port_checker_base_url, "/query", body)

	if err != nil {
		return false, err
	}

	if status != "200 OK" {
		return false, fmt.Errorf("Unexpected status code: %s", status)
	}

	responseResult := make(map[string]interface{})

	json.Unmarshal([]byte(response), &responseResult)

	if checkResult, ok := responseResult["status"]; ok {
		return checkResult == "reachable", nil
	}

	return false, fmt.Errorf("status field missing in the response")
}
