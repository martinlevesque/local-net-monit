package networking

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/martinlevesque/local-net-monit/internal/env"
)

type NetworkChangeType string

const (
	NetworkChangeTypeNodeUpdated                NetworkChangeType = "NodeUpdated"
	NetworkChangeTypeNodeDeleted                NetworkChangeType = "NodeDeleted"
	NetworkChangePortUpdated                    NetworkChangeType = "PortUpdated"
	NetworkChangeIpInfoUpdated                  NetworkChangeType = "PortUpdated"
	NetworkChangePublicNodeUpdated              NetworkChangeType = "PublicNodeUpdated"
	NetworkChangeTypeFullLocalScanCompleted     NetworkChangeType = "FullLocalScanCompleted"
	NetworkChangeTypePartialLocalScanCompleted  NetworkChangeType = "PartialLocalScanCompleted"
	NetworkChangeTypeFullPublicScanCompleted    NetworkChangeType = "FullPublicScanCompleted"
	NetworkChangeTypePartialPublicScanCompleted NetworkChangeType = "PartialPublicScanCompleted"
	NetworkChangeTypeCheckPublicPortsInterval   NetworkChangeType = "CheckPublicPortsInterval"
)

type NetworkChange struct {
	Timestamp   time.Time
	ChangeType  NetworkChangeType
	Description string
	UpdatedNode *Node
	DeletedNode *Node
	PublicNode  *Node // the public/Internet IP node
}

type RecentNetworkChange struct {
	Timestamp   string
	Description string
}

type Port struct {
	PortNumber int
	Verified   bool
	Notes      string
}

type Node struct {
	IP               string
	Name             string
	Verified         bool
	LastPingDuration time.Duration
	Ports            []Port
	Online           bool
	LastOnlineAt     string
}

type NetScanner struct {
	NotifyChannel          chan NetworkChange
	NodeStatuses           sync.Map
	PublicNode             *Node
	ScannerNode            *Node
	BroadcastChange        func(string)
	LastLocalFullScanLoop  time.Time
	LastLocalScanLoop      time.Time
	LastPublicFullScanLoop time.Time
	LastPublicScanLoop     time.Time
	RecentChanges          []RecentNetworkChange
	NotifyChangesToChannel bool
}

func (ns *NetScanner) NotifyChange(change NetworkChange) {
	change.Timestamp = time.Now()

	if ns.NotifyChangesToChannel {
		ns.NotifyChannel <- change
	}
}

func (ns *NetScanner) AppendRecentChange(change RecentNetworkChange) {
	ns.RecentChanges = append(ns.RecentChanges, change)

	if len(ns.RecentChanges) > 10 {
		ns.RecentChanges = ns.RecentChanges[1:]
	}
}

func (ns *NetScanner) CopyNodeStatuses() map[string]*Node {
	nodeStatuses := make(map[string]*Node)

	ns.NodeStatuses.Range(func(key, value interface{}) bool {
		nodeStatuses[key.(string)] = value.(*Node)
		return true
	})

	return nodeStatuses
}

