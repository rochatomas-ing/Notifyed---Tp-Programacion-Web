package handlersTP

import (
	"context"
	"net/http"

	db "servidor-NotifyEd/db/sqlc"
	"servidor-NotifyEd/service"

	"github.com/google/uuid"
)

type courseService interface {
	Create(ctx context.Context, name string) (db.Course, error)
	Get(ctx context.Context, id uuid.UUID) (db.Course, error)
	List(ctx context.Context) ([]db.Course, error)
	UpdateName(ctx context.Context, id uuid.UUID, name string) (db.Course, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

func RegisterCourseRoutes(mux *http.ServeMux, courses *service.CourseService, semesters *service.SemesterService) {
	mux.HandleFunc("/courses", coursesCollectionHandler(courses))
	mux.HandleFunc("/courses/{id}", courseHandler(courses))
	mux.HandleFunc("/courses/{id}/semesters", courseSemestersHandler(semesters))
}

func coursesCollectionHandler(courses courseService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			result, err := courses.List(r.Context())
			if err != nil {
				HandleServiceError(w, err)
				return
			}
			WriteJSON(w, http.StatusOK, result)
		case http.MethodPost:
			var request struct {
				Name string `json:"name"`
			}
			if !DecodeJSON(w, r, &request) {
				return
			}
			result, err := courses.Create(r.Context(), request.Name)
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

func courseHandler(courses courseService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := ParsePathUUID(w, r, "id")
		if !ok {
			return
		}
		switch r.Method {
		case http.MethodGet:
			result, err := courses.Get(r.Context(), id)
			if err != nil {
				HandleServiceError(w, err)
				return
			}
			WriteJSON(w, http.StatusOK, result)
		case http.MethodPut:
			var request struct {
				Name string `json:"name"`
			}
			if !DecodeJSON(w, r, &request) {
				return
			}
			result, err := courses.UpdateName(r.Context(), id, request.Name)
			if err != nil {
				HandleServiceError(w, err)
				return
			}
			WriteJSON(w, http.StatusOK, result)
		case http.MethodDelete:
			if err := courses.Delete(r.Context(), id); err != nil {
				HandleServiceError(w, err)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			HandleError(w, http.StatusMethodNotAllowed, "método no permitido")
		}
	}
}

func courseSemestersHandler(semesters interface {
	ListByCourse(context.Context, uuid.UUID) ([]db.Semester, error)
}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			HandleError(w, http.StatusMethodNotAllowed, "método no permitido")
			return
		}
		id, ok := ParsePathUUID(w, r, "id")
		if !ok {
			return
		}
		result, err := semesters.ListByCourse(r.Context(), id)
		if err != nil {
			HandleServiceError(w, err)
			return
		}
		WriteJSON(w, http.StatusOK, result)
	}
}
