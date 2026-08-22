package main

import (
	"fmt"
	"net/http"
	"servidor-NotifyEd/handlersTP"
)

func main() {

	http.HandleFunc("/", handlersTP.HandleIndex)

	port := "0.0.0.0:8080"
	fmt.Printf("Servidor escuchando en http://localhost%s\n", port)

	err := http.ListenAndServe(port, nil)
	if err != nil {
		fmt.Printf("Error al iniciar el servidor: %s\n", err)
	}

}
