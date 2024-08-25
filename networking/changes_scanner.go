package networking

import (
	"fmt"
	"log"
	"time"
)

type NetworkChange struct {
	Description string
	UpdatedNode *Node
	DeletedNode *Node
}

type Node struct {
	IP               string
	LastPingDuration time.Duration
}

type NetScanner struct {
	NotifyChannel chan NetworkChange
	NodeStatuses  map[string]Node
}

func (ns *NetScanner) Scan() {
	ns.NodeStatuses = make(map[string]Node)

	for {
		ns.scanLoop()
		time.Sleep(2 * time.Second)
	}
}

func (ns *NetScanner) scanLoop() {
	// Get the local IP address
	localIP := LocalIPResolver()

	log.Printf("Local IP: %s\n", localIP.String())

	ipNet, err := FindSubnetForIP(localIP)

	if err != nil {
		fmt.Println(err)
		return
	}

	log.Printf("ip net: %s\n", ipNet.IP.String())

	networkIps := GetIPRange(ipNet)

	for _, ip := range networkIps {
		if ip.Equal(localIP) {
			continue
		}

		log.Printf("Pinging %s\n", ip.String())

		pingResult, err := ResolvePing(ip.String())

		if err != nil {

			if node, ok := ns.NodeStatuses[ip.String()]; ok {
				delete(ns.NodeStatuses, ip.String())

				ns.NotifyChannel <- NetworkChange{
					Description: fmt.Sprintf("Node %s deleted", ip.String()),
					UpdatedNode: nil,
					DeletedNode: &node,
				}
			}

			continue
		}

		log.Printf("Ping to %s took %v\n", ip, pingResult.Duration)

		// Update the node status
		if node, ok := ns.NodeStatuses[ip.String()]; ok {
			node.LastPingDuration = pingResult.Duration
			ns.NodeStatuses[ip.String()] = node

			ns.NotifyChannel <- NetworkChange{
				Description: fmt.Sprintf("Node %s updated", ip.String()),
				UpdatedNode: &node,
				DeletedNode: nil,
			}
		} else {
			node = Node{
				IP:               ip.String(),
				LastPingDuration: pingResult.Duration,
			}

			ns.NodeStatuses[ip.String()] = node

			ns.NotifyChannel <- NetworkChange{
				Description: fmt.Sprintf("New node found: %s", ip.String()),
				UpdatedNode: &node,
				DeletedNode: nil,
			}
		}
	}
}
