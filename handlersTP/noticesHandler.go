package handlersTP

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/google/uuid"

	db "servidor-NotifyEd/db/sqlc"
)

// Cuerpo esperado en POST /notices
type createNoticeRequest struct {
	IDSemester string `json:"id_semester"`
	IDSender   string `json:"id_sender"`
	Title      string `json:"title"`
	Message    string `json:"message"`
}

// Cuerpo esperado en PUT /notices/{id}
type updateNoticeRequest struct {
	Message string `json:"message"`
}

// HandleNotices atiende /notices: GET (listar) | POST (crear).
func HandleNotices(repo *db.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			listNotices(repo, w, r)
		case http.MethodPost:
			createNotice(repo, w, r)
		default:
			HandleError(w, http.StatusMethodNotAllowed, "metodo no permitido")
		}
	}
}

// HandleNotice atiende /notices/{id}: GET (obtener) | PUT (actualizar) | DELETE (borrar).
func HandleNotice(repo *db.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getNotice(repo, w, r)
		case http.MethodPut:
			updateNotice(repo, w, r)
		case http.MethodDelete:
			deleteNotice(repo, w, r)
		default:
			HandleError(w, http.StatusMethodNotAllowed, "metodo no permitido")
		}
	}
}

func createNotice(repo *db.Queries, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Decodificar el cuerpo JSON de la solicitud y verificar que los campos esten correctamente formateados
	var body createNoticeRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		HandleError(w, http.StatusBadRequest, "JSON inválido")
		return
	}

	// Validar que los campos obligatorios no estén vacíos
	if body.Title == "" || body.Message == "" {
		HandleError(w, http.StatusBadRequest, "title y message no pueden estar vacíos")
		return
	}

	// Validar que el campo IDSemester sea UUID válido
	idSemester, err := uuid.Parse(body.IDSemester)
	if err != nil {
		HandleError(w, http.StatusBadRequest, "id_semester no es un UUID válido")
		return
	}

	// Validar que el campo IDSender sea UUID válido
	idSender, err := uuid.Parse(body.IDSender)
	if err != nil {
		HandleError(w, http.StatusBadRequest, "id_sender no es un UUID válido")
		return
	}

	// Verificar que el profesor (IDSender) exista en la base de datos
	if _, err := repo.GetUserById(ctx, idSender); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			HandleError(w, http.StatusBadRequest, "el profesor indicado no existe")
			return
		}
		log.Println("createNotice: error buscando al profesor:", err)
		HandleError(w, http.StatusInternalServerError, "error interno del servidor")
		return
	}

	// Crear el aviso en la base de datos y verificar que no haya habido errores
	aviso, err := repo.CreateNotice(ctx, db.CreateNoticeParams{
		IDSemester: idSemester,
		IDSender:   idSender,
		Title:      body.Title,
		Message:    body.Message,
	})
	if err != nil {
		log.Println("createNotice: error creando el aviso:", err)
		HandleError(w, http.StatusInternalServerError, "error interno del servidor")
		return
	}

	// Responder con el aviso creado y un código de estado 201 (Created)
	WriteJSON(w, http.StatusCreated, aviso)
}

func listNotices(repo *db.Queries, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Listar todos los avisos en la base de datos y verificar que no haya habido errores
	avisos, err := repo.ListNotices(ctx)
	if err != nil {
		log.Println("listNotices: error listando los avisos:", err)
		HandleError(w, http.StatusInternalServerError, "error interno del servidor")
		return
	}

	// Si no hay avisos, devolver un arreglo vacío en lugar de null, para mantener consistencia en la respuesta JSON
	if avisos == nil {
		avisos = []db.Notice{}
	}

	// Responder con la lista de avisos y un código de estado 200 (OK)
	WriteJSON(w, http.StatusOK, avisos)
}

func getNotice(repo *db.Queries, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Validar que el ID en la URL sea un UUID válido
	id, err := ParseUUID(r, "/notices/")
	if err != nil {
		HandleError(w, http.StatusBadRequest, "el id de la URL no es un UUID válido")
		return
	}

	// Obtener el aviso de la base de datos y verificar que no haya habido errores
	aviso, err := repo.GetNoticeById(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			HandleError(w, http.StatusNotFound, "aviso no encontrado")
			return
		}
		log.Println("getNotice: error obteniendo el aviso:", err)
		HandleError(w, http.StatusInternalServerError, "error interno del servidor")
		return
	}

	// Responder con el aviso encontrado y un código de estado 200 (OK)
	WriteJSON(w, http.StatusOK, aviso)
}

func updateNotice(repo *db.Queries, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Validar que el ID en la URL sea un UUID válido
	id, err := ParseUUID(r, "/notices/")
	if err != nil {
		HandleError(w, http.StatusBadRequest, "el id de la URL no es un UUID válido")
		return
	}

	// Decodificar el cuerpo JSON de la solicitud y verificar que los campos esten correctamente formateados
	var body updateNoticeRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		HandleError(w, http.StatusBadRequest, "JSON inválido")
		return
	}

	// Validar que el campo Message no esté vacío
	if body.Message == "" {
		HandleError(w, http.StatusBadRequest, "message no puede estar vacío")
		return
	}

	// Actualizar el mensaje del aviso en la base de datos y verificar que no haya habido errores
	aviso, err := repo.UpdateNoticeMessage(ctx, db.UpdateNoticeMessageParams{
		IDNotice: id,
		Message:  body.Message,
	})
	if err != nil {
		// Si no se encontró el aviso, devolver un error 404 (Not Found)
		if errors.Is(err, sql.ErrNoRows) {
			HandleError(w, http.StatusNotFound, "aviso no encontrado")
			return
		}
		// Si hubo otro error, devolver un error 500 (Internal Server Error)
		log.Println("updateNotice: error actualizando el aviso:", err)
		HandleError(w, http.StatusInternalServerError, "error interno del servidor")
		return
	}

	// Responder con el aviso actualizado y un código de estado 200 (OK)
	WriteJSON(w, http.StatusOK, aviso)
}

func deleteNotice(repo *db.Queries, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Validar que el ID en la URL sea un UUID válido
	id, err := ParseUUID(r, "/notices/")
	if err != nil {
		HandleError(w, http.StatusBadRequest, "el id de la URL no es un UUID válido")
		return
	}

	// Borrar el aviso de la base de datos y verificar que no haya habido errores
	filas, err := repo.DeleteNotice(ctx, id)
	if err != nil {
		// Si hubo un error al borrar el aviso, devolver un error 500 (Internal Server Error)
		log.Println("deleteNotice: error borrando el aviso:", err)
		HandleError(w, http.StatusInternalServerError, "error interno del servidor")
		return
	}

	// Verificar si se eliminó algún aviso y devolver un error 404 (Not Found) si no se encontró el aviso
	if filas == 0 {
		HandleError(w, http.StatusNotFound, "aviso no encontrado")
		return
	}

	// Responder con un código de estado 204 (No Content) indicando que el aviso fue borrado exitosamente
	w.WriteHeader(http.StatusNoContent)
}
