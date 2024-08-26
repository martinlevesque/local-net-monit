package networking

import (
	"fmt"
	"log"
	"net"
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
	lastFullScanLoop := time.Now()
	lastFullScanLoop = lastFullScanLoop.Add(-5 * time.Minute)

	for {
		// Get the local IP address
		localIP := LocalIPResolver()

		ipNet, err := FindSubnetForIP(localIP)

		if err != nil {
			fmt.Println(err)
			return
		}

		networkIps := GetIPRange(ipNet)

		if time.Since(lastFullScanLoop) > 5*time.Minute {
			log.Println("Full scan loop")
			ns.scanLoop(localIP, networkIps)
			lastFullScanLoop = time.Now()
		} else {
			log.Println("Partial scan loop")
			ns.scanLoop(localIP, ns.currentNetworkIps())
		}

		time.Sleep(10 * time.Second)
	}
}

func (ns *NetScanner) currentNetworkIps() []net.IP {
	var ipList []net.IP

	for _, node := range ns.NodeStatuses {
		ip := net.ParseIP(node.IP)

		if ip != nil { // Make sure the IP is valid
			ipList = append(ipList, ip)
		}
	}

	return ipList
}

func (ns *NetScanner) scanLoop(localIP net.IP, networkIps []net.IP) {

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
