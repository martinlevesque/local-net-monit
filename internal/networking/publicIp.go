package networking

import (
	"fmt"
	"github.com/martinlevesque/local-net-monit/internal/httpTooling"
	"strings"
)

func ResolverPublicIp() (string, error) {
	status, body := httpTooling.Get("http://checkip.amazonaws.com", "")

	statusStripped := strings.Trim(status, " \n")

	if statusStripped != "200" {
		return "", fmt.Errorf("Failed to get public IP: %s", statusStripped)
	}

	return strings.Trim(body, " \n"), nil
}
