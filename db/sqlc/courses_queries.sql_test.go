package db

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// CreateCourse
// ---------------------------------------------------------------------------

func TestCreateCourse(t *testing.T) {
	casos := []struct {
		nombre     string
		courseName string
		wantErr    bool
	}{
		{"materia válida", "Análisis Matemático I", false},
		{"nombre con acentos y ñ", "Diseño de Sistemas", false},
		{"nombre en el límite de 40 chars", strings.Repeat("a", 40), false},
		{"nombre más largo que 40 chars", strings.Repeat("a", 41), true},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			q := withTx(t)

			got, err := q.CreateCourse(context.Background(), c.courseName)
			if checkErr(t, err, c.wantErr) {
				return
			}

			if got.Name != c.courseName {
				t.Errorf("Name = %q, want %q", got.Name, c.courseName)
			}
			if got.IDCourse == uuid.Nil {
				t.Error("IDCourse quedó en el UUID cero; revisar DEFAULT gen_random_uuid()")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// GetCourseById
// ---------------------------------------------------------------------------

func TestGetCourseById(t *testing.T) {
	t.Run("devuelve la materia creada", func(t *testing.T) {
		q := withTx(t)
		ctx := context.Background()

		creada := crearMateria(t, q)

		got, err := q.GetCourseById(ctx, creada.IDCourse)
		if err != nil {
			t.Fatalf("GetCourseById devolvió error: %v", err)
		}

		if got.IDCourse != creada.IDCourse {
			t.Errorf("IDCourse = %v, want %v", got.IDCourse, creada.IDCourse)
		}
		if got.Name != creada.Name {
			t.Errorf("Name = %q, want %q", got.Name, creada.Name)
		}
	})

	t.Run("id inexistente devuelve ErrNoRows", func(t *testing.T) {
		q := withTx(t)

		_, err := q.GetCourseById(context.Background(), uuid.New())
		if !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("err = %v, want sql.ErrNoRows", err)
		}
	})
}

// ---------------------------------------------------------------------------
// GetCourseByName
// ---------------------------------------------------------------------------

func TestGetCourseByName(t *testing.T) {
	t.Run("devuelve la materia buscada por nombre", func(t *testing.T) {
		q := withTx(t)
		ctx := context.Background()

		creada, err := q.CreateCourse(ctx, "Física II")
		if err != nil {
			t.Fatalf("setup: %v", err)
		}

		got, err := q.GetCourseByName(ctx, "Física II")
		if err != nil {
			t.Fatalf("GetCourseByName devolvió error: %v", err)
		}
		if got.IDCourse != creada.IDCourse {
			t.Errorf("IDCourse = %v, want %v", got.IDCourse, creada.IDCourse)
		}
	})

	t.Run("nombre inexistente devuelve ErrNoRows", func(t *testing.T) {
		q := withTx(t)

		_, err := q.GetCourseByName(context.Background(), "Materia Que No Existe")
		if !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("err = %v, want sql.ErrNoRows", err)
		}
	})
}

// ---------------------------------------------------------------------------
// ListCourses
// ---------------------------------------------------------------------------

func TestListCourses(t *testing.T) {
	t.Run("sin materias devuelve lista vacía", func(t *testing.T) {
		q := withTx(t)

		got, err := q.ListCourses(context.Background())
		if err != nil {
			t.Fatalf("ListCourses devolvió error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("se esperaba lista vacía, hay %d materias", len(got))
		}
	})

	t.Run("devuelve todas ordenadas por nombre", func(t *testing.T) {
		q := withTx(t)
		ctx := context.Background()

		// Se crean desordenadas a propósito.
		for _, nombre := range []string{"Zoología", "Álgebra", "Métodos Numéricos"} {
			if _, err := q.CreateCourse(ctx, nombre); err != nil {
				t.Fatalf("setup %q: %v", nombre, err)
			}
		}

		got, err := q.ListCourses(ctx)
		if err != nil {
			t.Fatalf("ListCourses devolvió error: %v", err)
		}
		if len(got) != 3 {
			t.Fatalf("se devolvieron %d materias, want 3", len(got))
		}

		for i := 1; i < len(got); i++ {
			if got[i].Name < got[i-1].Name {
				t.Errorf("el orden por nombre está roto: %q viene después de %q",
					got[i].Name, got[i-1].Name)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// UpdateCourseName
// ---------------------------------------------------------------------------

func TestUpdateCourseName(t *testing.T) {
	t.Run("actualiza el nombre", func(t *testing.T) {
		q := withTx(t)
		ctx := context.Background()

		creada := crearMateria(t, q)

		got, err := q.UpdateCourseName(ctx, UpdateCourseNameParams{
			IDCourse: creada.IDCourse,
			Name:     "Nombre Actualizado",
		})
		if err != nil {
			t.Fatalf("UpdateCourseName devolvió error: %v", err)
		}

		if got.Name != "Nombre Actualizado" {
			t.Errorf("Name = %q, want %q", got.Name, "Nombre Actualizado")
		}
		if got.IDCourse != creada.IDCourse {
			t.Errorf("el update cambió el id: %v, want %v", got.IDCourse, creada.IDCourse)
		}
	})

	t.Run("id inexistente devuelve ErrNoRows", func(t *testing.T) {
		q := withTx(t)

		_, err := q.UpdateCourseName(context.Background(), UpdateCourseNameParams{
			IDCourse: uuid.New(),
			Name:     "No existe",
		})
		if !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("err = %v, want sql.ErrNoRows", err)
		}
	})
}

// ---------------------------------------------------------------------------
// DeleteCourse
// ---------------------------------------------------------------------------

func TestDeleteCourse(t *testing.T) {
	t.Run("borra la materia existente", func(t *testing.T) {
		q := withTx(t)
		ctx := context.Background()

		creada := crearMateria(t, q)

		filas, err := q.DeleteCourse(ctx, creada.IDCourse)
		if err != nil {
			t.Fatalf("DeleteCourse devolvió error: %v", err)
		}
		if filas != 1 {
			t.Errorf("filas afectadas = %d, want 1", filas)
		}

		if _, err := q.GetCourseById(ctx, creada.IDCourse); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("la materia sigue existiendo tras el delete (err = %v)", err)
		}
	})

	t.Run("id inexistente afecta cero filas", func(t *testing.T) {
		q := withTx(t)

		filas, err := q.DeleteCourse(context.Background(), uuid.New())
		if err != nil {
			t.Fatalf("se esperaba nil, se obtuvo: %v", err)
		}
		if filas != 0 {
			t.Errorf("filas afectadas = %d, want 0", filas)
		}
	})

	// La FK semester_course tiene que impedir borrar una materia con cursadas.
	t.Run("no se puede borrar una materia con cursadas", func(t *testing.T) {
		q := withTx(t)
		ctx := context.Background()

		cursada := crearCursadaVigente(t, q)

		_, err := q.DeleteCourse(ctx, cursada.IDCourse)
		if err == nil {
			t.Fatal("se borró una materia que tenía cursadas; revisar la FK semester_course")
		}
	})
}
