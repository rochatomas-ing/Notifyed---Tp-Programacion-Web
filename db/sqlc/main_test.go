package db

import (
	"context"
	"database/sql"
	"log"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

var testDB *sql.DB

// TestMain corre una sola vez antes que todos los tests del paquete.
// Abre la conexión que después comparten todos.
func TestMain(m *testing.M) {
	dsn := os.Getenv("TEST_DB_URL")
	if dsn == "" {
		dsn = "postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable"
	}

	var err error
	testDB, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("no se pudo abrir la conexión: %v", err)
	}
	defer testDB.Close()

	if err := testDB.Ping(); err != nil {
		log.Fatalf("la base no responde en %s: %v\n"+
			"¿está levantado el contenedor? ¿coincide TEST_DB_URL con el .env?", dsn, err)
	}

	os.Exit(m.Run())
}

// withTx abre una transacción para el test y la revierte al terminar.
// Así cada test arranca con la base en el mismo estado y no dependen
// del orden en que se ejecuten.
func withTx(t *testing.T) *Queries {
	t.Helper()

	tx, err := testDB.Begin()
	if err != nil {
		t.Fatalf("no se pudo abrir la transacción: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback() })

	return New(tx)
}

// checkErr compara el error obtenido contra lo esperado.
// Devuelve true si el test debe terminar acá (se esperaba error y hubo error).
func checkErr(t *testing.T, err error, wantErr bool) bool {
	t.Helper()
	if (err != nil) != wantErr {
		t.Fatalf("error = %v, wantErr = %v", err, wantErr)
	}
	return wantErr
}

// ---------------------------------------------------------------------------
// Helpers de setup.
// Preparan el escenario que un test necesita. Nunca ejecutan la query que el
// test está probando: eso va en el cuerpo del test, a la vista.
// ---------------------------------------------------------------------------

// emailUnico evita chocar con la UNIQUE de users.email entre tests.
func emailUnico(prefijo string) string {
	return prefijo + "-" + uuid.NewString() + "@test.com"
}

func crearProfesor(t *testing.T, q *Queries) User {
	t.Helper()

	u, err := q.CreateUser(context.Background(), CreateUserParams{
		Fullname:     "Profesor Test",
		Email:        emailUnico("profe"),
		PasswordHash: "$2a$10$hashDePruebaNoEsUnHashReal",
	})
	if err != nil {
		t.Fatalf("setup: no se pudo crear el profesor: %v", err)
	}
	return u
}

func crearMateria(t *testing.T, q *Queries) Course {
	t.Helper()

	c, err := q.CreateCourse(context.Background(), "Materia "+uuid.NewString()[:8])
	if err != nil {
		t.Fatalf("setup: no se pudo crear la materia: %v", err)
	}
	return c
}

// crearCursada arma la cadena completa: profesor, materia y cursada.
// finDeCursada indica cuándo vence; pasá una fecha pasada para probar
// el cierre automático.
func crearCursada(t *testing.T, q *Queries, finDeCursada time.Time) Semester {
	t.Helper()

	prof := crearProfesor(t, q)
	materia := crearMateria(t, q)

	s, err := q.CreateSemester(context.Background(), CreateSemesterParams{
		IDCourse:    materia.IDCourse,
		ProfessorID: prof.IDUser,
		Year:        int32(finDeCursada.Year()),
		SubToken:    uuid.NewString()[:32],
		EndDate:     finDeCursada,
	})
	if err != nil {
		t.Fatalf("setup: no se pudo crear la cursada: %v", err)
	}
	return s
}

// crearCursadaVigente es el caso normal: una cursada que todavía no terminó.
func crearCursadaVigente(t *testing.T, q *Queries) Semester {
	t.Helper()
	return crearCursada(t, q, time.Now().AddDate(0, 6, 0))
}
