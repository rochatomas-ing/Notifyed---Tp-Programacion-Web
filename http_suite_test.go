package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	db "servidor-NotifyEd/db/sqlc"
	dbconn "servidor-NotifyEd/db_conn"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// getTestDB obtiene la conexión a PostgreSQL asegurando que el contenedor esté corriendo
func getTestDB(t *testing.T) (*sql.DB, *db.Queries) {
	t.Helper()

	conn, err := dbconn.Connect()
	if err != nil {
		dsn := os.Getenv("TEST_DB_URL")
		if dsn == "" {
			dsn = "postgres://postgres:postgres@localhost:5432/avisos?sslmode=disable"
		}
		var openErr error
		conn, openErr = sql.Open("postgres", dsn)
		if openErr != nil || conn.Ping() != nil {
			t.Fatalf("Fallo de conexión a la base de datos PostgreSQL (%v). La suite exige que el contenedor de Docker esté activo. Ejecuta 'make up' e intenta nuevamente.", err)
		}
	}

	queries := db.New(conn)
	ensureProfessor(context.Background(), conn, queries)
	return conn, queries
}

// newNoRedirectClient crea un cliente HTTP que no sigue redirecciones automáticas (para verificar 303 y Location)
func newNoRedirectClient() *http.Client {
	return &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: 5 * time.Second,
	}
}

// Helpers de aserción para la suite HTTP SSR
func assertStatus(t *testing.T, resp *http.Response, expected int, ctxMsg string) {
	t.Helper()
	if resp.StatusCode != expected {
		t.Fatalf("[%s] Código de estado HTTP incorrecto: obtenido %d, esperado %d", ctxMsg, resp.StatusCode, expected)
	}
}

func assertRedirect(t *testing.T, resp *http.Response, expectedLocation string, ctxMsg string) {
	t.Helper()
	if resp.StatusCode != http.StatusSeeOther && resp.StatusCode != http.StatusFound {
		t.Fatalf("[%s] Se esperaba redirección HTTP (303 o 302), pero se obtuvo %d", ctxMsg, resp.StatusCode)
	}
	location := resp.Header.Get("Location")
	if location != expectedLocation {
		t.Fatalf("[%s] Cabecera Location de redirección incorrecta: obtenido %q, esperado %q", ctxMsg, location, expectedLocation)
	}
}

func assertContains(t *testing.T, body string, substr string, ctxMsg string) {
	t.Helper()
	if !strings.Contains(body, substr) {
		t.Fatalf("[%s] No se encontró el texto esperado en el HTML renderizado: %q", ctxMsg, substr)
	}
}

func assertNotContains(t *testing.T, body string, substr string, ctxMsg string) {
	t.Helper()
	if strings.Contains(body, substr) {
		t.Fatalf("[%s] Se encontró un texto que NO debería estar presente en el HTML: %q", ctxMsg, substr)
	}
}

func readResponseBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Error al leer cuerpo de respuesta HTTP: %v", err)
	}
	return string(bytes)
}

