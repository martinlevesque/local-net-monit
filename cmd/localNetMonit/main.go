package main

import (
	"fmt"
	"github.com/martinlevesque/local-net-monit/internal/networking"
	"github.com/martinlevesque/local-net-monit/internal/web"
	"log"
	"sync"
	"time"
)

func main() {
	networkChannelReader := make(chan networking.NetworkChange)

	networkScanner := networking.NetScanner{
		NotifyChannel:          networkChannelReader,
		ScannerNode:            nil,
		PublicNode:             nil,
		NodeStatuses:           sync.Map{},
		LastLocalFullScanLoop:  time.Now().Add(-networking.LocalPortsFullCheckInterval()),
		LastLocalScanLoop:      time.Now(),
		LastPublicFullScanLoop: time.Now().Add(-networking.PublicPortsFullCheckInterval()),
		LastPublicScanLoop:     time.Now(),
	}

	err := networkScanner.LoadSnapshot()

	if err != nil {
		log.Printf("No snapshot available: %v\n", err)
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

			err := networkScanner.Snapshot()

			if err != nil {
				log.Printf("Error while taking snapshot: %v\n", err)
			}
		}
	}
}