func (ns *NetScanner) Json() (string, error) {
	data := map[string]interface{}{
		"NodeStatuses":           ns.CopyNodeStatuses(),
		"PublicNode":             ns.PublicNode,
		"ScannerNode":            ns.ScannerNode,
		"RecentChanges":          ns.RecentChanges,
		"LastLocalFullScanLoop":  ns.LastLocalFullScanLoop,
		"LastLocalScanLoop":      ns.LastLocalScanLoop,
		"LastPublicFullScanLoop": ns.LastPublicFullScanLoop,
		"LastPublicScanLoop":     ns.LastPublicScanLoop,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")

	if err != nil {
		return "", err
	}

	return string(jsonData), nil

}

func (ns *NetScanner) Snapshot() error {
	storagePath := env.EnvVar("SNAPSHOT_STORAGE_PATH", "localNetMonit.json")

	content, err := ns.Json()

	if err != nil {
		return err
	}

	err = os.WriteFile(storagePath, []byte(content), 0644)

	if err != nil {
		return err
	}

	return nil
}

func (ns *NetScanner) LoadSnapshot() error {
	storagePath := env.EnvVar("SNAPSHOT_STORAGE_PATH", "localNetMonit.json")

	content, err := os.ReadFile(storagePath)

	if err != nil {
		return err
	}

	var data map[string]interface{}

	err = json.Unmarshal(content, &data)

	if err != nil {
		return err
	}

	if nodeStatuses, ok := data["NodeStatuses"].(map[string]interface{}); ok {
		for key, value := range nodeStatuses {
			nodeData := value.(map[string]interface{})

			node := loadNode(nodeData)

			ns.NodeStatuses.Store(key, node)
		}
	}

	if publicNodeData, ok := data["PublicNode"].(map[string]interface{}); ok {
		ns.PublicNode = loadNode(publicNodeData)
	}

	readTimeInto(data, "LastPublicFullScanLoop", &ns.LastPublicFullScanLoop)
	readTimeInto(data, "LastPublicScanLoop", &ns.LastPublicScanLoop)
	readTimeInto(data, "LastLocalFullScanLoop", &ns.LastLocalFullScanLoop)
	readTimeInto(data, "LastLocalScanLoop", &ns.LastLocalScanLoop)

	// load recent RecentChanges

	if recentChangesData, ok := data["RecentChanges"].([]interface{}); ok {
		for _, changeData := range recentChangesData {
			change := changeData.(map[string]interface{})
			timestamp, err := time.Parse(time.RFC3339, change["Timestamp"].(string))

			if err != nil {
				continue
			}

			ns.RecentChanges = append(ns.RecentChanges, RecentNetworkChange{
				Timestamp:   timestamp.Format(time.RFC3339),
				Description: change["Description"].(string),
			})
		}
	}

	return nil
}

func readTimeInto(data map[string]interface{}, key string, target *time.Time) {
	if value, ok := data[key].(string); ok {
		timeValue, err := time.Parse(time.RFC3339, value)

		if err == nil {
			*target = timeValue
		}
	}
}

func loadNode(data map[string]interface{}) *Node {
	portsData := data["Ports"].([]interface{})
	ports := make([]Port, len(portsData))

	for i, portData := range portsData {
		port := portData.(map[string]interface{})
		ports[i] = Port{
			PortNumber: int(port["PortNumber"].(float64)),
			Verified:   port["Verified"].(bool),
			Notes:      port["Notes"].(string),
		}

	}

	var lastOnlineAtString string
	if lastOnlineAt, ok := data["LastOnlineAt"]; ok && lastOnlineAt != nil {
		if str, ok := lastOnlineAt.(string); ok {
			lastOnlineAtString = str
		}
	}

	name, ok := data["Name"].(string)
	if !ok {
		name = "" // default if not present or not a string
	}

	node := &Node{
		IP:               data["IP"].(string),
		Name:             name,
		LastPingDuration: time.Duration(data["LastPingDuration"].(float64)),
		Ports:            ports,
		Verified:         data["Verified"].(bool),
		Online:           data["Online"].(bool),
		LastOnlineAt:     lastOnlineAtString,
	}

	return node
}

func (node *Node) VerifyPort(port int, verified bool, notes string) bool {
	portUpdated := false

	for i, currentPort := range node.Ports {
		if currentPort.PortNumber == port {
			node.Ports[i].Verified = verified
			node.Ports[i].Notes = notes

			portUpdated = true
		}
	}

	return portUpdated
}

func (node *Node) VerifyIp(name string, verified bool) bool {
	updated := false

	node.Name = name
	node.Verified = verified

	return updated
}

func LocalPortsFullCheckInterval() time.Duration {
	return time.Duration(env.EnvVarInt("LOCAL_PORTS_FULL_CHECK_INTERVAL_MINUTES", 120)) * time.Minute
}

func PublicPortsFullCheckInterval() time.Duration {
	return time.Duration(env.EnvVarInt("PUBLIC_PORTS_FULL_CHECK_INTERVAL_MINUTES", 120)) * time.Minute
}

func NodeUptimeTimeoutInterval() time.Duration {
	return time.Duration(env.EnvVarInt("NODE_UPTIME_TIMEOUT_HOURS", 128)) * time.Hour
}

func (ns *NetScanner) Scan() {
	for {
		log.Println("Scanning loop started")

		ns.VerifyNodeUptimeTimeouts()

		publicIP, err := ResolverPublicIp()

		if err != nil {
			log.Fatalf("Failed to get public IP: %v", err)
		}

		log.Printf("Public IP: %s\n", publicIP)

		if ns.PublicNode == nil || ns.PublicNode.IP != publicIP {
			log.Printf("Public IP changed to %s\n", publicIP)

			ns.PublicNode = &Node{
				IP:               publicIP,
				Name:             "",
				Ports:            []Port{},
				LastPingDuration: time.Duration(0),
			}
		}

		if env.EnvVar("MONITOR_PUBLIC_PORTS", "true") == "true" {
			ns.scanPublicNodePorts()
		}

		if env.EnvVar("MONITOR_LOCAL_PORTS", "true") == "true" {
			ns.scanLocalNodePorts()
		}

		log.Println("Scanning loop ended")
		time.Sleep(10 * time.Second)
	}
}

func (ns *NetScanner) VerifyNodeUptimeTimeouts() {
	ns.NodeStatuses.Range(func(key, untypedNode interface{}) bool {
		node := untypedNode.(*Node)

		if node.LastOnlineAt == "" {
			return true
		}

		lastOnlineTime, err := time.Parse(time.RFC3339, node.LastOnlineAt)
		if err != nil {
			log.Printf("Error parsing LastOnlineAt for node %s: %v", node.IP, err)
			return true
		}

		if time.Since(lastOnlineTime) > NodeUptimeTimeoutInterval() {
			ns.NodeStatuses.Delete(key.(string))
		}

		return true
	})
}

func (ns *NetScanner) scanLocalNodePorts() {
	// Get the local IP address
	localIP := LocalIPResolver()

	ipNet, err := FindSubnetForIP(localIP)

	if err != nil {
		fmt.Println(err)
		return
	}

	networkIps := GetIPRange(ipNet)

	var fullScan bool

	if time.Since(ns.LastLocalFullScanLoop) > LocalPortsFullCheckInterval() {
		log.Println("Full scan loop")
		ns.scanLoop(localIP, networkIps)

		fullScan = true
	} else {
		log.Println("Partial scan loop")
		ns.scanLoop(localIP, ns.currentNetworkIps())

		fullScan = false
	}

	if fullScan {
		ns.LastLocalFullScanLoop = time.Now()

		ns.NotifyChange(NetworkChange{
			ChangeType:  NetworkChangeTypeFullLocalScanCompleted,
			Description: fmt.Sprintf("Full local scan completed"),
		})
	} else {
		ns.LastLocalScanLoop = time.Now()

		ns.NotifyChange(NetworkChange{
			ChangeType:  NetworkChangeTypePartialLocalScanCompleted,
			Description: fmt.Sprintf("Partial local scan completed"),
		})
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

	// list of ports to check
	portsToCheck := []int{}
	portsCheckBatch := []int{}
	NB_PORTS_TO_CHECK_PER_BATCH := env.EnvVarInt("NB_PUBLIC_PORTS_TO_CHECK_PER_BATCH", 20)

	var fullScan bool

	if time.Since(ns.LastPublicFullScanLoop) > PublicPortsFullCheckInterval() {
		for port := 1; port <= 65535; port++ {
			portsToCheck = append(portsToCheck, port)
		}

		fullScan = true
	} else {
		// only check the ports that are already known
		for _, port := range ns.PublicNode.Ports {
			portsToCheck = append(portsToCheck, port.PortNumber)
		}

		fullScan = false
	}

	for _, port := range portsToCheck {

		portsCheckBatch = append(portsCheckBatch, port)

		if len(portsCheckBatch) >= NB_PORTS_TO_CHECK_PER_BATCH {
			ns.checkPublicNodePorts(portsCheckBatch)

			ns.NotifyChange(NetworkChange{
				ChangeType: NetworkChangeTypeCheckPublicPortsInterval,
				Description: fmt.Sprintf("Checking public ports %d to %d",
					portsCheckBatch[0],
					portsCheckBatch[len(portsCheckBatch)-1],
				),
			})

			portsCheckBatch = []int{}
		}

		time.Sleep(20 * time.Millisecond)
	}

	if len(portsCheckBatch) > 0 {
		// there might some remaining ports to check
		ns.checkPublicNodePorts(portsCheckBatch)
	}

	if fullScan {
		ns.LastPublicFullScanLoop = time.Now()

		ns.NotifyChange(NetworkChange{
			ChangeType:  NetworkChangeTypeFullPublicScanCompleted,
			Description: fmt.Sprintf("Full public scan completed"),
		})
	} else {
		ns.LastPublicScanLoop = time.Now()

		ns.NotifyChange(NetworkChange{
			ChangeType:  NetworkChangeTypePartialPublicScanCompleted,
			Description: fmt.Sprintf("Partial public scan completed"),
		})
	}
}

func (ns *NetScanner) checkPublicNodePorts(ports []int) {
	// run the check in parallel

	var wg sync.WaitGroup
	resultMap := make(map[int]bool)
	var mutex sync.Mutex

	for _, port := range ports {
		wg.Add(1)
		go func(port int) {
			defer wg.Done()
			resultPublicOpen, err := IsPublicPortOpen(ns.PublicNode.IP, port)

			if err != nil {
				log.Printf("Failed to check port %d: %v\n", port, err)
				return
			}

			mutex.Lock()
			resultMap[port] = resultPublicOpen
			mutex.Unlock()

		}(port)
	}

	wg.Wait()

	for port, resultPublicOpen := range resultMap {
		if resultPublicOpen {
			log.Printf("Port tcp %d is open on %s\n", port, ns.PublicNode.IP)

			if !portExistsInList(port, ns.PublicNode.Ports) {
				ns.PublicNode.Ports = append(
					ns.PublicNode.Ports,
					Port{PortNumber: port, Verified: false},
				)

				ns.NotifyChange(NetworkChange{
					ChangeType:  NetworkChangeTypeNodeUpdated,
					Description: fmt.Sprintf("Node %s detect port %d open", ns.PublicNode.IP, port),
					UpdatedNode: ns.PublicNode,
					DeletedNode: nil,
				})
			}
		} else {
			if portExistsInList(port, ns.PublicNode.Ports) {
				ns.PublicNode.Ports = slices.DeleteFunc(ns.PublicNode.Ports, func(p Port) bool {
					return p.PortNumber == port
				})

				ns.NotifyChange(NetworkChange{
					ChangeType:  NetworkChangeTypeNodeUpdated,
					Description: fmt.Sprintf("Node %s detect port %d closed", ns.PublicNode.IP, port),
					UpdatedNode: ns.PublicNode,
					DeletedNode: nil,
				})
			}
		}
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
	now := time.Now()
	nowString := now.Format(time.RFC3339)

	pingResult, err := ResolvePing(ip.String())

	if err != nil {

		if untypedNode, ok := ns.NodeStatuses.Load(ip.String()); ok {
			node := untypedNode.(*Node)

			node.Online = false // Set to false when ping fails

			ns.NotifyChange(NetworkChange{
				ChangeType:  NetworkChangeTypeNodeDeleted,
				Description: fmt.Sprintf("Node %s deleted", ip.String()),
				UpdatedNode: nil,
				DeletedNode: node,
			})
		}

		return
	}

	var currrentNode *Node = nil

	// Update the node status
	if untypedNode, ok := ns.NodeStatuses.Load(ip.String()); ok {
		node := untypedNode.(*Node)
		node.LastPingDuration = pingResult.Duration
		node.Online = true
		node.LastOnlineAt = nowString
		currrentNode = node
		ns.NodeStatuses.Store(ip.String(), node)

		ns.NotifyChange(NetworkChange{
			ChangeType:  NetworkChangeTypeNodeUpdated,
			Description: fmt.Sprintf("Node %s updated", ip.String()),
			UpdatedNode: node,
			DeletedNode: nil,
		})
	} else {
		node := &Node{
			IP:               ip.String(),
			Name:             "",
			LastPingDuration: pingResult.Duration,
			Ports:            []Port{},
			Online:           true,
			LastOnlineAt:     nowString,
		}

		ns.NodeStatuses.Store(ip.String(), node)
		currrentNode = node

		ns.NotifyChange(NetworkChange{
			ChangeType:  NetworkChangeTypeNodeUpdated,
			Description: fmt.Sprintf("New node found: %s", ip.String()),
			UpdatedNode: node,
			DeletedNode: nil,
		})
	}

	if localIP.Equal(ip) {
		ns.ScannerNode = currrentNode
	}
}

func (ns *NetScanner) scanPorts(node *Node) {
	for port := 1; port <= 65535; port++ {
		if isTCPPortOpen(node.IP, port) {
			log.Printf("Port tcp %d is open on %s\n", port, node.IP)

			if !portExistsInList(port, node.Ports) {
				node.Ports = append(
					node.Ports,
					Port{PortNumber: port, Verified: false, Notes: ""},
				)
				ns.NotifyChange(NetworkChange{
					ChangeType:  NetworkChangePortUpdated,
					Description: fmt.Sprintf("Node %s updated, port %d added", node.IP, port),
					UpdatedNode: node,
					DeletedNode: nil,
				})
			}
		} else {
			if portExistsInList(port, node.Ports) {
				node.Ports = slices.DeleteFunc(node.Ports, func(p Port) bool {
					return p.PortNumber == port
				})

				ns.NotifyChange(NetworkChange{
					ChangeType:  NetworkChangePortUpdated,
					Description: fmt.Sprintf("Node %s updated, port %d removed", node.IP, port),
					UpdatedNode: node,
					DeletedNode: nil,
				})
			}
		}
	}
}
