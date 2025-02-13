package web

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/martinlevesque/local-net-monit/internal/env"
	"github.com/martinlevesque/local-net-monit/internal/networking"
	"html/template"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"regexp"
	"runtime"
	"sync"
	"time"
)

var wsConnections sync.Map

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func broadcastToWebSockets(message string) {
	wsConnections.Range(func(key, value interface{}) bool {
		conn := value.(*websocket.Conn)
		err := conn.WriteMessage(websocket.TextMessage, []byte(message))
		if err != nil {
			log.Printf("Error sending message to WebSocket: %v", err)
			conn.Close()
			wsConnections.Delete(key)
		}
		return true
	})
}

func PrepareTemplates() map[string]*template.Template {
	_, root_path, _, _ := runtime.Caller(0)
	templates_dir := filepath.Join(filepath.Dir(root_path), "../..", "templates")
	templates := make(map[string]*template.Template)

	templates_list := []string{"index.html"}

	for _, template_name := range templates_list {
		templates[template_name] = template.Must(
			template.ParseFiles(filepath.Join(templates_dir, template_name)),
		)
	}

	return templates
}

func handleWebSocket(conn *websocket.Conn) {
	defer conn.Close()

	connId := fmt.Sprintf("%p", conn)
	wsConnections.Store(connId, conn)

	for {
		// Example: Read message from client
		messageType, p, err := conn.ReadMessage()
		log.Printf("Received message: %s", p)
		log.Printf("Message type: %d", messageType)

		if err != nil {
			log.Println("Error reading WebSocket message:", err)
			wsConnections.Delete(connId)
			return
		}
	}
}

type VerifyRequest struct {
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	Verified bool   `json:"verified"`
	Notes    string `json:"notes"`
}

func retrieveOriginIP(r *http.Request) string {
	// Get the IP address of the client
	originIP := r.Header.Get("X-Forwarded-For")

	if originIP == "" {
		originIP, _, _ = net.SplitHostPort(r.RemoteAddr)
	}

	return originIP
}

func originIpAllowed(originIp string) bool {
	ipRegexPattern := env.EnvVar("WEB_ROOT_ALLOWED_ORIGIN_IP_PATTERN", "")

	if ipRegexPattern == "" {
		return true
	}

	ipRegex, err := regexp.Compile(ipRegexPattern)

	if err != nil {
		fmt.Println("Error compiling regex:", err)
		return false
	}

	return ipRegex.MatchString(originIp)
}

func handleVerify(netScanner *networking.NetScanner, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var verifyReq VerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&verifyReq); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		log.Printf("Error decoding JSON: %v", err)
		return
	}

	// Process the data as needed
	log.Printf("Received verification request: IP=%s, Port=%d, Verified=%t, Notes=%s",
		verifyReq.IP, verifyReq.Port, verifyReq.Verified, verifyReq.Notes)

	portUpdated := false

	// Look for the local node statuses
	if node, ok := netScanner.NodeStatuses.Load(verifyReq.IP); ok {
		node := node.(*networking.Node)

		hasUpdatedPort := node.VerifyPort(verifyReq.Port, verifyReq.Verified, verifyReq.Notes)

		netScanner.NotifyChange(networking.NetworkChange{
			ChangeType:  networking.NetworkChangePortUpdated,
			Description: fmt.Sprintf("Node %s detect port %d open", verifyReq.IP, verifyReq.Port),
			UpdatedNode: node,
			DeletedNode: nil,
		})

		if hasUpdatedPort {
			portUpdated = true
		}
	}

	// Public node check
	if netScanner.PublicNode.IP == verifyReq.IP {
		hasUpdatedPort := netScanner.PublicNode.VerifyPort(verifyReq.Port, verifyReq.Verified, verifyReq.Notes)

		if hasUpdatedPort {
			portUpdated = true
		}
	}

	if portUpdated {
		// Send a response back to the client
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success"}`))
	} else {
		http.Error(w, "IP:Port not found", http.StatusNotFound)
	}
}

