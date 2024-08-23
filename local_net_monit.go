package main

import (
	"fmt"
	"github.com/martinlevesque/local-net-monit/networking"
	"log"
)

func main() {
	// Get the local IP address
	localIP := networking.LocalIPResolver()

	ipNet, err := networking.FindSubnetForIP(localIP)
	if err != nil {
		fmt.Println(err)
		return
	}

	log.Printf("ip net: %s\n", ipNet.IP.String())

	networking.GetIPRange(ipNet)

	/*
		for _, ip := range ips {
		}
	*/

	host := "10.0.0.237"
	pingResult, err := networking.ResolvePing(host)

	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Ping to %s took %v\n", host, pingResult.Duration)

	// todo https://chatgpt.com/c/20e7fa88-7a13-478d-9852-e75708328a47
}
