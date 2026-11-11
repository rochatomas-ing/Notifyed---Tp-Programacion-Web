package main

import (
	"fmt"
	"net/http"
	"strings"

	db "servidor-NotifyEd/db/sqlc"
	dbconn "servidor-NotifyEd/db_conn"
	"servidor-NotifyEd/middleware"
	"servidor-NotifyEd/views"

	"github.com/google/uuid"
)

func main() {
	conn, err := dbconn.Connect()
	if err != nil {
		fmt.Printf("Error al conectar con la base de datos: %s\n", err)
		return
	}
	defer conn.Close()

	queries := db.New(conn)
	mux := http.NewServeMux()

	// Verificación de disponibilidad de la base de datos
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		if err := conn.PingContext(r.Context()); err != nil {
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	// 1. Handler principal para GET /: obtiene datos con sqlc y renderiza IndexPage
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		courses, err := queries.ListCourses(r.Context())
		if err != nil {
			http.Error(w, "Error al obtener materias de la base de datos", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := views.IndexPage(courses).Render(r.Context(), w); err != nil {
			http.Error(w, "Error al renderizar componente de vista", http.StatusInternalServerError)
		}
	})

	// 2. Manejo de formulario (Estilo clásico): POST /courses usando patrón PRG
	mux.HandleFunc("POST /courses", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Formulario inválido", http.StatusBadRequest)
			return
		}

		name := strings.TrimSpace(r.FormValue("name"))
		if name == "" || len([]rune(name)) > 40 {
			http.Error(w, "El nombre de la materia es obligatorio y debe tener hasta 40 caracteres", http.StatusBadRequest)
			return
		}

		if _, err := queries.CreateCourse(r.Context(), name); err != nil {
			http.Error(w, "Error al guardar la materia en la base de datos", http.StatusInternalServerError)
			return
		}

		// Redirige al usuario de vuelta a la página principal (Post/Redirect/Get)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	// 3. Manejo de eliminación clásico: POST /courses/delete usando patrón PRG
	mux.HandleFunc("POST /courses/delete", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Formulario inválido", http.StatusBadRequest)
			return
		}

		idStr := r.FormValue("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			http.Error(w, "Identificador de materia inválido", http.StatusBadRequest)
			return
		}

		if _, err := queries.DeleteCourse(r.Context(), id); err != nil {
			http.Error(w, "Error al eliminar la materia (puede tener cursadas u otros datos asociados)", http.StatusConflict)
			return
		}

		// Redirige al usuario de vuelta a la página principal
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	port := ":8080"
	fmt.Printf("Servidor escuchando en http://localhost%s\n", port)

	err = http.ListenAndServe(port, middleware.LoggingMiddleware(mux))
	if err != nil {
		fmt.Printf("Error al iniciar el servidor: %s\n", err)
	}
}
