package web

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
)

func prepareTemplates() map[string]*template.Template {
	tmpl := make(map[string]*template.Template)

	tmpl["templates/index.html"] = template.Must(template.ParseFiles("templates/index.html"))

	return tmpl
}

func BootstrapHttpServer() {
	port := 8080
	serverAddress := fmt.Sprintf(":%d", port)

	templates := prepareTemplates()

	log.Println("Starting HTTP server at", serverAddress)
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		queryParamsP := r.URL.Query().Get("p")
		log.Println("GET /")
		tmpl := templates["templates/index.html"]
		//fmt.Fprint(w, "Hello, World!")
		data := struct {
			Items []string
		}{
			Items: []string{"item1", "item2", "item3", queryParamsP},
		}

		err := tmpl.Execute(w, data)

		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	})

	http.ListenAndServe(serverAddress, mux)
}
