package main

import (
	"github.com/martinlevesque/local-net-monit/networking"
	"log"
)

func main() {
	networkChannelReader := make(chan networking.NetworkChange)

	networkScanner := networking.NetScanner{
		NotifyChannel: networkChannelReader,
	}

	go networkScanner.Scan()

	for {
		log.Println("Waiting for network changes")

		select {
		case change := <-networkChannelReader:
			log.Println(change.Description)

			if change.UpdatedNode != nil {
				log.Printf("-- Updated node: %s\n", change.UpdatedNode.IP)
			}
		}
	}
}
