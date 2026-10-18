package middleware

import (
	"log"
	"net/http"
)

// LoggingMiddleware loguea el método y la ruta de cada petición antes de
// delegar en el handler siguiente. Se envuelve el mux completo desde main.go:
// http.ListenAndServe(port, middleware.LoggingMiddleware(mux))
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Request: %s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
