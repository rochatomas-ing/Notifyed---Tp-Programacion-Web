package handlersTP

import (
	"net/http"
	"path/filepath"
)

func HandleIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Metodo no permitido", http.StatusMethodNotAllowed)
		return
	}

	if r.URL.Path == "/" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeFile(w, r, "./static/index.html")
		return
	}

	// Servir archivos estáticos contenidos en ./static/ (styles.css, app.js, etc.)
	cleanedPath := filepath.Clean(r.URL.Path)
	targetFile := filepath.Join("./static", cleanedPath)
	http.ServeFile(w, r, targetFile)
}

