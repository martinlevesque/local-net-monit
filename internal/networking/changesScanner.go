package networking

import (
	"fmt"
	"log"
	"net"
	"slices"
	"time"
)

type NetworkChangeType string

const (
	NetworkChangeTypeNodeUpdated   NetworkChangeType = "NodeUpdated"
	NetworkChangeTypeNodeDeleted   NetworkChangeType = "NodeDeleted"
	NetworkChangePortUpdated       NetworkChangeType = "PortUpdated"
	NetworkChangePublicNodeUpdated NetworkChangeType = "PublicNodeUpdated"
)

type NetworkChange struct {
	ChangeType  NetworkChangeType
	Description string
	UpdatedNode *Node
	DeletedNode *Node
	PublicNode  *Node // the public/Internet IP node
}

type Node struct {
	IP               string
	LastPingDuration time.Duration
	Ports            []int
}

type NetScanner struct {
	NotifyChannel chan NetworkChange
	NodeStatuses  map[string]*Node
	PublicNode    *Node
}

func (ns *NetScanner) Scan() {
	ns.NodeStatuses = make(map[string]*Node)
	lastFullScanLoop := time.Now()
	lastFullScanLoop = lastFullScanLoop.Add(-5 * time.Minute)

	publicIP, err := ResolverPublicIp()

	if err != nil {
		log.Fatalf("Failed to get public IP: %v", err)
	}

	ns.PublicNode = &Node{
		IP:               publicIP,
		Ports:            []int{},
		LastPingDuration: time.Duration(0),
	}

	for {
		newPublicIP, err := ResolverPublicIp()

		if err != nil {
			log.Println("Failed to get public IP: ", err)
		} else if newPublicIP != ns.PublicNode.IP {
			ns.PublicNode.IP = newPublicIP
			// todo: notify changes
		}

		log.Println("Scanning ports for public IP")
		resultPublicOpen := IsPublicPortOpen(ns.PublicNode.IP, 32400)
		log.Println("Public port 32401 is open: ", resultPublicOpen)

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
					ChangeType:  NetworkChangeTypeNodeDeleted,
					Description: fmt.Sprintf("Node %s deleted", ip.String()),
					UpdatedNode: nil,
					DeletedNode: node,
				}
			}

			continue
		}

		log.Printf("Ping to %s took %v\n", ip, pingResult.Duration)
		var currrentNode *Node = nil

		// Update the node status
		if node, ok := ns.NodeStatuses[ip.String()]; ok {
			node.LastPingDuration = pingResult.Duration
			currrentNode = node
			ns.NodeStatuses[ip.String()] = node

			ns.NotifyChannel <- NetworkChange{
				ChangeType:  NetworkChangeTypeNodeUpdated,
				Description: fmt.Sprintf("Node %s updated", ip.String()),
				UpdatedNode: node,
				DeletedNode: nil,
			}
		} else {
			node = &Node{
				IP:               ip.String(),
				LastPingDuration: pingResult.Duration,
				Ports:            []int{},
			}

			ns.NodeStatuses[ip.String()] = node
			currrentNode = node

			ns.NotifyChannel <- NetworkChange{
				ChangeType:  NetworkChangeTypeNodeUpdated,
				Description: fmt.Sprintf("New node found: %s", ip.String()),
				UpdatedNode: node,
				DeletedNode: nil,
			}
		}

		scanPorts(currrentNode)
	}
}

func scanPorts(node *Node) {
	for port := 1; port <= 65535; port++ {
		if isTCPPortOpen(node.IP, port) {
			// todo notify changes
			log.Printf("Port tcp %d is open on %s\n", port, node.IP)

			if !slices.Contains(node.Ports, port) {
				node.Ports = append(node.Ports, port)
			}
		} else {
			if slices.Contains(node.Ports, port) {
				node.Ports = slices.DeleteFunc(node.Ports, func(i int) bool {
					return i == port
				})
			}
		}
	}
}
