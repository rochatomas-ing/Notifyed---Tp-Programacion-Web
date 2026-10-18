package main

import (
	"fmt"
	"net/http"
	db "servidor-NotifyEd/db/sqlc"
	dbconn "servidor-NotifyEd/db_conn"
	"servidor-NotifyEd/handlersTP"
	"servidor-NotifyEd/service"
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
	courses := service.NewCourseService(queries)
	semesters := service.NewSemesterService(queries)
	handlersTP.RegisterCourseRoutes(mux, courses, semesters)
	handlersTP.RegisterSemesterRoutes(mux, semesters)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		if err := conn.PingContext(r.Context()); err != nil {
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/", handlersTP.HandleIndex)

	port := ":8080"
	fmt.Printf("Servidor escuchando en http://localhost%s\n", port)

	err = http.ListenAndServe(port, mux)
	if err != nil {
		fmt.Printf("Error al iniciar el servidor: %s\n", err)
	}
}
