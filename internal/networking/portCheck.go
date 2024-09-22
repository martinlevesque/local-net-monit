package networking

import (
	"fmt"
	"net"
	"time"
)

func isUDPPortOpen(host string, port int) bool {
	address := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("udp", address, 2*time.Second)

	defer conn.Close()

	if err != nil {
		fmt.Println("udp dial error: ", err)
		return false
	}

	_, err = conn.Write([]byte("test"))

	if err != nil {
		fmt.Println("udp write error: ", err)
		return false
	}

	// Set a read deadline for the response
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buffer := make([]byte, 1024)
	_, err = conn.Read(buffer)

	// If we get an error on read, it's likely the port is closed
	if err != nil {
		fmt.Println("udp read error: ", err)
		return false
	}

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
