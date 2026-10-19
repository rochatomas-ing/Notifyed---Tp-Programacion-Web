package handlersTP

import (
	"context"
	"net/http"
	"time"

	db "servidor-NotifyEd/db/sqlc"
	"servidor-NotifyEd/service"

	"github.com/google/uuid"
)

type semesterService interface {
	Create(context.Context, db.CreateSemesterParams) (db.Semester, error)
	Get(context.Context, uuid.UUID) (db.Semester, error)
	GetByToken(context.Context, string) (db.GetSemesterByTokenRow, error)
	ListByProfessor(context.Context, uuid.UUID) ([]db.ListSemestersByProfessorRow, error)
	UpdateEndDate(context.Context, uuid.UUID, time.Time) (db.Semester, error)
	Delete(context.Context, uuid.UUID) error
}

type semesterCreateRequest struct {
	IDCourse    uuid.UUID `json:"id_course"`
	ProfessorID uuid.UUID `json:"professor_id"`
	Year        int32     `json:"year"`
	SubToken    string    `json:"sub_token"`
	EndDate     string    `json:"end_date"`
}

type semesterUpdateRequest struct {
	EndDate string `json:"end_date"`
}

func RegisterSemesterRoutes(mux *http.ServeMux, semesters *service.SemesterService) {
	mux.HandleFunc("/semesters", semestersCollectionHandler(semesters))
	mux.HandleFunc("/semesters/{idOrToken}", semesterHandler(semesters))
	mux.HandleFunc("/profesores/{id}/semesters", professorSemestersHandler(semesters))
}

func semestersCollectionHandler(semesters semesterService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			createSemester(semesters, w, r)
		case http.MethodGet:
			listSemestersByProfessorQuery(semesters, w, r)
		default:
			HandleError(w, http.StatusMethodNotAllowed, "método no permitido")
		}
	}
}

func createSemester(semesters semesterService, w http.ResponseWriter, r *http.Request) {
	var request semesterCreateRequest
	if !DecodeJSON(w, r, &request) {
		return
	}
	endDate, err := time.Parse("2006-01-02", request.EndDate)
	if err != nil {
		HandleError(w, http.StatusBadRequest, "end_date debe tener formato YYYY-MM-DD")
		return
	}
	result, err := semesters.Create(r.Context(), db.CreateSemesterParams{
		IDCourse: request.IDCourse, ProfessorID: request.ProfessorID,
		Year: request.Year, SubToken: request.SubToken, EndDate: endDate,
	})
	if err != nil {
		HandleServiceError(w, err)
		return
	}
	WriteJSON(w, http.StatusCreated, result)
}

func listSemestersByProfessorQuery(semesters semesterService, w http.ResponseWriter, r *http.Request) {
	professorID, err := uuid.Parse(r.URL.Query().Get("professor_id"))
	if err != nil {
		HandleError(w, http.StatusBadRequest, "se requiere un professor_id UUID válido para listar cursadas")
		return
	}
	result, err := semesters.ListByProfessor(r.Context(), professorID)
	if err != nil {
		HandleServiceError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, result)
}

func semesterHandler(semesters semesterService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		identifier := r.PathValue("idOrToken")
		switch r.Method {
		case http.MethodGet:
			getSemester(semesters, w, r, identifier)
		case http.MethodPut:
			updateSemester(semesters, w, r, identifier)
		case http.MethodDelete:
			deleteSemester(semesters, w, r, identifier)
		default:
			HandleError(w, http.StatusMethodNotAllowed, "método no permitido")
		}
	}
}

func getSemester(semesters semesterService, w http.ResponseWriter, r *http.Request, identifier string) {
	if id, err := uuid.Parse(identifier); err == nil {
		result, err := semesters.Get(r.Context(), id)
		if err != nil {
			HandleServiceError(w, err)
			return
		}
		WriteJSON(w, http.StatusOK, result)
		return
	}
	result, err := semesters.GetByToken(r.Context(), identifier)
	if err != nil {
		HandleServiceError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, result)
}

func updateSemester(semesters semesterService, w http.ResponseWriter, r *http.Request, identifier string) {
	id, err := uuid.Parse(identifier)
	if err != nil {
		HandleError(w, http.StatusBadRequest, "el identificador debe ser un UUID válido")
		return
	}
	var request semesterUpdateRequest
	if !DecodeJSON(w, r, &request) {
		return
	}
	endDate, err := time.Parse("2006-01-02", request.EndDate)
	if err != nil {
		HandleError(w, http.StatusBadRequest, "end_date debe tener formato YYYY-MM-DD")
		return
	}
	result, err := semesters.UpdateEndDate(r.Context(), id, endDate)
	if err != nil {
		HandleServiceError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, result)
}

func deleteSemester(semesters semesterService, w http.ResponseWriter, r *http.Request, identifier string) {
	id, err := uuid.Parse(identifier)
	if err != nil {
		HandleError(w, http.StatusBadRequest, "el identificador debe ser un UUID válido")
		return
	}
	if err := semesters.Delete(r.Context(), id); err != nil {
		HandleServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func professorSemestersHandler(semesters semesterService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			HandleError(w, http.StatusMethodNotAllowed, "método no permitido")
			return
		}
		id, ok := ParsePathUUID(w, r, "id")
		if !ok {
			return
		}
		listProfessorSemesters(semesters, w, r, id)
	}
}

func listProfessorSemesters(semesters semesterService, w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	result, err := semesters.ListByProfessor(r.Context(), id)
	if err != nil {
		HandleServiceError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, result)
}
