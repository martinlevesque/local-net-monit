package networking

import (
	"fmt"
	"github.com/martinlevesque/local-net-monit/internal/env"
	"log"
	"net"
	"slices"
	"sync"
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
	Notes      string
}

type Node struct {
	IP               string
	LastPingDuration time.Duration
	Ports            []Port
}

type NetScanner struct {
	NotifyChannel   chan NetworkChange
	NodeStatuses    sync.Map
	PublicNode      *Node
	ScannerNode     *Node
	BroadcastChange func(string)
}

func (ns *NetScanner) Scan() {
	ns.NodeStatuses = sync.Map{}
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
		if env.EnvVar("MONITOR_PUBLIC_PORTS", "true") == "true" {
			ns.scanPublicNodePorts()
		}

		if env.EnvVar("MONITOR_LOCAL_PORTS", "true") == "true" {
			log.Println("Scanning local node ports")
			ns.scanLocalNodePorts(lastFullScanLoop)
		}

		time.Sleep(10 * time.Second)
	}
}

func (ns *NetScanner) scanLocalNodePorts(lastFullScanLoop time.Time) {
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

	ns.NodeStatuses.Range(func(_, untypedNode interface{}) bool {
		node := untypedNode.(*Node)
		ip := net.ParseIP(node.IP)

		if ip != nil { // Make sure the IP is valid
			ipList = append(ipList, ip)
		}

		return true
	})

	return ipList
}

func (ns *NetScanner) scanLoop(localIP net.IP, networkIps []net.IP) {
	ns.fullNetworkPings(localIP, networkIps)
	var wg sync.WaitGroup

	ns.NodeStatuses.Range(func(_, untypedNode any) bool {
		wg.Add(1)
		if node, ok := untypedNode.(*Node); ok {
			go func(node *Node) {
				defer wg.Done()
				ns.scanPorts(node)
			}(node)
		}

		return true
	})

	wg.Wait()
}

func (ns *NetScanner) fullNetworkPings(localIP net.IP, networkIps []net.IP) {
	var wg sync.WaitGroup

	// Ping each IP in parallel
	for _, ip := range networkIps {
		wg.Add(1)
		go func(ip net.IP) {
			defer wg.Done()
			ns.pingIp(localIP, ip)
		}(ip)
	}

	// Wait for all pings to complete
	wg.Wait()
}

func (ns *NetScanner) pingIp(localIP net.IP, ip net.IP) {
	log.Printf("Pinging %s\n", ip.String())

	pingResult, err := ResolvePing(ip.String())

	if err != nil {

		if node, ok := ns.NodeStatuses.Load(ip.String()); ok {
			ns.NodeStatuses.Delete(ip.String())

			ns.NotifyChannel <- NetworkChange{
				ChangeType:  NetworkChangeTypeNodeDeleted,
				Description: fmt.Sprintf("Node %s deleted", ip.String()),
				UpdatedNode: nil,
				DeletedNode: node.(*Node),
			}
		}

		return
	}

	var currrentNode *Node = nil

	// Update the node status
	if untypedNode, ok := ns.NodeStatuses.Load(ip.String()); ok {
		node := untypedNode.(*Node)
		node.LastPingDuration = pingResult.Duration
		currrentNode = node
		ns.NodeStatuses.Store(ip.String(), node)

		ns.NotifyChannel <- NetworkChange{
			ChangeType:  NetworkChangeTypeNodeUpdated,
			Description: fmt.Sprintf("Node %s updated", ip.String()),
			UpdatedNode: node,
			DeletedNode: nil,
		}
	} else {
		node := &Node{
			IP:               ip.String(),
			LastPingDuration: pingResult.Duration,
			Ports:            []Port{},
		}

		ns.NodeStatuses.Store(ip.String(), node)
		currrentNode = node

		ns.NotifyChannel <- NetworkChange{
			ChangeType:  NetworkChangeTypeNodeUpdated,
			Description: fmt.Sprintf("New node found: %s", ip.String()),
			UpdatedNode: node,
			DeletedNode: nil,
		}
	}

	if localIP.Equal(ip) {
		ns.ScannerNode = currrentNode
	}
}

func (ns *NetScanner) scanPorts(node *Node) {
	for port := 1; port <= 65535; port++ {
		if isTCPPortOpen(node.IP, port) {
			// todo notify changes
			log.Printf("Port tcp %d is open on %s\n", port, node.IP)

			if !portExistsInList(port, node.Ports) {
				node.Ports = append(
					node.Ports,
					Port{PortNumber: port, Verified: false, Notes: ""},
				)
				ns.NotifyChannel <- NetworkChange{
					ChangeType:  NetworkChangePortUpdated,
					Description: fmt.Sprintf("Node %s updated, port %d added", node.IP, port),
					UpdatedNode: node,
					DeletedNode: nil,
				}
			}
		} else {
			if portExistsInList(port, node.Ports) {
				node.Ports = slices.DeleteFunc(node.Ports, func(p Port) bool {
					return p.PortNumber == port
				})

				ns.NotifyChannel <- NetworkChange{
					ChangeType:  NetworkChangePortUpdated,
					Description: fmt.Sprintf("Node %s updated, port %d removed", node.IP, port),
					UpdatedNode: node,
					DeletedNode: nil,
				}
			}
		}
	}
}
