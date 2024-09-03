package networking

// using an external service, check if a given port is opened on the public IP

import (
	"encoding/json"
	"github.com/martinlevesque/local-net-monit/internal/httpTooling"
	"log"
)

func IsPublicPortOpen(host string, port int) bool {
	body := make(map[string]interface{})

	body["host"] = host
	body["ports"] = []int{port}

	log.Println("Checking public portttttt")
	status, response, err := httpTooling.Post("https://portchecker.io/api/v1", "/query", body)
	log.Println("Checked public portttttt 222")

	if err != nil {
		log.Println("Failed to check public port:", err)
		return false
	}

	log.Println("Checkkk status", status)

	if status != "200 OK" {
		log.Println("Failed to check public port: ", status)
		return false
	}

	log.Println("Checkkk response", response)

	responseResult := make(map[string]interface{})

	json.Unmarshal([]byte(response), &responseResult)

	if checkResult, ok := responseResult["check"]; ok {
		if len(checkResult.([]interface{})) > 0 {
			log.Println("Checkkk result", checkResult.([]interface{})[0])
			return checkResult.([]interface{})[0].(map[string]interface{})["status"] == true
		}
	}

	return false
}
