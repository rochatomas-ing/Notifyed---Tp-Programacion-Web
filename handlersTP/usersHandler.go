package handlersTP

import (
	"net/http"
	"net/mail"

	db "servidor-NotifyEd/db/sqlc"

	"golang.org/x/crypto/bcrypt"
)

type createUserRequest struct {
	Fullname string `json:"fullname"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type updateUserFullnameRequest struct {
	Fullname string `json:"fullname"`
}

type updateUserPasswordRequest struct {
	Password string `json:"password"`
}

func RegisterUserRoutes(mux *http.ServeMux, users *db.Queries) {
	mux.HandleFunc("/users", usersHandler(users))
	mux.HandleFunc("/users/{id}", userHandler(users))
}

func usersHandler(users *db.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			listUsers(users, w, r)
		case http.MethodPost:
			createUser(users, w, r)
		default:
			HandleError(w, http.StatusMethodNotAllowed, "método no permitido")
		}
	}
}

// GET /users: Listar todos los usuarios
func listUsers(users *db.Queries, w http.ResponseWriter, r *http.Request) {
	usuarios, err := users.ListUsers(r.Context())

	if err != nil {
		HandleError(w, http.StatusInternalServerError, "error al listar usuarios")
		return
	}

	if usuarios == nil {
		usuarios = []db.ListUsersRow{}
	}

	WriteJSON(w, http.StatusOK, usuarios)

}

// POST /users: Crear un nuevo usuario
func createUser(users *db.Queries, w http.ResponseWriter, r *http.Request) {

	var req createUserRequest

	if !DecodeJSON(w, r, &req) {
		return
	}

	//Valido que los campos no esten vacios
	if req.Fullname == "" || req.Email == "" || req.Password == "" {
		HandleError(w, http.StatusBadRequest, "todos los campos son requeridos")
		return
	}

	//Valido que el email tenga un formato válido
	if _, err := mail.ParseAddress(req.Email); err != nil {
		HandleError(w, http.StatusBadRequest, "email inválido")
		return
	}

	//Valido que el email no este registrado
	_, err := users.GetUserByEmail(r.Context(), req.Email)
	if err == nil {
		HandleError(w, http.StatusBadRequest, "email ya registrado")
		return
	}

	// Hash de la contraseña
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		HandleError(w, http.StatusInternalServerError, "error al generar hash de contraseña")
		return
	}

	// Creo el usuario en la base de datos
	usuario, err := users.CreateUser(r.Context(), db.CreateUserParams{
		Fullname:     req.Fullname,
		Email:        req.Email,
		PasswordHash: string(passwordHash),
	})

	if err != nil {
		HandleError(w, http.StatusInternalServerError, "error al crear usuario")
		return
	}

	WriteJSON(w, http.StatusCreated, usuario)

}

func userHandler(users *db.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getUser(users, w, r)
		case http.MethodPut:
			updateUser(users, w, r)
		case http.MethodDelete:
			deleteUser(users, w, r)
		default:
			HandleError(w, http.StatusMethodNotAllowed, "método no permitido")
		}
	}
}

// GET /users/{id}: Obtener un usuario por ID
func getUser(users *db.Queries, w http.ResponseWriter, r *http.Request) {

	ctx := r.Context()
	id, err := ParseUUID(r, "/users/")
	if err != nil {
		HandleError(w, http.StatusBadRequest, "id inválido")
		return
	}

	usuario, err := users.GetUserById(ctx, id)
	if err != nil {
		HandleError(w, http.StatusNotFound, "usuario no encontrado")
		return
	}

	WriteJSON(w, http.StatusOK, usuario)
}

func updateUser(users *db.Queries, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := ParseUUID(r, "/users/")
	if err != nil {
		HandleError(w, http.StatusBadRequest, "id inválido")
		return
	}

	var req updateUserFullnameRequest

	if !DecodeJSON(w, r, &req) {
		return
	}

	if req.Fullname == "" {
		HandleError(w, http.StatusBadRequest, "fullname no puede estar vacío")
		return
	}

	usuario, err := users.UpdateUserFullname(ctx, db.UpdateUserFullnameParams{
		IDUser:   id,
		Fullname: req.Fullname,
	})

	if err != nil {
		HandleError(w, http.StatusInternalServerError, "error al actualizar usuario")
		return
	}

	WriteJSON(w, http.StatusOK, usuario)
}

func deleteUser(users *db.Queries, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := ParseUUID(r, "/users/")
	if err != nil {
		HandleError(w, http.StatusBadRequest, "id inválido")
		return
	}

	rowsAffected, err := users.DeleteUser(ctx, id)
	if err != nil {
		HandleError(w, http.StatusInternalServerError, "error al eliminar usuario")
		return
	}

	if rowsAffected == 0 {
		HandleError(w, http.StatusNotFound, "usuario no encontrado")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
