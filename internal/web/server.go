package web

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/martinlevesque/local-net-monit/internal/env"
	"github.com/martinlevesque/local-net-monit/internal/networking"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"runtime"
	"sync"
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
		templates[template_name] = template.Must(template.ParseFiles(filepath.Join(templates_dir, template_name)))
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

	// Update the node status
	if node, ok := netScanner.NodeStatuses.Load(verifyReq.IP); ok {
		node := node.(*networking.Node)

		for i, port := range node.Ports {
			if port.PortNumber == verifyReq.Port {
				node.Ports[i].Verified = verifyReq.Verified
				node.Ports[i].Notes = verifyReq.Notes

				portUpdated = true
			}
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

func handleRoot(netScanner *networking.NetScanner, templates map[string]*template.Template, w http.ResponseWriter, _ *http.Request) {
	log.Println("GET /")
	tmpl := templates["index.html"]

	scannerNodeIP := ""

	if netScanner.ScannerNode != nil {
		scannerNodeIP = netScanner.ScannerNode.IP
	}

	nodeStatuses := make(map[string]*networking.Node)

	// todo make a function
	netScanner.NodeStatuses.Range(func(key, value interface{}) bool {
		nodeStatuses[key.(string)] = value.(*networking.Node)
		return true
	})

	data := struct {
		NetScanner    *networking.NetScanner
		NodeStatuses  map[string]*networking.Node
		ScannerNodeIP string
	}{
		NetScanner:    netScanner,
		NodeStatuses:  nodeStatuses,
		ScannerNodeIP: scannerNodeIP,
	}

	err := tmpl.Execute(w, data)

	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("template execution error: %v", err)
	}
}

type HostPortStatus struct {
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	Verified bool   `json:"verified"`
}

func handleStatus(netScanner *networking.NetScanner, w http.ResponseWriter, _ *http.Request) {
	envVarStatusPublicPorts := env.EnvVar("STATUS_PUBLIC_PORTS", "true")
	envVarStatusLocalPorts := env.EnvVar("STATUS_LOCAL_PORTS", "false")

	statuses := make([]HostPortStatus, 0)
	hasUnverifiedPorts := false

	if envVarStatusPublicPorts == "true" {
		for _, port := range netScanner.PublicNode.Ports {
			statuses = append(statuses, HostPortStatus{
				IP:       netScanner.PublicNode.IP,
				Port:     port.PortNumber,
				Verified: port.Verified,
			})

			if !port.Verified {
				hasUnverifiedPorts = true
			}
		}
	}

	if envVarStatusLocalPorts == "true" {
		netScanner.NodeStatuses.Range(func(key, value interface{}) bool {
			node := value.(*networking.Node)

			for _, port := range node.Ports {
				statuses = append(statuses, HostPortStatus{
					IP:       node.IP,
					Port:     port.PortNumber,
					Verified: port.Verified,
				})

				if !port.Verified {
					hasUnverifiedPorts = true
				}
			}

			return true
		})
	}

	w.Header().Set("Content-Type", "application/json")

	if hasUnverifiedPorts {
		w.WriteHeader(http.StatusUnprocessableEntity)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	jsonResponse, err := json.Marshal(statuses)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(jsonResponse)
}

func BootstrapHttpServer(netScanner *networking.NetScanner) *http.Server {
	port := 8080
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
