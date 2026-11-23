package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"net/mail"
	"net/url"
	"strconv"
	"strings"
	"time"

	db "servidor-NotifyEd/db/sqlc"
	dbconn "servidor-NotifyEd/db_conn"
	"servidor-NotifyEd/middleware"
	"servidor-NotifyEd/views"

	"github.com/google/uuid"
	"github.com/skip2/go-qrcode"
)

// ensureProfessor garantiza que exista el usuario profesor en la base de datos
// para evitar violaciones de clave foránea (FK) en semesters y notices.
func ensureProfessor(ctx context.Context, conn *sql.DB, queries *db.Queries) db.User {
	profID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	user, err := queries.GetUserById(ctx, profID)
	if err == nil {
		return user
	}

	// Si no existe, insertar el usuario profesor por defecto del seed
	const insertQuery = `
	INSERT INTO users (id_user, fullname, email, password_hash)
	VALUES ('11111111-1111-1111-1111-111111111111', 'Profesor de Prueba', 'profesor@test.com', 'seed-no-hay-login-en-este-tp')
	ON CONFLICT (id_user) DO NOTHING;
	`
	if _, execErr := conn.ExecContext(ctx, insertQuery); execErr == nil {
		if u, err := queries.GetUserById(ctx, profID); err == nil {
			return u
		}
	}

	// Si hay otro usuario en la tabla users, usarlo como profesor activo
	users, err := queries.ListUsers(ctx)
	if err == nil && len(users) > 0 {
		return db.User{
			IDUser:   users[0].IDUser,
			Fullname: users[0].Fullname,
			Email:    users[0].Email,
		}
	}

	return db.User{
		IDUser:   profID,
		Fullname: "Profesor de Prueba",
		Email:    "profesor@test.com",
	}
}

// buildSubscribeURL construye la URL completa de suscripción para el código QR o enlaces
func buildSubscribeURL(r *http.Request, token string) string {
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/subscribir/%s", scheme, r.Host, token)
}

// generateSubToken genera un token único de 16 caracteres para el QR de suscripción
func generateSubToken() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return uuid.New().String()[:16]
	}
	return hex.EncodeToString(b)
}

func main() {
	conn, err := dbconn.Connect()
	if err != nil {
		fmt.Printf("Error al conectar con la base de datos: %s\n", err)
		return
	}
	defer conn.Close()

	queries := db.New(conn)

	// Garantizar la existencia del usuario profesor al iniciar
	ensureProfessor(context.Background(), conn, queries)

	mux := setupRoutes(conn, queries)

	port := ":8080"
	fmt.Printf("Servidor NotifyEd escuchando en http://localhost%s\n", port)

	err = http.ListenAndServe(port, middleware.LoggingMiddleware(mux))
	if err != nil {
		fmt.Printf("Error al iniciar el servidor: %s\n", err)
	}
}

