package main

import (
	"fmt"
	"github.com/martinlevesque/local-net-monit/internal/networking"
	"github.com/martinlevesque/local-net-monit/internal/web"
	"log"
)

func main() {
	networkChannelReader := make(chan networking.NetworkChange)

	networkScanner := networking.NetScanner{
		NotifyChannel: networkChannelReader,
		ScannerNode:   nil,
	}

	go web.BootstrapHttpServer(&networkScanner)

	go networkScanner.Scan()

	for {
		log.Println("Waiting for network changes")

		select {
		case change := <-networkChannelReader:
			log.Println(change.Description)
			log.Printf("-- Updated node: %v\n", change)
			stringifiedChange := fmt.Sprintf("%v", change)
			networkScanner.BroadcastChange(stringifiedChange)
			networkScanner.Snapshot()
		}
	}
}