func handleRoot(netScanner *networking.NetScanner, templates map[string]*template.Template, w http.ResponseWriter, req *http.Request) {
	log.Println("GET /")
	tmpl := templates["index.html"]

	originIp := retrieveOriginIP(req)

	if !originIpAllowed(originIp) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	scannerNodeIP := ""

	if netScanner.ScannerNode != nil {
		scannerNodeIP = netScanner.ScannerNode.IP
	}

	nodeStatuses := netScanner.CopyNodeStatuses()

	data := struct {
		NetScanner             *networking.NetScanner
		NodeStatuses           map[string]*networking.Node
		LastPublicFullScanLoop string
		LastPublicScanLoop     string
		LastLocalFullScanLoop  string
		LastLocalScanLoop      string
		ScannerNodeIP          string
		WebSocketUrl           string
		RecentChanges          []networking.RecentNetworkChange
	}{
		NetScanner:             netScanner,
		NodeStatuses:           nodeStatuses,
		LastPublicFullScanLoop: netScanner.LastPublicFullScanLoop.Format(time.RFC3339),
		LastPublicScanLoop:     netScanner.LastPublicScanLoop.Format(time.RFC3339),
		LastLocalFullScanLoop:  netScanner.LastLocalFullScanLoop.Format(time.RFC3339),
		LastLocalScanLoop:      netScanner.LastLocalScanLoop.Format(time.RFC3339),
		ScannerNodeIP:          scannerNodeIP,
		RecentChanges:          netScanner.RecentChanges,
	}

	err := tmpl.Execute(w, data)

	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("template execution error: %v", err)
	}
}

type ResponseStatus struct {
	Status string `json:"status"`
}

func handleStatus(netScanner *networking.NetScanner, w http.ResponseWriter, _ *http.Request) {
	envVarStatusPublicPorts := env.EnvVar("STATUS_PUBLIC_PORTS", "true")
	envVarStatusLocalPorts := env.EnvVar("STATUS_LOCAL_PORTS", "false")

	hasUnverifiedPorts := false

	if envVarStatusPublicPorts == "true" {
		for _, port := range netScanner.PublicNode.Ports {
			if !port.Verified {
				hasUnverifiedPorts = true
			}
		}
	}

	if envVarStatusLocalPorts == "true" {
		netScanner.NodeStatuses.Range(func(key, value interface{}) bool {
			node := value.(*networking.Node)

			for _, port := range node.Ports {
				if !port.Verified {
					hasUnverifiedPorts = true
				}
			}

			return true
		})
	}

	w.Header().Set("Content-Type", "application/json")

	statusBody := ResponseStatus{
		Status: "OK",
	}

	if hasUnverifiedPorts {
		statusBody.Status = "NOK"
		w.WriteHeader(http.StatusUnprocessableEntity)
	} else {
		w.WriteHeader(http.StatusOK)

		statusBody.Status = "OK"
	}

	jsonResponse, err := json.Marshal(statusBody)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(jsonResponse)
}

func BootstrapHttpServer(netScanner *networking.NetScanner) *http.Server {
	port := env.EnvVarInt("PORT", 8080)
	serverAddress := fmt.Sprintf(":%d", port)

	templates := PrepareTemplates()

	log.Println("Starting HTTP server at", serverAddress)
	mux := http.NewServeMux()
	netScanner.BroadcastChange = broadcastToWebSockets

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		handleRoot(netScanner, templates, w, r)
	})

	// WebSocket handler
	mux.HandleFunc("GET /ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("Error upgrading to WebSocket:", err)
			return
		}
		handleWebSocket(conn)
	})

	mux.HandleFunc("POST /verify", func(w http.ResponseWriter, r *http.Request) {
		handleVerify(netScanner, w, r)
	})

	mux.HandleFunc("GET /status", func(w http.ResponseWriter, r *http.Request) {
		handleStatus(netScanner, w, r)
	})

	server := &http.Server{
		Addr:    serverAddress,
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server ListenAndServe: %v", err)
		}
	}()

	return server
}