// setupRoutes inicializa y registra todas las rutas HTTP de la aplicación NotifyEd
func setupRoutes(conn *sql.DB, queries *db.Queries) *http.ServeMux {
	mux := http.NewServeMux()

	// -------------------------------------------------------------------------
	// Archivos estáticos (/static/)
	// -------------------------------------------------------------------------
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	// -------------------------------------------------------------------------
	// Healthcheck
	// -------------------------------------------------------------------------
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		if err := conn.PingContext(r.Context()); err != nil {
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	// -------------------------------------------------------------------------
	// 1. Apartado de Materias y Cursadas (Página Principal)
	// -------------------------------------------------------------------------
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && r.URL.Path != "/materias" {
			http.NotFound(w, r)
			return
		}

		ctx := r.Context()
		prof := ensureProfessor(ctx, conn, queries)

		courses, err := queries.ListCourses(ctx)
		if err != nil {
			courses = []db.Course{}
		}

		semesters, err := queries.ListSemestersWithCourseByProfessor(ctx, prof.IDUser)
		if err != nil {
			semesters = []db.ListSemestersWithCourseByProfessorRow{}
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if r.Header.Get("HX-Request") == "true" {
			if err := views.CoursesContent(courses, semesters, true).Render(ctx, w); err != nil {
				http.Error(w, "Error al renderizar el contenido de materias", http.StatusInternalServerError)
			}
			return
		}
		if err := views.CoursesPage(courses, semesters, prof.Fullname).Render(ctx, w); err != nil {
			http.Error(w, "Error al renderizar la vista de materias", http.StatusInternalServerError)
		}
	})

	// Crear nueva materia (POST /courses)
	mux.HandleFunc("POST /courses", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Formulario inválido", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		prof := ensureProfessor(ctx, conn, queries)

		name := strings.TrimSpace(r.FormValue("name"))
		if name == "" || len([]rune(name)) > 40 {
			http.Error(w, "El nombre de la materia es obligatorio y debe tener hasta 40 caracteres", http.StatusBadRequest)
			return
		}

		if _, err := queries.CreateCourse(ctx, name); err != nil {
			log.Printf("[POST /courses] Error al crear materia: %v", err)
			http.Error(w, fmt.Sprintf("Error al guardar la materia en la base de datos: %v", err), http.StatusInternalServerError)
			return
		}

		// Si la petición proviene de HTMX, devuelve la lista actualizada más el formulario OOB
		if r.Header.Get("HX-Request") == "true" {
			courses, _ := queries.ListCourses(ctx)
			semesters, _ := queries.ListSemestersWithCourseByProfessor(ctx, prof.IDUser)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			views.CourseCreatedResponse(courses, semesters).Render(ctx, w)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	// Eliminar materia (POST /courses/delete)
	mux.HandleFunc("POST /courses/delete", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Formulario inválido", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		prof := ensureProfessor(ctx, conn, queries)

		id, err := uuid.Parse(r.FormValue("id"))
		if err != nil {
			http.Error(w, "ID de materia inválido", http.StatusBadRequest)
			return
		}

		if _, err := queries.DeleteCourse(ctx, id); err != nil {
			log.Printf("[POST /courses/delete] Error al eliminar materia: %v", err)
			http.Error(w, "No se puede eliminar la materia (tiene cursadas u otros datos asociados)", http.StatusConflict)
			return
		}

		if r.Header.Get("HX-Request") == "true" {
			courses, _ := queries.ListCourses(ctx)
			semesters, _ := queries.ListSemestersWithCourseByProfessor(ctx, prof.IDUser)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			views.CourseCreatedResponse(courses, semesters).Render(ctx, w)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	// Crear nueva cursada (POST /semesters)
	mux.HandleFunc("POST /semesters", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			log.Printf("[POST /semesters] Error al parsear formulario: %v", err)
			http.Error(w, "Formulario inválido", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		prof := ensureProfessor(ctx, conn, queries)

		courseIDStr := r.FormValue("id_course")
		courseID, err := uuid.Parse(courseIDStr)
		if err != nil {
			log.Printf("[POST /semesters] ID de materia inválido '%s': %v", courseIDStr, err)
			http.Error(w, "Materia seleccionada inválida", http.StatusBadRequest)
			return
		}

		yearStr := r.FormValue("year")
		yearInt, err := strconv.Atoi(yearStr)
		if err != nil || yearInt < 2000 || yearInt > 2100 {
			log.Printf("[POST /semesters] Año inválido '%s': %v", yearStr, err)
			http.Error(w, "Año de cursada inválido", http.StatusBadRequest)
			return
		}

		endDateStr := r.FormValue("end_date")
		endDate, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			log.Printf("[POST /semesters] Fecha inválida '%s': %v", endDateStr, err)
			http.Error(w, "Fecha de finalización inválida (formato YYYY-MM-DD)", http.StatusBadRequest)
			return
		}

		subToken := generateSubToken()

		_, err = queries.CreateSemester(ctx, db.CreateSemesterParams{
			IDCourse:    courseID,
			ProfessorID: prof.IDUser,
			Year:        int32(yearInt),
			SubToken:    subToken,
			EndDate:     endDate,
		})
		if err != nil {
			log.Printf("[POST /semesters] Error de base de datos al crear cursada: %v", err)
			http.Error(w, fmt.Sprintf("Error al crear la cursada en la base de datos: %v", err), http.StatusInternalServerError)
			return
		}

		if r.Header.Get("HX-Request") == "true" {
			courses, _ := queries.ListCourses(ctx)
			semesters, _ := queries.ListSemestersWithCourseByProfessor(ctx, prof.IDUser)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			views.CoursesList(courses, semesters).Render(ctx, w)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	// Eliminar cursada (POST /semesters/delete)
	mux.HandleFunc("POST /semesters/delete", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Formulario inválido", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		prof := ensureProfessor(ctx, conn, queries)

		id, err := uuid.Parse(r.FormValue("id"))
		if err != nil {
			http.Error(w, "ID de cursada inválido", http.StatusBadRequest)
			return
		}

		if _, err := queries.DeleteSemester(ctx, id); err != nil {
			log.Printf("[POST /semesters/delete] Error al eliminar cursada: %v", err)
			http.Error(w, "Error al eliminar la cursada", http.StatusInternalServerError)
			return
		}

		if r.Header.Get("HX-Request") == "true" {
			courses, _ := queries.ListCourses(ctx)
			semesters, _ := queries.ListSemestersWithCourseByProfessor(ctx, prof.IDUser)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			views.CoursesList(courses, semesters).Render(ctx, w)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	// -------------------------------------------------------------------------
	// 2. Apartado de Historial de Avisos
	// -------------------------------------------------------------------------
	mux.HandleFunc("GET /avisos", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		prof := ensureProfessor(ctx, conn, queries)

		courseFilter := strings.TrimSpace(r.URL.Query().Get("course_id"))
		semesterFilter := strings.TrimSpace(r.URL.Query().Get("semester_id"))

		courses, _ := queries.ListCourses(ctx)
		semesters, _ := queries.ListSemestersWithCourseByProfessor(ctx, prof.IDUser)

		var semNullUUID uuid.NullUUID
		if semesterFilter != "" {
			if semID, parseErr := uuid.Parse(semesterFilter); parseErr == nil {
				semNullUUID = uuid.NullUUID{UUID: semID, Valid: true}
			}
		}

		var courseNullUUID uuid.NullUUID
		if courseFilter != "" {
			if courseID, parseErr := uuid.Parse(courseFilter); parseErr == nil {
				courseNullUUID = uuid.NullUUID{UUID: courseID, Valid: true}
			}
		}

		notices, err := queries.ListNoticesFiltered(ctx, db.ListNoticesFilteredParams{
			IDSender:   prof.IDUser,
			IDSemester: semNullUUID,
			IDCourse:   courseNullUUID,
		})
		if err != nil || notices == nil {
			notices = []db.ListNoticesFilteredRow{}
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if r.Header.Get("HX-Target") == "notices-list-container" {
			views.NoticesList(notices).Render(ctx, w)
			return
		}
		if r.Header.Get("HX-Request") == "true" {
			views.NoticesContent(notices, courses, semesters, courseFilter, semesterFilter, true).Render(ctx, w)
			return
		}
		views.NoticesPage(notices, courses, semesters, courseFilter, semesterFilter, prof.Fullname).Render(ctx, w)
	})

	// Eliminar aviso (POST /avisos/delete)
	mux.HandleFunc("POST /avisos/delete", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Formulario inválido", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		prof := ensureProfessor(ctx, conn, queries)

		id, err := uuid.Parse(r.FormValue("id"))
		if err != nil {
			http.Error(w, "ID de aviso inválido", http.StatusBadRequest)
			return
		}

		if _, err := queries.DeleteNotice(ctx, id); err != nil {
			log.Printf("[POST /avisos/delete] Error al eliminar aviso: %v", err)
			http.Error(w, "Error al eliminar el aviso", http.StatusInternalServerError)
			return
		}

		if r.Header.Get("HX-Request") == "true" {
			notices, _ := queries.ListNoticesFiltered(ctx, db.ListNoticesFilteredParams{
				IDSender: prof.IDUser,
			})
			if notices == nil {
				notices = []db.ListNoticesFilteredRow{}
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			views.NoticesList(notices).Render(ctx, w)
			return
		}

		http.Redirect(w, r, "/avisos", http.StatusSeeOther)
	})

	// -------------------------------------------------------------------------
	// 3. Apartado de Gestión de Alumnos
	// -------------------------------------------------------------------------
	mux.HandleFunc("GET /alumnos", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		prof := ensureProfessor(ctx, conn, queries)

		email := strings.TrimSpace(r.URL.Query().Get("email"))
		var inscriptions []db.ListInscriptionsByEmailAndProfessorRow
		searched := false

		if email != "" {
			searched = true
			res, err := queries.ListInscriptionsByEmailAndProfessor(ctx, db.ListInscriptionsByEmailAndProfessorParams{
				Btrim:       email,
				ProfessorID: prof.IDUser,
			})
			if err == nil && res != nil {
				inscriptions = res
			}
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if r.Header.Get("HX-Target") == "students-results-container" {
			views.StudentsResults(email, inscriptions, searched).Render(ctx, w)
			return
		}
		if r.Header.Get("HX-Request") == "true" {
			views.StudentsContent(email, inscriptions, searched, true).Render(ctx, w)
			return
		}
		views.StudentsPage(email, inscriptions, searched, prof.Fullname).Render(ctx, w)
	})

	// Alternar estado de suscripción (POST /alumnos/toggle)
	mux.HandleFunc("POST /alumnos/toggle", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Formulario inválido", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		prof := ensureProfessor(ctx, conn, queries)

		id, err := uuid.Parse(r.FormValue("id"))
		if err != nil {
			http.Error(w, "ID de inscripción inválido", http.StatusBadRequest)
			return
		}

		activeVal := r.FormValue("active") == "true"
		email := r.FormValue("email")

		_, err = queries.UpdateInscriptionActiveById(ctx, db.UpdateInscriptionActiveByIdParams{
			IDInscription: id,
			Active:        activeVal,
		})
		if err != nil {
			log.Printf("[POST /alumnos/toggle] Error al actualizar estado del alumno: %v", err)
			http.Error(w, "Error al actualizar estado del alumno", http.StatusInternalServerError)
			return
		}

		// Si es HTMX, devuelve únicamente la fila de la tabla actualizada in-situ
		if r.Header.Get("HX-Request") == "true" {
			res, _ := queries.ListInscriptionsByEmailAndProfessor(ctx, db.ListInscriptionsByEmailAndProfessorParams{
				Btrim:       email,
				ProfessorID: prof.IDUser,
			})
			for _, item := range res {
				if item.IDInscription == id {
					w.Header().Set("Content-Type", "text/html; charset=utf-8")
					views.StudentRow(item, email).Render(ctx, w)
					return
				}
			}
		}

		http.Redirect(w, r, "/alumnos?email="+url.QueryEscape(email), http.StatusSeeOther)
	})

	// Eliminar inscripción de alumno (POST /alumnos/delete)
	mux.HandleFunc("POST /alumnos/delete", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Formulario inválido", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		id, err := uuid.Parse(r.FormValue("id"))
		if err != nil {
			http.Error(w, "ID de inscripción inválido", http.StatusBadRequest)
			return
		}
		email := r.FormValue("email")

		if _, err := queries.DeleteInscription(ctx, id); err != nil {
			log.Printf("[POST /alumnos/delete] Error al eliminar inscripción: %v", err)
			http.Error(w, "Error al eliminar inscripción", http.StatusInternalServerError)
			return
		}

		// Si es HTMX, 200 OK vacía le indica a hx-swap="delete" que elimine la fila del DOM
		if r.Header.Get("HX-Request") == "true" {
			w.WriteHeader(http.StatusOK)
			return
		}

		http.Redirect(w, r, "/alumnos?email="+url.QueryEscape(email), http.StatusSeeOther)
	})

	// -------------------------------------------------------------------------
	// 4. Apartado de Publicar Aviso
	// -------------------------------------------------------------------------
	mux.HandleFunc("GET /publicar-aviso", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		prof := ensureProfessor(ctx, conn, queries)

		semesters, err := queries.ListSemestersWithCourseByProfessor(ctx, prof.IDUser)
		if err != nil {
			semesters = []db.ListSemestersWithCourseByProfessorRow{}
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if r.Header.Get("HX-Request") == "true" {
			views.PublishNoticeContent(semesters, true).Render(ctx, w)
			return
		}
		views.PublishNoticePage(semesters, prof.Fullname).Render(ctx, w)
	})

	// Publicar aviso (POST /avisos)
	mux.HandleFunc("POST /avisos", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Formulario inválido", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		prof := ensureProfessor(ctx, conn, queries)

		semesterID, err := uuid.Parse(r.FormValue("id_semester"))
		if err != nil {
			http.Error(w, "Cursada seleccionada inválida", http.StatusBadRequest)
			return
		}

		title := strings.TrimSpace(r.FormValue("title"))
		message := strings.TrimSpace(r.FormValue("message"))

		if title == "" || message == "" {
			http.Error(w, "El título y el mensaje del aviso son obligatorios", http.StatusBadRequest)
			return
		}

		_, err = queries.CreateNotice(ctx, db.CreateNoticeParams{
			IDSemester: semesterID,
			IDSender:   prof.IDUser,
			Title:      title,
			Message:    message,
		})
		if err != nil {
			log.Printf("[POST /avisos] Error al guardar aviso: %v", err)
			http.Error(w, fmt.Sprintf("Error al guardar el aviso en la base de datos: %v", err), http.StatusInternalServerError)
			return
		}

		// Con HTMX, HX-Location realiza una transición SPA hacia /avisos sin recargar la página
		if r.Header.Get("HX-Request") == "true" {
			w.Header().Set("HX-Location", "/avisos")
			w.WriteHeader(http.StatusOK)
			return
		}

		// Redirige al historial de avisos (PRG)
		http.Redirect(w, r, "/avisos", http.StatusSeeOther)
	})

	// -------------------------------------------------------------------------
	// 5. Suscripción de Alumnos y Generación de Códigos QR
	// -------------------------------------------------------------------------

	// Servir imagen PNG del Código QR (GET /qr/{token})
	mux.HandleFunc("GET /qr/{token}", func(w http.ResponseWriter, r *http.Request) {
		token := r.PathValue("token")
		if token == "" {
			http.NotFound(w, r)
			return
		}

		ctx := r.Context()
		_, err := queries.GetSemesterByToken(ctx, token)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		subscribeURL := buildSubscribeURL(r, token)
		png, err := qrcode.Encode(subscribeURL, qrcode.Medium, 256)
		if err != nil {
			log.Printf("[GET /qr/%s] Error al generar QR: %v", token, err)
			http.Error(w, "Error al generar código QR", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.Write(png)
	})

	// Vista completa para proyectar el Código QR del profesor (GET /cursadas/{token}/qr)
	mux.HandleFunc("GET /cursadas/{token}/qr", func(w http.ResponseWriter, r *http.Request) {
		token := r.PathValue("token")
		if token == "" {
			http.NotFound(w, r)
			return
		}

		ctx := r.Context()
		semester, err := queries.GetSemesterByToken(ctx, token)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		prof, err := queries.GetUserById(ctx, semester.ProfessorID)
		profName := "Profesor"
		if err == nil {
			profName = prof.Fullname
		}

		subscribeURL := buildSubscribeURL(r, token)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := views.QRPage(semester, profName, subscribeURL).Render(ctx, w); err != nil {
			http.Error(w, "Error al renderizar página QR", http.StatusInternalServerError)
		}
	})

	// Vista símil login de suscripción para el alumno (GET /subscribir/{token})
	mux.HandleFunc("GET /subscribir/{token}", func(w http.ResponseWriter, r *http.Request) {
		token := r.PathValue("token")
		if token == "" {
			http.NotFound(w, r)
			return
		}

		ctx := r.Context()
		semester, err := queries.GetSemesterByToken(ctx, token)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		prof, err := queries.GetUserById(ctx, semester.ProfessorID)
		profName := "Profesor"
		if err == nil {
			profName = prof.Fullname
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := views.SubscribePage(semester, profName, "", "", false).Render(ctx, w); err != nil {
			http.Error(w, "Error al renderizar página de suscripción", http.StatusInternalServerError)
		}
	})

	// Procesar formulario de suscripción del alumno (POST /subscribir/{token})
	mux.HandleFunc("POST /subscribir/{token}", func(w http.ResponseWriter, r *http.Request) {
		token := r.PathValue("token")
		if token == "" {
			http.NotFound(w, r)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, "Formulario inválido", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		semester, err := queries.GetSemesterByToken(ctx, token)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		prof, err := queries.GetUserById(ctx, semester.ProfessorID)
		profName := "Profesor"
		if err == nil {
			profName = prof.Fullname
		}

		if !semester.IsOpen {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			if r.Header.Get("HX-Request") == "true" {
				views.SubscribeBox(semester, profName, "", "El plazo de inscripción para esta cursada ha finalizado.", false).Render(ctx, w)
				return
			}
			views.SubscribePage(semester, profName, "", "El plazo de inscripción para esta cursada ha finalizado.", false).Render(ctx, w)
			return
		}

		rawEmail := strings.TrimSpace(r.FormValue("email"))
		parsedMail, err := mail.ParseAddress(rawEmail)
		if err != nil || rawEmail == "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			if r.Header.Get("HX-Request") == "true" {
				views.SubscribeBox(semester, profName, rawEmail, "Por favor, ingresa un correo electrónico válido (ej: nombre@dominio.com).", false).Render(ctx, w)
				return
			}
			views.SubscribePage(semester, profName, rawEmail, "Por favor, ingresa un correo electrónico válido (ej: nombre@dominio.com).", false).Render(ctx, w)
			return
		}

		cleanEmail := strings.ToLower(parsedMail.Address)

		_, err = queries.CreateInscription(ctx, db.CreateInscriptionParams{
			IDSemester: semester.IDSemester,
			Email:      cleanEmail,
		})
		if err != nil {
			log.Printf("[POST /subscribir/%s] Error al registrar inscripción: %v", token, err)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			if r.Header.Get("HX-Request") == "true" {
				views.SubscribeBox(semester, profName, cleanEmail, "Ocurrió un error al registrar la suscripción. Intente nuevamente.", false).Render(ctx, w)
				return
			}
			views.SubscribePage(semester, profName, cleanEmail, "Ocurrió un error al registrar la suscripción. Intente nuevamente.", false).Render(ctx, w)
			return
		}

		// Suscripción exitosa
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if r.Header.Get("HX-Request") == "true" {
			views.SubscribeBox(semester, profName, cleanEmail, "", true).Render(ctx, w)
			return
		}
		views.SubscribePage(semester, profName, cleanEmail, "", true).Render(ctx, w)
	})

	return mux
}

