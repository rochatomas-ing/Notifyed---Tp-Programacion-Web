package handlersTP

import (
	"context"
	"net/http"

	db "servidor-NotifyEd/db/sqlc"
	"servidor-NotifyEd/service"

	"github.com/google/uuid"
)

type inscriptionService interface {
	Subscribe(context.Context, string, string) (db.Inscription, error)
	Unsubscribe(context.Context, string, string) error
	GetByEmail(context.Context, string, string) (db.Inscription, error)
	ListBySemester(context.Context, uuid.UUID) ([]db.Inscription, error)
	CountActive(context.Context, uuid.UUID) (int64, error)
	Toggle(context.Context, uuid.UUID) (db.Inscription, error)
	Delete(context.Context, uuid.UUID) error
}

// Cuerpo esperado en POST y DELETE /semesters/{idOrToken}/inscriptions
type inscriptionEmailRequest struct {
	Email string `json:"email"`
}

// RegisterInscriptionRoutes registra la parte del alumno (alta, baja y estado
// sobre /semesters/{idOrToken}/inscriptions) y la del profesor (listado,
// contador, toggle y borrado duro).
func RegisterInscriptionRoutes(mux *http.ServeMux, inscriptions *service.InscriptionService) {
	mux.HandleFunc("/semesters/{idOrToken}/inscriptions", semesterInscriptionsHandler(inscriptions))
	mux.HandleFunc("/semesters/{id}/suscriptos", semesterSubscribersHandler(inscriptions))
	mux.HandleFunc("/inscriptions/{id}", inscriptionHandler(inscriptions))
	mux.HandleFunc("/inscriptions/{id}/toggle", inscriptionToggleHandler(inscriptions))
}

// semesterInscriptionsHandler atiende la ruta compartida del alumno y del
// profesor (decisión #4 del contrato): con UUID lista los inscriptos, con
// sub_token resuelve al alumno por email.
func semesterInscriptionsHandler(inscriptions inscriptionService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reference := r.PathValue("idOrToken")
		switch r.Method {
		case http.MethodGet:
			if id, err := uuid.Parse(reference); err == nil {
				result, err := inscriptions.ListBySemester(r.Context(), id)
				if err != nil {
					HandleServiceError(w, err)
					return
				}
				WriteJSON(w, http.StatusOK, result)
				return
			}
			email := r.URL.Query().Get("email")
			if email == "" {
				HandleError(w, http.StatusBadRequest, "el email es obligatorio para consultar la suscripción")
				return
			}
			result, err := inscriptions.GetByEmail(r.Context(), reference, email)
			if err != nil {
				HandleServiceError(w, err)
				return
			}
			WriteJSON(w, http.StatusOK, result)
		case http.MethodPost:
			var request inscriptionEmailRequest
			if !DecodeJSON(w, r, &request) {
				return
			}
			result, err := inscriptions.Subscribe(r.Context(), reference, request.Email)
			if err != nil {
				HandleServiceError(w, err)
				return
			}
			WriteJSON(w, http.StatusCreated, result)
		case http.MethodDelete:
			var request inscriptionEmailRequest
			if !DecodeJSON(w, r, &request) {
				return
			}
			if err := inscriptions.Unsubscribe(r.Context(), reference, request.Email); err != nil {
				HandleServiceError(w, err)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			HandleError(w, http.StatusMethodNotAllowed, "método no permitido")
		}
	}
}

// semesterSubscribersHandler devuelve el contador de suscriptos activos.
func semesterSubscribersHandler(inscriptions inscriptionService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			HandleError(w, http.StatusMethodNotAllowed, "método no permitido")
			return
		}
		id, ok := ParsePathUUID(w, r, "id")
		if !ok {
			return
		}
		count, err := inscriptions.CountActive(r.Context(), id)
		if err != nil {
			HandleServiceError(w, err)
			return
		}
		WriteJSON(w, http.StatusOK, map[string]int64{"active_subscribers": count})
	}
}

// inscriptionHandler atiende /inscriptions/{id}: borrado duro del panel.
func inscriptionHandler(inscriptions inscriptionService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			HandleError(w, http.StatusMethodNotAllowed, "método no permitido")
			return
		}
		id, ok := ParsePathUUID(w, r, "id")
		if !ok {
			return
		}
		if err := inscriptions.Delete(r.Context(), id); err != nil {
			HandleServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// inscriptionToggleHandler suspende o reactiva una inscripción.
func inscriptionToggleHandler(inscriptions inscriptionService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			HandleError(w, http.StatusMethodNotAllowed, "método no permitido")
			return
		}
		id, ok := ParsePathUUID(w, r, "id")
		if !ok {
			return
		}
		result, err := inscriptions.Toggle(r.Context(), id)
		if err != nil {
			HandleServiceError(w, err)
			return
		}
		WriteJSON(w, http.StatusOK, result)
	}
}
