package main

import (
	"fmt"
	"github.com/martinlevesque/local-net-monit/internal/networking"
	"log"
	"net/http"
)

func main() {
	go bootstrapHttpServer()

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

func bootstrapHttpServer() {
	log.Println("Starting HTTP server")
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		log.Println("GET /")
		fmt.Fprint(w, "Hello, World!")
	})

	http.ListenAndServe(":8080", mux)
}
