package main

import (
	"fmt"
	"github.com/martinlevesque/local-net-monit/networking"
	"log"
)

func main() {
	// Get the local IP address
	localIP := networking.LocalIPResolver()

	log.Printf("Local IP: %s\n", localIP.String())

	ipNet, err := networking.FindSubnetForIP(localIP)

	if err != nil {
		fmt.Println(err)
		return
	}

	log.Printf("ip net: %s\n", ipNet.IP.String())

	networkIps := networking.GetIPRange(ipNet)

	for _, ip := range networkIps {
		if ip.Equal(localIP) {
			continue
		}

		log.Printf("Pinging %s\n", ip.String())

		pingResult, err := networking.ResolvePing(ip.String())

		if err != nil {
			continue
		}

		log.Printf("Ping to %s took %v\n", ip, pingResult.Duration)
	}
}
