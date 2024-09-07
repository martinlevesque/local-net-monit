package web

import (
	"fmt"
	"github.com/martinlevesque/local-net-monit/internal/networking"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"runtime"
)

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

func BootstrapHttpServer(netScanner *networking.NetScanner) *http.Server {
	port := 8080
	serverAddress := fmt.Sprintf(":%d", port)

	templates := PrepareTemplates()

	log.Println("Starting HTTP server at", serverAddress)
	mux := http.NewServeMux()

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
