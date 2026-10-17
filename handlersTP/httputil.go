package handlersTP

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

func WriteJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func HandleError(w http.ResponseWriter, statusCode int, message string) {
	WriteJSON(w, statusCode, map[string]string{"error": message})
}

// ParseUUID extrae el segmento de la ruta que sigue al prefijo (por ejemplo
// "/notices/" en /notices/{id}) y lo parsea como UUID. Si el segmento no es un
// UUID válido devuelve error y el handler decide responder 400.
func ParseUUID(r *http.Request, prefix string) (uuid.UUID, error) {
	value := strings.TrimPrefix(r.URL.Path, prefix)
	return uuid.Parse(value)
}
