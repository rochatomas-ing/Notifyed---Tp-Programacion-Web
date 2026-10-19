package handlersTP

import (
	"context"
	"net/http"

	db "servidor-NotifyEd/db/sqlc"
	"servidor-NotifyEd/service"

	"github.com/google/uuid"
)

type noticeService interface {
	Create(context.Context, db.CreateNoticeParams) (db.Notice, error)
	Get(context.Context, uuid.UUID) (db.Notice, error)
	List(context.Context) ([]db.Notice, error)
	ListBySemester(context.Context, uuid.UUID) ([]db.ListNoticesWithSenderRow, error)
	UpdateMessage(context.Context, uuid.UUID, string) (db.Notice, error)
	Delete(context.Context, uuid.UUID) error
}

// Cuerpo esperado en POST /notices
type noticeCreateRequest struct {
	IDSemester uuid.UUID `json:"id_semester"`
	IDSender   uuid.UUID `json:"id_sender"`
	Title      string    `json:"title"`
	Message    string    `json:"message"`
}

// Cuerpo esperado en PUT /notices/{id}
type noticeUpdateRequest struct {
	Message string `json:"message"`
}

// RegisterNoticeRoutes registra /notices (GET listar | POST crear),
// /notices/{id} (GET | PUT | DELETE) y el historial de una cursada.
func RegisterNoticeRoutes(mux *http.ServeMux, notices *service.NoticeService) {
	mux.HandleFunc("/notices", noticesCollectionHandler(notices))
	mux.HandleFunc("/notices/{id}", noticeHandler(notices))
	mux.HandleFunc("/semesters/{id}/notices", semesterNoticesHandler(notices))
}

func noticesCollectionHandler(notices noticeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			result, err := notices.List(r.Context())
			if err != nil {
				HandleServiceError(w, err)
				return
			}
			WriteJSON(w, http.StatusOK, result)
		case http.MethodPost:
			var request noticeCreateRequest
			if !DecodeJSON(w, r, &request) {
				return
			}
			result, err := notices.Create(r.Context(), db.CreateNoticeParams{
				IDSemester: request.IDSemester,
				IDSender:   request.IDSender,
				Title:      request.Title,
				Message:    request.Message,
			})
			if err != nil {
				HandleServiceError(w, err)
				return
			}
			WriteJSON(w, http.StatusCreated, result)
		default:
			HandleError(w, http.StatusMethodNotAllowed, "método no permitido")
		}
	}
}

func noticeHandler(notices noticeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := ParsePathUUID(w, r, "id")
		if !ok {
			return
		}
		switch r.Method {
		case http.MethodGet:
			result, err := notices.Get(r.Context(), id)
			if err != nil {
				HandleServiceError(w, err)
				return
			}
			WriteJSON(w, http.StatusOK, result)
		case http.MethodPut:
			var request noticeUpdateRequest
			if !DecodeJSON(w, r, &request) {
				return
			}
			result, err := notices.UpdateMessage(r.Context(), id, request.Message)
			if err != nil {
				HandleServiceError(w, err)
				return
			}
			WriteJSON(w, http.StatusOK, result)
		case http.MethodDelete:
			if err := notices.Delete(r.Context(), id); err != nil {
				HandleServiceError(w, err)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			HandleError(w, http.StatusMethodNotAllowed, "método no permitido")
		}
	}
}

// semesterNoticesHandler sirve el historial publicado de una cursada
// (GET /semesters/{id}/notices).
func semesterNoticesHandler(notices noticeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			HandleError(w, http.StatusMethodNotAllowed, "método no permitido")
			return
		}
		id, ok := ParsePathUUID(w, r, "id")
		if !ok {
			return
		}
		result, err := notices.ListBySemester(r.Context(), id)
		if err != nil {
			HandleServiceError(w, err)
			return
		}
		WriteJSON(w, http.StatusOK, result)
	}
}
