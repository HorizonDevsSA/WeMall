package main

import (
	"embed"
	"html/template"
	"io/fs"
	"log"
	"net/http"

	"github.com/wemall/services/documentation-service/internal/content"
	"github.com/wemall/services/documentation-service/internal/handlers"
)

//go:embed templates/*.html
var templatesFS embed.FS

//go:embed static/*
var staticFS embed.FS

func main() {
	// 1. Parse Embedded HTML Templates
	tmpl, err := template.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		log.Fatalf("failed to parse templates: %v", err)
	}

	// 2. Fetch seed API reference category catalog
	apiCategories := content.GetAPICategories()

	// 3. Initialize route handlers
	h := handlers.NewHandler(tmpl, apiCategories)

	// 4. Create sub-filesystem for static asset mapping
	staticSubFS, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatalf("failed to create static sub-filesystem: %v", err)
	}

	// 5. Setup multiplexer routes
	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticSubFS))))
	mux.HandleFunc("/flows/", h.APIDetail)
	mux.HandleFunc("/apis/", h.APIDetail)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		h.Dashboard(w, r)
	})

	// 6. Start Web Server
	port := "8085"
	log.Printf("Starting WeMall API Documentation Service on port %s...", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