// -----------------------------------------------------------------------------
// 1. Recorrido de Navegación Base (GET de todos los apartados)
// -----------------------------------------------------------------------------
func TestHTTPSuite_1_Navigation(t *testing.T) {
	conn, queries := getTestDB(t)
	defer conn.Close()

	handler := setupRoutes(conn, queries)
	server := httptest.NewServer(handler)
	defer server.Close()

	client := server.Client()

	tests := []struct {
		name         string
		path         string
		expectedHTML []string
	}{
		{
			name: "Apartado Materias (Raíz)",
			path: "/",
			expectedHTML: []string{
				"NotifyEd",
				"Gestión de Materias y Cursadas",
				"Profesor:",
				"Nueva Materia",
				"Nueva Cursada",
			},
		},
		{
			name: "Apartado Materias (/materias)",
			path: "/materias",
			expectedHTML: []string{
				"NotifyEd",
				"Gestión de Materias y Cursadas",
			},
		},
		{
			name: "Apartado Historial de Avisos",
			path: "/avisos",
			expectedHTML: []string{
				"Historial de Avisos",
				"Filtrar avisos",
				"Por Materia:",
			},
		},
		{
			name: "Apartado Gestión de Alumnos",
			path: "/alumnos",
			expectedHTML: []string{
				"Gestión de Alumnos",
				"Correo electrónico del alumno",
				"Buscar Alumno",
			},
		},
		{
			name: "Apartado Publicar Aviso",
			path: "/publicar-aviso",
			expectedHTML: []string{
				"Publicar Nuevo Aviso",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := client.Get(server.URL + tc.path)
			if err != nil {
				t.Fatalf("Error en GET %s: %v", tc.path, err)
			}
			assertStatus(t, resp, http.StatusOK, "GET "+tc.path)

			body := readResponseBody(t, resp)
			for _, expectedText := range tc.expectedHTML {
				assertContains(t, body, expectedText, tc.name)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// 2. Recorrido de Materias y Cursadas (Patrón PRG)
// -----------------------------------------------------------------------------
func TestHTTPSuite_2_CourseAndSemesterLifecyclePRG(t *testing.T) {
	conn, queries := getTestDB(t)
	defer conn.Close()

	handler := setupRoutes(conn, queries)
	server := httptest.NewServer(handler)
	defer server.Close()

	noRedirectClient := newNoRedirectClient()
	regularClient := server.Client()

	// 1. Crear Materia (POST /courses)
	uniqueCourseName := "Programación Web " + uuid.NewString()[:8]
	formCourse := url.Values{"name": {uniqueCourseName}}

	postResp, err := noRedirectClient.PostForm(server.URL+"/courses", formCourse)
	if err != nil {
		t.Fatalf("Error en POST /courses: %v", err)
	}
	assertRedirect(t, postResp, "/", "Creación de materia con PRG")
	postResp.Body.Close()

	// Seguir redirección (GET /)
	getResp, err := regularClient.Get(server.URL + "/")
	if err != nil {
		t.Fatalf("Error en GET / tras redirección: %v", err)
	}
	assertStatus(t, getResp, http.StatusOK, "GET / tras POST /courses")
	body := readResponseBody(t, getResp)
	assertContains(t, body, uniqueCourseName, "Verificación de la materia en el DOM")

	// Obtener el ID de la materia creada para habilitarle una cursada
	courses, err := queries.ListCourses(context.Background())
	if err != nil {
		t.Fatalf("Error al consultar materias en DB: %v", err)
	}
	var createdCourse db.Course
	for _, c := range courses {
		if c.Name == uniqueCourseName {
			createdCourse = c
			break
		}
	}
	if createdCourse.Name == "" {
		t.Fatalf("No se encontró la materia recién creada en la base de datos")
	}

	// 2. Habilitar Cursada (POST /semesters)
	formSemester := url.Values{
		"id_course": {createdCourse.IDCourse.String()},
		"year":      {"2026"},
		"end_date":  {"2026-12-20"},
	}
	postSemResp, err := noRedirectClient.PostForm(server.URL+"/semesters", formSemester)
	if err != nil {
		t.Fatalf("Error en POST /semesters: %v", err)
	}
	assertRedirect(t, postSemResp, "/", "Creación de cursada con PRG")
	postSemResp.Body.Close()

	// Seguir redirección y verificar que la materia aparezca sin ID visible y con cursadas colapsadas por defecto
	getSemResp, err := regularClient.Get(server.URL + "/")
	if err != nil {
		t.Fatalf("Error en GET / tras habilitar cursada: %v", err)
	}
	bodySem := readResponseBody(t, getSemResp)
	assertNotContains(t, bodySem, "ID: <code>"+createdCourse.IDCourse.String(), "El ID de la materia no debe mostrarse")
	assertContains(t, bodySem, "Ver cursadas (1)", "Botón para desplegar cursadas de la materia")
	assertNotContains(t, bodySem, "Abierta / Vigente", "Por defecto las cursadas deben estar colapsadas")

	// Desplegar cursadas de la materia mediante el endpoint HTMX
	expandReq, _ := http.NewRequest("GET", server.URL+"/courses/"+createdCourse.IDCourse.String()+"/cursadas?open=true", nil)
	expandReq.Header.Set("HX-Request", "true")
	expandResp, err := regularClient.Do(expandReq)
	if err != nil {
		t.Fatalf("Error al desplegar cursadas de la materia: %v", err)
	}
	assertStatus(t, expandResp, http.StatusOK, "GET /courses/{id}/cursadas?open=true")
	bodyExpanded := readResponseBody(t, expandResp)
	assertContains(t, bodyExpanded, "Ocultar cursadas (1)", "Botón para ocultar cursadas desplegadas")
	assertContains(t, bodyExpanded, "2026", "Año de la cursada en la tabla desplegada")
	assertContains(t, bodyExpanded, "Abierta / Vigente", "Estado de la cursada en el DOM desplegado")
	assertContains(t, bodyExpanded, "Ver QR / Token", "Presencia del botón de QR y Token al desplegar")

	// Colapsar nuevamente
	collapseReq, _ := http.NewRequest("GET", server.URL+"/courses/"+createdCourse.IDCourse.String()+"/cursadas?open=false", nil)
	collapseReq.Header.Set("HX-Request", "true")
	collapseResp, err := regularClient.Do(collapseReq)
	if err != nil {
		t.Fatalf("Error al colapsar cursadas de la materia: %v", err)
	}
	assertStatus(t, collapseResp, http.StatusOK, "GET /courses/{id}/cursadas?open=false")
	bodyCollapsed := readResponseBody(t, collapseResp)
	assertContains(t, bodyCollapsed, "Ver cursadas (1)", "Botón restaurado tras colapsar")
	assertNotContains(t, bodyCollapsed, "Ver QR / Token", "Tabla oculta tras colapsar")
}

// -----------------------------------------------------------------------------
// 3. Recorrido de Generación de QR y Pantalla de Proyección
// -----------------------------------------------------------------------------
func TestHTTPSuite_3_QRCodeAndProjection(t *testing.T) {
	conn, queries := getTestDB(t)
	defer conn.Close()

	ctx := context.Background()
	prof := ensureProfessor(ctx, conn, queries)

	// Crear materia y cursada de prueba
	course, err := queries.CreateCourse(ctx, "Materia QR "+uuid.NewString()[:8])
	if err != nil {
		t.Fatalf("Error al crear materia: %v", err)
	}
	token := generateSubToken()
	semester, err := queries.CreateSemester(ctx, db.CreateSemesterParams{
		IDCourse:    course.IDCourse,
		ProfessorID: prof.IDUser,
		Year:        2026,
		SubToken:    token,
		EndDate:     time.Now().AddDate(0, 6, 0),
	})
	if err != nil {
		t.Fatalf("Error al crear cursada: %v", err)
	}

	handler := setupRoutes(conn, queries)
	server := httptest.NewServer(handler)
	defer server.Close()
	client := server.Client()

	// 1. Validar endpoint de imagen PNG del QR (GET /qr/{token})
	qrResp, err := client.Get(server.URL + "/qr/" + token)
	if err != nil {
		t.Fatalf("Error en GET /qr/%s: %v", token, err)
	}
	assertStatus(t, qrResp, http.StatusOK, "GET /qr/{token}")

	contentType := qrResp.Header.Get("Content-Type")
	if contentType != "image/png" {
		t.Fatalf("Content-Type incorrecto en /qr/%s: obtenido %q, esperado %q", token, contentType, "image/png")
	}

	pngBytes, err := io.ReadAll(qrResp.Body)
	qrResp.Body.Close()
	if err != nil {
		t.Fatalf("Error al leer imagen PNG: %v", err)
	}
	if len(pngBytes) < 8 || string(pngBytes[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Fatalf("La respuesta no contiene los magic bytes de una imagen PNG válida")
	}

	// 2. Validar pantalla completa de proyección (GET /cursadas/{token}/qr)
	projResp, err := client.Get(server.URL + "/cursadas/" + token + "/qr")
	if err != nil {
		t.Fatalf("Error en GET /cursadas/%s/qr: %v", token, err)
	}
	assertStatus(t, projResp, http.StatusOK, "GET /cursadas/{token}/qr")

	projBody := readResponseBody(t, projResp)
	assertContains(t, projBody, "Código QR de Suscripción", "Título de pantalla de proyección")
	assertContains(t, projBody, course.Name, "Nombre de la materia en proyección")
	assertContains(t, projBody, "/qr/"+token, "Referencia a la imagen QR en el DOM")
	assertContains(t, projBody, "/subscribir/"+token, "Enlace directo de suscripción")

	_ = semester
}

// -----------------------------------------------------------------------------
// 4. Recorrido de Suscripción del Alumno (Portal Símil Login)
// -----------------------------------------------------------------------------
func TestHTTPSuite_4_StudentSubscriptionPortal(t *testing.T) {
	conn, queries := getTestDB(t)
	defer conn.Close()

	ctx := context.Background()
	prof := ensureProfessor(ctx, conn, queries)

	course, _ := queries.CreateCourse(ctx, "Algoritmos "+uuid.NewString()[:8])
	token := generateSubToken()
	_, err := queries.CreateSemester(ctx, db.CreateSemesterParams{
		IDCourse:    course.IDCourse,
		ProfessorID: prof.IDUser,
		Year:        2026,
		SubToken:    token,
		EndDate:     time.Now().AddDate(0, 5, 0),
	})
	if err != nil {
		t.Fatalf("Error al crear cursada de prueba: %v", err)
	}

	handler := setupRoutes(conn, queries)
	server := httptest.NewServer(handler)
	defer server.Close()
	client := server.Client()

	// 1. Acceso del alumno a la pantalla de suscripción (GET /subscribir/{token})
	getResp, err := client.Get(server.URL + "/subscribir/" + token)
	if err != nil {
		t.Fatalf("Error en GET /subscribir/%s: %v", token, err)
	}
	assertStatus(t, getResp, http.StatusOK, "GET /subscribir/{token}")
	body := readResponseBody(t, getResp)
	assertContains(t, body, "Portal de Suscripción de Alumnos", "Identidad del portal")
	assertContains(t, body, course.Name, "Materia informada al alumno")
	assertContains(t, body, "Suscribirme a la cursada", "Botón de suscripción")
	assertContains(t, body, "Ingresar con cuenta Google", "Opción Google anticipada")

	// 2. Envío con correo inválido (POST /subscribir/{token})
	invalidForm := url.Values{"email": {"email-sin-formato"}}
	invalidResp, err := client.PostForm(server.URL+"/subscribir/"+token, invalidForm)
	if err != nil {
		t.Fatalf("Error en POST /subscribir/%s: %v", token, err)
	}
	assertStatus(t, invalidResp, http.StatusOK, "POST /subscribir inválido")
	invalidBody := readResponseBody(t, invalidResp)
	assertContains(t, invalidBody, "correo electrónico válido", "Mensaje de error por email inválido")

	// 3. Envío con correo válido (POST /subscribir/{token})
	studentEmail := fmt.Sprintf("alumno.%s@universidad.edu.ar", uuid.NewString()[:6])
	validForm := url.Values{"email": {studentEmail}}
	validResp, err := client.PostForm(server.URL+"/subscribir/"+token, validForm)
	if err != nil {
		t.Fatalf("Error en POST /subscribir/%s con éxito: %v", token, err)
	}
	assertStatus(t, validResp, http.StatusOK, "POST /subscribir exitoso")
	validBody := readResponseBody(t, validResp)
	assertContains(t, validBody, "¡Suscripción confirmada!", "Mensaje de éxito")
	assertContains(t, validBody, studentEmail, "Email confirmado en el HTML")

	// 4. Verificación en el panel del profesor (GET /alumnos?email=...)
	adminResp, err := client.Get(server.URL + "/alumnos?email=" + url.QueryEscape(studentEmail))
	if err != nil {
		t.Fatalf("Error en GET /alumnos: %v", err)
	}
	assertStatus(t, adminResp, http.StatusOK, "GET /alumnos tras suscripción")
	adminBody := readResponseBody(t, adminResp)
	assertContains(t, adminBody, studentEmail, "Presencia del alumno en la gestión docente")
	assertContains(t, adminBody, course.Name, "Materia vinculada a la inscripción")
}

// -----------------------------------------------------------------------------
// 5. Recorrido de Publicación e Historial de Avisos
// -----------------------------------------------------------------------------
func TestHTTPSuite_5_NoticePublishingAndHistoryPRG(t *testing.T) {
	conn, queries := getTestDB(t)
	defer conn.Close()

	ctx := context.Background()
	prof := ensureProfessor(ctx, conn, queries)

	course, _ := queries.CreateCourse(ctx, "Base de Datos "+uuid.NewString()[:8])
	token := generateSubToken()
	semester, err := queries.CreateSemester(ctx, db.CreateSemesterParams{
		IDCourse:    course.IDCourse,
		ProfessorID: prof.IDUser,
		Year:        2026,
		SubToken:    token,
		EndDate:     time.Now().AddDate(0, 4, 0),
	})
	if err != nil {
		t.Fatalf("Error al crear cursada: %v", err)
	}

	handler := setupRoutes(conn, queries)
	server := httptest.NewServer(handler)
	defer server.Close()

	noRedirectClient := newNoRedirectClient()
	regularClient := server.Client()

	// 1. Publicar nuevo aviso (POST /avisos)
	noticeTitle := "Cambio de aula examen " + uuid.NewString()[:6]
	noticeMessage := "El examen del viernes se traslada al Aula Magna del Pabellón 2."
	formNotice := url.Values{
		"id_semester": {semester.IDSemester.String()},
		"title":       {noticeTitle},
		"message":     {noticeMessage},
	}

	postResp, err := noRedirectClient.PostForm(server.URL+"/avisos", formNotice)
	if err != nil {
		t.Fatalf("Error en POST /avisos: %v", err)
	}
	assertRedirect(t, postResp, "/publicar-aviso?enviado=1", "Publicación de aviso permanece en publicar-aviso (PRG)")
	postResp.Body.Close()

	// 1b. Verificar pantalla de confirmación en /publicar-aviso?enviado=1
	sentResp, err := regularClient.Get(server.URL + "/publicar-aviso?enviado=1")
	if err != nil {
		t.Fatalf("Error en GET /publicar-aviso?enviado=1: %v", err)
	}
	assertStatus(t, sentResp, http.StatusOK, "GET /publicar-aviso?enviado=1")
	sentBody := readResponseBody(t, sentResp)
	assertContains(t, sentBody, "Aviso enviado", "Mensaje de confirmación en pantalla")

	// 2. Consultar historial (GET /avisos) y verificar el aviso publicado en el DOM
	historyResp, err := regularClient.Get(server.URL + "/avisos")
	if err != nil {
		t.Fatalf("Error en GET /avisos: %v", err)
	}
	assertStatus(t, historyResp, http.StatusOK, "GET /avisos")
	historyBody := readResponseBody(t, historyResp)
	assertContains(t, historyBody, noticeTitle, "Título del aviso en el historial")
	assertContains(t, historyBody, noticeMessage, "Mensaje del aviso en el historial")
	assertContains(t, historyBody, course.Name, "Badge de la materia en el aviso")

	// 3. Filtrar avisos por cursada (GET /avisos?id_semester=...)
	filterResp, err := regularClient.Get(fmt.Sprintf("%s/avisos?id_semester=%s", server.URL, semester.IDSemester.String()))
	if err != nil {
		t.Fatalf("Error en GET /avisos filtrado: %v", err)
	}
	filterBody := readResponseBody(t, filterResp)
	assertContains(t, filterBody, noticeTitle, "Aviso presente en filtro de cursada")
}

// -----------------------------------------------------------------------------
// 6. Recorrido de Bajas y Borrado (Toggle Alumno y Borrado de Cursada / Materia)
// -----------------------------------------------------------------------------
func TestHTTPSuite_6_DeletionsAndToggle(t *testing.T) {
	conn, queries := getTestDB(t)
	defer conn.Close()

	ctx := context.Background()
	prof := ensureProfessor(ctx, conn, queries)

	course, _ := queries.CreateCourse(ctx, "Sistemas Operativos "+uuid.NewString()[:8])
	token := generateSubToken()
	semester, _ := queries.CreateSemester(ctx, db.CreateSemesterParams{
		IDCourse:    course.IDCourse,
		ProfessorID: prof.IDUser,
		Year:        2026,
		SubToken:    token,
		EndDate:     time.Now().AddDate(0, 3, 0),
	})
	studentEmail := "baja." + uuid.NewString()[:6] + "@test.com"
	inscription, err := queries.CreateInscription(ctx, db.CreateInscriptionParams{
		IDSemester: semester.IDSemester,
		Email:      studentEmail,
	})
	if err != nil {
		t.Fatalf("Error al inscribir alumno de prueba: %v", err)
	}

	handler := setupRoutes(conn, queries)
	server := httptest.NewServer(handler)
	defer server.Close()

	noRedirectClient := newNoRedirectClient()
	regularClient := server.Client()

	// 1. Desactivar alumno (POST /alumnos/toggle con active=false)
	toggleForm := url.Values{
		"id":     {inscription.IDInscription.String()},
		"active": {"false"},
		"email":  {studentEmail},
	}
	toggleResp, err := noRedirectClient.PostForm(server.URL+"/alumnos/toggle", toggleForm)
	if err != nil {
		t.Fatalf("Error en POST /alumnos/toggle: %v", err)
	}
	expectedLoc := "/alumnos?email=" + url.QueryEscape(studentEmail)
	assertRedirect(t, toggleResp, expectedLoc, "Toggle estado alumno PRG")
	toggleResp.Body.Close()

	// Verificar estado inactivo en el HTML
	statusResp, _ := regularClient.Get(server.URL + expectedLoc)
	statusBody := readResponseBody(t, statusResp)
	assertContains(t, statusBody, "Baja manual", "Estado inactivo del alumno")
	assertContains(t, statusBody, "Reactivar", "Botón para reactivar al alumno")

	// 2. Dar de baja definitiva al alumno (POST /alumnos/delete)
	deleteForm := url.Values{
		"id":    {inscription.IDInscription.String()},
		"email": {studentEmail},
	}
	delResp, err := noRedirectClient.PostForm(server.URL+"/alumnos/delete", deleteForm)
	if err != nil {
		t.Fatalf("Error en POST /alumnos/delete: %v", err)
	}
	assertRedirect(t, delResp, expectedLoc, "Baja de alumno PRG")
	delResp.Body.Close()

	delCheckResp, _ := regularClient.Get(server.URL + expectedLoc)
	delCheckBody := readResponseBody(t, delCheckResp)
	assertContains(t, delCheckBody, "No se encontraron inscripciones", "Confirmación de baja en el DOM")

	// 3. Eliminar la cursada asociada (POST /semesters/delete)
	delSemForm := url.Values{"id": {semester.IDSemester.String()}}
	delSemResp, err := noRedirectClient.PostForm(server.URL+"/semesters/delete", delSemForm)
	if err != nil {
		t.Fatalf("Error en POST /semesters/delete: %v", err)
	}
	assertRedirect(t, delSemResp, "/", "Borrado de cursada PRG")
	delSemResp.Body.Close()

	// 4. Eliminar la materia ya sin cursadas asociadas (POST /courses/delete)
	delCourseForm := url.Values{"id": {course.IDCourse.String()}}
	delCourseResp, err := noRedirectClient.PostForm(server.URL+"/courses/delete", delCourseForm)
	if err != nil {
		t.Fatalf("Error en POST /courses/delete: %v", err)
	}
	assertRedirect(t, delCourseResp, "/", "Borrado de materia PRG")
	delCourseResp.Body.Close()

	// Verificar que la materia ya no figure en el panel principal
	homeResp, _ := regularClient.Get(server.URL + "/")
	homeBody := readResponseBody(t, homeResp)
	assertNotContains(t, homeBody, course.Name, "Materia eliminada ausente del panel")
}

// -----------------------------------------------------------------------------
// 7. Recorrido HTMX: Navegación SPA parcial con swap OOB en la barra de navegación
// -----------------------------------------------------------------------------
func TestHTTPSuite_7_HTMX_TabNavigation_And_OOB(t *testing.T) {
	conn, queries := getTestDB(t)
	defer conn.Close()

	handler := setupRoutes(conn, queries)
	server := httptest.NewServer(handler)
	defer server.Close()

	client := server.Client()

	req, err := http.NewRequest("GET", server.URL+"/avisos", nil)
	if err != nil {
		t.Fatalf("Error al crear request HTMX: %v", err)
	}
	req.Header.Set("HX-Request", "true")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Error en request HTMX GET /avisos: %v", err)
	}
	assertStatus(t, resp, http.StatusOK, "GET /avisos vía HTMX")

	body := readResponseBody(t, resp)

	// No debe devolver el documento HTML completo
	assertNotContains(t, body, "<!DOCTYPE html>", "No debe incluir DOCTYPE en respuesta parcial")
	assertNotContains(t, body, "<html", "No debe incluir etiqueta html en respuesta parcial")

	// Debe devolver el contenido de avisos
	assertContains(t, body, "Historial de Avisos", "Contenido del apartado de avisos")

	// Debe incluir la barra de navegación con hx-swap-oob="true" para iluminar la pestaña activa
	assertContains(t, body, `hx-swap-oob="true"`, "Presencia de swap OOB en barra de navegación")
	assertContains(t, body, `class="top-nav-link active"`, "Pestaña activa iluminada vía OOB")
}

// -----------------------------------------------------------------------------
// 8. Recorrido HTMX: Creación de Materia con actualización Out-of-Band del formulario de Cursadas
// -----------------------------------------------------------------------------
func TestHTTPSuite_8_HTMX_CourseCreation_With_OOB_SemesterSelect(t *testing.T) {
	conn, queries := getTestDB(t)
	defer conn.Close()

	handler := setupRoutes(conn, queries)
	server := httptest.NewServer(handler)
	defer server.Close()

	client := server.Client()

	uniqueCourse := "Materia HTMX " + uuid.NewString()[:8]
	form := url.Values{"name": {uniqueCourse}}

	req, err := http.NewRequest("POST", server.URL+"/courses", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("Error al crear request HTMX POST: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Error en POST /courses vía HTMX: %v", err)
	}
	assertStatus(t, resp, http.StatusOK, "POST /courses vía HTMX devuelve 200 con fragmentos")

	body := readResponseBody(t, resp)

	// 1. Debe contener la nueva materia en la lista principal
	assertContains(t, body, uniqueCourse, "Nueva materia presente en la lista")

	// 2. Debe contener el formulario de cursada actualizado vía OOB con el nuevo select
	assertContains(t, body, `id="new-semester-form-container"`, "Contenedor de nueva cursada")
	assertContains(t, body, `hx-swap-oob="true"`, "Atributo hx-swap-oob en formulario de cursada")
	assertContains(t, body, uniqueCourse, "La nueva materia figura en las opciones del selector de cursada OOB")
}

// -----------------------------------------------------------------------------
// 9. Recorrido HTMX: Búsqueda dinámica de alumnos y toggle in-situ de fila
// -----------------------------------------------------------------------------
func TestHTTPSuite_9_HTMX_StudentLiveSearch_And_RowToggle(t *testing.T) {
	conn, queries := getTestDB(t)
	defer conn.Close()

	ctx := context.Background()
	prof := ensureProfessor(ctx, conn, queries)

	course, _ := queries.CreateCourse(ctx, "Redes "+uuid.NewString()[:8])
	token := generateSubToken()
	semester, _ := queries.CreateSemester(ctx, db.CreateSemesterParams{
		IDCourse:    course.IDCourse,
		ProfessorID: prof.IDUser,
		Year:        2026,
		SubToken:    token,
		EndDate:     time.Now().AddDate(0, 3, 0),
	})
	studentEmail := "htmx." + uuid.NewString()[:6] + "@alumno.edu.ar"
	inscription, _ := queries.CreateInscription(ctx, db.CreateInscriptionParams{
		IDSemester: semester.IDSemester,
		Email:      studentEmail,
	})

	handler := setupRoutes(conn, queries)
	server := httptest.NewServer(handler)
	defer server.Close()

	client := server.Client()

	// 1. Búsqueda en tiempo real con HX-Target="students-results-container"
	searchReq, _ := http.NewRequest("GET", server.URL+"/alumnos?email="+url.QueryEscape(studentEmail), nil)
	searchReq.Header.Set("HX-Request", "true")
	searchReq.Header.Set("HX-Target", "students-results-container")

	searchResp, err := client.Do(searchReq)
	if err != nil {
		t.Fatalf("Error en búsqueda HTMX: %v", err)
	}
	assertStatus(t, searchResp, http.StatusOK, "Búsqueda en tiempo real HTMX")
	searchBody := readResponseBody(t, searchResp)
	assertContains(t, searchBody, studentEmail, "Alumno encontrado en resultados parciales")
	assertNotContains(t, searchBody, "<!DOCTYPE html>", "No debe incluir página completa")

	// 2. Alternar estado in-situ con HX-Request (devuelve únicamente la fila <tr>)
	toggleForm := url.Values{
		"id":     {inscription.IDInscription.String()},
		"active": {"false"},
		"email":  {studentEmail},
	}
	toggleReq, _ := http.NewRequest("POST", server.URL+"/alumnos/toggle", strings.NewReader(toggleForm.Encode()))
	toggleReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	toggleReq.Header.Set("HX-Request", "true")

	toggleResp, err := client.Do(toggleReq)
	if err != nil {
		t.Fatalf("Error en toggle HTMX: %v", err)
	}
	assertStatus(t, toggleResp, http.StatusOK, "Toggle de alumno vía HTMX")
	toggleBody := readResponseBody(t, toggleResp)

	// La respuesta debe ser únicamente el elemento <tr>
	assertContains(t, toggleBody, "student-row-"+inscription.IDInscription.String(), "ID de fila del alumno")
	assertContains(t, toggleBody, "Baja manual", "Badge de baja manual actualizado in-situ")
	assertContains(t, toggleBody, "Reactivar", "Botón reactivar actualizado in-situ")
	assertNotContains(t, toggleBody, "<table", "No debe devolver la tabla entera, solo la fila")
}

// -----------------------------------------------------------------------------
// 10. Recorrido HTMX: Desplegable dependiente Materia -> Cursada y envío in-situ de Aviso
// -----------------------------------------------------------------------------
func TestHTTPSuite_10_HTMX_PublishNotice_DependentSelect_And_InPlaceSubmit(t *testing.T) {
	conn, queries := getTestDB(t)
	defer conn.Close()

	ctx := context.Background()
	prof := ensureProfessor(ctx, conn, queries)

	courseA, _ := queries.CreateCourse(ctx, "Física "+uuid.NewString()[:8])
	courseB, _ := queries.CreateCourse(ctx, "Química "+uuid.NewString()[:8])

	semA1, _ := queries.CreateSemester(ctx, db.CreateSemesterParams{
		IDCourse:    courseA.IDCourse,
		ProfessorID: prof.IDUser,
		Year:        2026,
		SubToken:    generateSubToken(),
		EndDate:     time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC),
	})
	semA2, _ := queries.CreateSemester(ctx, db.CreateSemesterParams{
		IDCourse:    courseA.IDCourse,
		ProfessorID: prof.IDUser,
		Year:        2027,
		SubToken:    generateSubToken(),
		EndDate:     time.Date(2027, 12, 15, 0, 0, 0, 0, time.UTC),
	})
	semB1, _ := queries.CreateSemester(ctx, db.CreateSemesterParams{
		IDCourse:    courseB.IDCourse,
		ProfessorID: prof.IDUser,
		Year:        2028,
		SubToken:    generateSubToken(),
		EndDate:     time.Date(2028, 12, 15, 0, 0, 0, 0, time.UTC),
	})

	handler := setupRoutes(conn, queries)
	server := httptest.NewServer(handler)
	defer server.Close()

	client := server.Client()

	// 1. GET /publicar-aviso muestra cada materia una sola vez y no pre-carga el select de cursadas
	pageResp, err := client.Get(server.URL + "/publicar-aviso")
	if err != nil {
		t.Fatalf("Error en GET /publicar-aviso: %v", err)
	}
	assertStatus(t, pageResp, http.StatusOK, "GET /publicar-aviso")
	pageBody := readResponseBody(t, pageResp)
	assertContains(t, pageBody, `id="notice-course"`, "Selector de materia presente")
	assertContains(t, pageBody, courseA.Name, "Materia A listada")
	assertContains(t, pageBody, courseB.Name, "Materia B listada")
	if strings.Count(pageBody, courseA.Name) != 1 {
		t.Fatalf("La materia %q apareció %d veces en /publicar-aviso; debe figurar solo una vez", courseA.Name, strings.Count(pageBody, courseA.Name))
	}
	assertNotContains(t, pageBody, `id="notice-semester"`, "El selector de cursadas no debe mostrarse antes de elegir materia")

	// 2. GET /publicar-aviso/cursadas?course_id=... devuelve solo las cursadas de la materia elegida
	semReq, _ := http.NewRequest("GET", server.URL+"/publicar-aviso/cursadas?course_id="+courseA.IDCourse.String(), nil)
	semReq.Header.Set("HX-Request", "true")
	semResp, err := client.Do(semReq)
	if err != nil {
		t.Fatalf("Error en GET /publicar-aviso/cursadas: %v", err)
	}
	assertStatus(t, semResp, http.StatusOK, "GET /publicar-aviso/cursadas")
	semBody := readResponseBody(t, semResp)
	assertContains(t, semBody, `id="notice-semester"`, "Selector de cursadas renderizado")
	assertContains(t, semBody, semA1.IDSemester.String(), "Cursada 1 de Materia A presente")
	assertContains(t, semBody, semA2.IDSemester.String(), "Cursada 2 de Materia A presente")
	assertNotContains(t, semBody, semB1.IDSemester.String(), "Cursada de Materia B no debe aparecer para Materia A")

	// 3. POST /avisos vía HTMX publica el aviso, permanece en la página, muestra 'Aviso enviado' y vacía el formulario
	noticeTitle := "Aviso HTMX " + uuid.NewString()[:6]
	noticeMessage := "Contenido publicado sin salir de /publicar-aviso."
	form := url.Values{
		"course_id":   {courseA.IDCourse.String()},
		"id_semester": {semA1.IDSemester.String()},
		"title":       {noticeTitle},
		"message":     {noticeMessage},
	}
	postReq, _ := http.NewRequest("POST", server.URL+"/avisos", strings.NewReader(form.Encode()))
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.Header.Set("HX-Request", "true")

	postResp, err := client.Do(postReq)
	if err != nil {
		t.Fatalf("Error en POST /avisos vía HTMX: %v", err)
	}
	assertStatus(t, postResp, http.StatusOK, "POST /avisos vía HTMX devuelve 200")
	if loc := postResp.Header.Get("HX-Location"); loc != "" {
		t.Fatalf("No debe enviar HX-Location al publicar aviso, pero se obtuvo: %q", loc)
	}
	postBody := readResponseBody(t, postResp)
	assertContains(t, postBody, "Aviso enviado", "Mensaje de confirmación 'Aviso enviado'")
	assertContains(t, postBody, "-- Selecciona una materia --", "Formulario reiniciado con selector de materia por defecto")
	assertNotContains(t, postBody, noticeTitle, "El formulario debe quedar vacío tras publicar")
	assertNotContains(t, postBody, noticeMessage, "El mensaje del formulario debe quedar vacío tras publicar")
	assertNotContains(t, postBody, `id="notice-semester"`, "El selector dependiente de cursadas debe quedar vacío tras reiniciar")

	// 4. Confirmar que el aviso se persistió correctamente en el historial
	histResp, _ := client.Get(server.URL + "/avisos")
	histBody := readResponseBody(t, histResp)
	assertContains(t, histBody, noticeTitle, "Aviso publicado persistido en el historial")
}

