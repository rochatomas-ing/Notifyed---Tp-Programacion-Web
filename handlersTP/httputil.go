package handlersTP

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"servidor-NotifyEd/service"
)

func WriteJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func HandleError(w http.ResponseWriter, statusCode int, message string) {
	WriteJSON(w, statusCode, map[string]string{"error": message})
}

func DecodeJSON(w http.ResponseWriter, r *http.Request, destination any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		HandleError(w, http.StatusBadRequest, "el cuerpo debe contener JSON válido")
		return false
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		HandleError(w, http.StatusBadRequest, "el cuerpo debe contener un único objeto JSON")
		return false
	}
	return true
}

func HandleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, sql.ErrNoRows):
		HandleError(w, http.StatusNotFound, "recurso no encontrado")
	case errors.Is(err, service.ErrInvalidInput):
		HandleError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrConflict):
		HandleError(w, http.StatusConflict, "el recurso entra en conflicto con datos existentes o relacionados")
	default:
		log.Printf("Error interno de servicio: %v", err)
		HandleError(w, http.StatusInternalServerError, "error interno del servidor")
	}
}

// ParseUUID extrae el segmento de la ruta que sigue al prefijo (por ejemplo
// "/notices/" en /notices/{id}) y lo parsea como UUID. Si el segmento no es un
// UUID válido devuelve error y el handler decide responder 400.
func ParseUUID(r *http.Request, prefix string) (uuid.UUID, error) {
	value := strings.TrimPrefix(r.URL.Path, prefix)
	return uuid.Parse(value)
}

func ParsePathUUID(w http.ResponseWriter, r *http.Request, key string) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue(key))
	if err != nil {
		HandleError(w, http.StatusBadRequest, "el identificador debe ser un UUID válido")
		return uuid.Nil, false
	}
	return id, true
}
