package web

import (
	"fmt"
	"github.com/gorilla/websocket"
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

func BootstrapHttpServer(netScanner *networking.NetScanner) *http.Server {
	port := 8080
	serverAddress := fmt.Sprintf(":%d", port)

	templates := PrepareTemplates()

	log.Println("Starting HTTP server at", serverAddress)
	mux := http.NewServeMux()
	netScanner.BroadcastChange = broadcastToWebSockets

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		log.Println("GET /")
		tmpl := templates["index.html"]

		scannerNodeIP := ""

		if netScanner.ScannerNode != nil {
			scannerNodeIP = netScanner.ScannerNode.IP
		}

		data := struct {
			NetScanner    *networking.NetScanner
			ScannerNodeIP string
		}{
			NetScanner:    netScanner,
			ScannerNodeIP: scannerNodeIP,
		}

		err := tmpl.Execute(w, data)

		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			log.Printf("template execution error: %v", err)
		}
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
