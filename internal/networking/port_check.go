package networking

import (
	"fmt"
	"net"
	"time"
)

func isUDPPortOpen(host string, port int) bool {
	address := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("udp", address, 2*time.Second)

	if err != nil {
		return false
	}

	defer conn.Close()

	return true
}

func isTCPPortOpen(host string, port int) bool {
	address := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", address, 2*time.Second)

	if err != nil {
		return false
	}

	defer conn.Close()

	return true
}
