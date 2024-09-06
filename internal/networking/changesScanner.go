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

type Port struct {
	PortNumber int
	Verified   bool
}

type Node struct {
	IP               string
	LastPingDuration time.Duration
	Ports            []Port
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
		Ports:            []Port{},
		LastPingDuration: time.Duration(0),
	}

	for {
		ns.scanPublicNodePorts()

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

func portExistsInList(port int, ports []Port) bool {
	return slices.ContainsFunc(ports, func(p Port) bool {
		return p.PortNumber == port
	})
}

func (ns *NetScanner) scanPublicNodePorts() {
	newPublicIP, err := ResolverPublicIp()

	if err != nil {
		log.Println("Failed to get public IP: ", err)
	} else if newPublicIP != ns.PublicNode.IP {
		ns.PublicNode.IP = newPublicIP
	}

	for port := 1; port <= 65535; port++ {
		resultPublicOpen := IsPublicPortOpen(ns.PublicNode.IP, port)

		if resultPublicOpen {
			log.Printf("Port tcp %d is open on %s\n", port, ns.PublicNode.IP)

			if !portExistsInList(port, ns.PublicNode.Ports) {
				ns.PublicNode.Ports = append(
					ns.PublicNode.Ports,
					Port{PortNumber: port, Verified: false},
				)
			}
		} else {
			if portExistsInList(port, ns.PublicNode.Ports) {
				ns.PublicNode.Ports = slices.DeleteFunc(ns.PublicNode.Ports, func(p Port) bool {
					return p.PortNumber == port
				})
			}
		}

		time.Sleep(20 * time.Millisecond)
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
				Ports:            []Port{},
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

			if !portExistsInList(port, node.Ports) {
				node.Ports = append(
					node.Ports,
					Port{PortNumber: port, Verified: false},
				)
			}
		} else {
			if portExistsInList(port, node.Ports) {
				node.Ports = slices.DeleteFunc(node.Ports, func(p Port) bool {
					return p.PortNumber == port
				})
			}
		}
	}
}
