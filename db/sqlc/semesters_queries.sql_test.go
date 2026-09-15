package db

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// CreateSemester
// ---------------------------------------------------------------------------

func TestCreateSemester(t *testing.T) {
	casos := []struct {
		nombre string
		year   int32
		// badCourse / badProfessor fuerzan un UUID inexistente para probar las FK.
		badCourse    bool
		badProfessor bool
		wantErr      bool
	}{
		{nombre: "cursada válida", year: 2026},
		{nombre: "año anterior", year: 2025},
		{nombre: "materia inexistente", year: 2026, badCourse: true, wantErr: true},
		{nombre: "profesor inexistente", year: 2026, badProfessor: true, wantErr: true},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			q := withTx(t)
			ctx := context.Background()

			prof := crearProfesor(t, q)
			materia := crearMateria(t, q)

			courseID := materia.IDCourse
			if c.badCourse {
				courseID = uuid.New()
			}
			professorID := prof.IDUser
			if c.badProfessor {
				professorID = uuid.New()
			}

			token := uuid.NewString()[:32]
			endDate := time.Now().AddDate(0, 6, 0)

			got, err := q.CreateSemester(ctx, CreateSemesterParams{
				IDCourse:    courseID,
				ProfessorID: professorID,
				Year:        c.year,
				SubToken:    token,
				EndDate:     endDate,
			})
			if checkErr(t, err, c.wantErr) {
				return
			}

			if got.IDSemester == uuid.Nil {
				t.Error("IDSemester quedó en el UUID cero")
			}
			if got.Year != c.year {
				t.Errorf("Year = %d, want %d", got.Year, c.year)
			}
			if got.SubToken != token {
				t.Errorf("SubToken = %q, want %q", got.SubToken, token)
			}
			if got.IDCourse != courseID {
				t.Errorf("IDCourse = %v, want %v", got.IDCourse, courseID)
			}
			if got.ProfessorID != professorID {
				t.Errorf("ProfessorID = %v, want %v", got.ProfessorID, professorID)
			}
		})
	}
}

// Necesita dos llamadas, así que va fuera de la tabla.
func TestCreateSemesterTokenDuplicado(t *testing.T) {
	q := withTx(t)
	ctx := context.Background()

	prof := crearProfesor(t, q)
	materia := crearMateria(t, q)
	token := uuid.NewString()[:32]

	params := CreateSemesterParams{
		IDCourse:    materia.IDCourse,
		ProfessorID: prof.IDUser,
		Year:        2026,
		SubToken:    token,
		EndDate:     time.Now().AddDate(0, 6, 0),
	}

	if _, err := q.CreateSemester(ctx, params); err != nil {
		t.Fatalf("la primera cursada falló: %v", err)
	}

	if _, err := q.CreateSemester(ctx, params); err == nil {
		t.Fatal("se crearon dos cursadas con el mismo token; revisar la UNIQUE toke_sub_uk")
	}
}

// ---------------------------------------------------------------------------
// GetSemesterById
// ---------------------------------------------------------------------------

func TestGetSemesterById(t *testing.T) {
	t.Run("devuelve la cursada creada", func(t *testing.T) {
		q := withTx(t)
		ctx := context.Background()

		creada := crearCursadaVigente(t, q)

		got, err := q.GetSemesterById(ctx, creada.IDSemester)
		if err != nil {
			t.Fatalf("GetSemesterById devolvió error: %v", err)
		}

		if got.IDSemester != creada.IDSemester {
			t.Errorf("IDSemester = %v, want %v", got.IDSemester, creada.IDSemester)
		}
		if got.SubToken != creada.SubToken {
			t.Errorf("SubToken = %q, want %q", got.SubToken, creada.SubToken)
		}
	})

	t.Run("id inexistente devuelve ErrNoRows", func(t *testing.T) {
		q := withTx(t)

		_, err := q.GetSemesterById(context.Background(), uuid.New())
		if !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("err = %v, want sql.ErrNoRows", err)
		}
	})
}

// ---------------------------------------------------------------------------
// GetSemesterByToken — la query que resuelve el QR
// ---------------------------------------------------------------------------

func TestGetSemesterByToken(t *testing.T) {
	t.Run("devuelve la cursada con el nombre de la materia", func(t *testing.T) {
		q := withTx(t)
		ctx := context.Background()

		creada := crearCursadaVigente(t, q)

		got, err := q.GetSemesterByToken(ctx, creada.SubToken)
		if err != nil {
			t.Fatalf("GetSemesterByToken devolvió error: %v", err)
		}

		if got.IDSemester != creada.IDSemester {
			t.Errorf("IDSemester = %v, want %v", got.IDSemester, creada.IDSemester)
		}
		if got.CourseName == "" {
			t.Error("CourseName vino vacío; revisar el JOIN con courses")
		}
	})

	// El cierre automático: una cursada vencida se marca como cerrada.
	t.Run("cursada vigente tiene is_open en true", func(t *testing.T) {
		q := withTx(t)
		ctx := context.Background()

		creada := crearCursadaVigente(t, q)

		got, err := q.GetSemesterByToken(ctx, creada.SubToken)
		if err != nil {
			t.Fatalf("GetSemesterByToken devolvió error: %v", err)
		}
		if !got.IsOpen {
			t.Error("IsOpen = false para una cursada que termina en 6 meses")
		}
	})

	t.Run("cursada vencida tiene is_open en false", func(t *testing.T) {
		q := withTx(t)
		ctx := context.Background()

		// Terminó hace un mes.
		vencida := crearCursada(t, q, time.Now().AddDate(0, -1, 0))

		got, err := q.GetSemesterByToken(ctx, vencida.SubToken)
		if err != nil {
			t.Fatalf("GetSemesterByToken devolvió error: %v", err)
		}
		if got.IsOpen {
			t.Error("IsOpen = true para una cursada que ya terminó")
		}
	})

	t.Run("token inexistente devuelve ErrNoRows", func(t *testing.T) {
		q := withTx(t)

		_, err := q.GetSemesterByToken(context.Background(), "token-que-no-existe-000000000000")
		if !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("err = %v, want sql.ErrNoRows", err)
		}
	})
}

// ---------------------------------------------------------------------------
// ListSemestersByProfessor
// ---------------------------------------------------------------------------

func TestListSemestersByProfessor(t *testing.T) {
	t.Run("profesor sin cursadas devuelve lista vacía", func(t *testing.T) {
		q := withTx(t)

		prof := crearProfesor(t, q)

		got, err := q.ListSemestersByProfessor(context.Background(), prof.IDUser)
		if err != nil {
			t.Fatalf("ListSemestersByProfessor devolvió error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("se esperaba lista vacía, hay %d cursadas", len(got))
		}
	})

	t.Run("devuelve solo las cursadas de ese profesor", func(t *testing.T) {
		q := withTx(t)
		ctx := context.Background()

		prof := crearProfesor(t, q)
		materia := crearMateria(t, q)

		// Dos cursadas del profesor.
		for i := 0; i < 2; i++ {
			if _, err := q.CreateSemester(ctx, CreateSemesterParams{
				IDCourse:    materia.IDCourse,
				ProfessorID: prof.IDUser,
				Year:        int32(2025 + i),
				SubToken:    uuid.NewString()[:32],
				EndDate:     time.Now().AddDate(0, 6, 0),
			}); err != nil {
				t.Fatalf("setup: %v", err)
			}
		}

		// Una cursada de otro profesor, que no debe aparecer.
		crearCursadaVigente(t, q)

		got, err := q.ListSemestersByProfessor(ctx, prof.IDUser)
		if err != nil {
			t.Fatalf("ListSemestersByProfessor devolvió error: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("se devolvieron %d cursadas, want 2", len(got))
		}
		for _, s := range got {
			if s.CourseName == "" {
				t.Error("CourseName vino vacío; revisar el JOIN con courses")
			}
		}
	})

	t.Run("cuenta los suscriptos activos", func(t *testing.T) {
		q := withTx(t)
		ctx := context.Background()

		cursada := crearCursadaVigente(t, q)

		sem, err := q.GetSemesterById(ctx, cursada.IDSemester)
		if err != nil {
			t.Fatalf("setup: %v", err)
		}

		for i := 0; i < 3; i++ {
			if _, err := q.CreateInscription(ctx, CreateInscriptionParams{
				IDSemester: cursada.IDSemester,
				Email:      emailUnico("alumno"),
			}); err != nil {
				t.Fatalf("setup inscripción: %v", err)
			}
		}

		got, err := q.ListSemestersByProfessor(ctx, sem.ProfessorID)
		if err != nil {
			t.Fatalf("ListSemestersByProfessor devolvió error: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("se devolvieron %d cursadas, want 1", len(got))
		}
		if got[0].ActiveSubscribers != 3 {
			t.Errorf("ActiveSubscribers = %d, want 3", got[0].ActiveSubscribers)
		}
	})
}

// ---------------------------------------------------------------------------
// ListSemestersByCourse
// ---------------------------------------------------------------------------

func TestListSemestersByCourse(t *testing.T) {
	t.Run("devuelve las cursadas de la materia", func(t *testing.T) {
		q := withTx(t)
		ctx := context.Background()

		prof := crearProfesor(t, q)
		materia := crearMateria(t, q)

		for i := 0; i < 2; i++ {
			if _, err := q.CreateSemester(ctx, CreateSemesterParams{
				IDCourse:    materia.IDCourse,
				ProfessorID: prof.IDUser,
				Year:        int32(2025 + i),
				SubToken:    uuid.NewString()[:32],
				EndDate:     time.Now().AddDate(0, 6, 0),
			}); err != nil {
				t.Fatalf("setup: %v", err)
			}
		}

		got, err := q.ListSemestersByCourse(ctx, materia.IDCourse)
		if err != nil {
			t.Fatalf("ListSemestersByCourse devolvió error: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("se devolvieron %d cursadas, want 2", len(got))
		}
		// Ordenadas por año descendente.
		if got[0].Year < got[1].Year {
			t.Errorf("el orden por año está roto: %d antes de %d", got[0].Year, got[1].Year)
		}
	})

	t.Run("materia sin cursadas devuelve lista vacía", func(t *testing.T) {
		q := withTx(t)

		materia := crearMateria(t, q)

		got, err := q.ListSemestersByCourse(context.Background(), materia.IDCourse)
		if err != nil {
			t.Fatalf("ListSemestersByCourse devolvió error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("se esperaba lista vacía, hay %d cursadas", len(got))
		}
	})
}

// ---------------------------------------------------------------------------
// UpdateSemesterEndDate
// ---------------------------------------------------------------------------

func TestUpdateSemesterEndDate(t *testing.T) {
	t.Run("actualiza la fecha de fin", func(t *testing.T) {
		q := withTx(t)
		ctx := context.Background()

		creada := crearCursadaVigente(t, q)
		nuevaFecha := time.Now().AddDate(1, 0, 0)

		got, err := q.UpdateSemesterEndDate(ctx, UpdateSemesterEndDateParams{
			IDSemester: creada.IDSemester,
			EndDate:    nuevaFecha,
		})
		if err != nil {
			t.Fatalf("UpdateSemesterEndDate devolvió error: %v", err)
		}

		if got.EndDate.Format("2006-01-02") != nuevaFecha.Format("2006-01-02") {
			t.Errorf("EndDate = %v, want %v", got.EndDate.Format("2006-01-02"), nuevaFecha.Format("2006-01-02"))
		}
		if got.SubToken != creada.SubToken {
			t.Error("el update pisó el sub_token")
		}
	})

	// Cerrar una cursada antes de tiempo: poner la fecha en el pasado.
	t.Run("poner la fecha en el pasado cierra la cursada", func(t *testing.T) {
		q := withTx(t)
		ctx := context.Background()

		creada := crearCursadaVigente(t, q)

		if _, err := q.UpdateSemesterEndDate(ctx, UpdateSemesterEndDateParams{
			IDSemester: creada.IDSemester,
			EndDate:    time.Now().AddDate(0, 0, -1),
		}); err != nil {
			t.Fatalf("UpdateSemesterEndDate devolvió error: %v", err)
		}

		got, err := q.GetSemesterByToken(ctx, creada.SubToken)
		if err != nil {
			t.Fatalf("GetSemesterByToken devolvió error: %v", err)
		}
		if got.IsOpen {
			t.Error("IsOpen = true después de poner end_date en el pasado")
		}
	})

	t.Run("id inexistente devuelve ErrNoRows", func(t *testing.T) {
		q := withTx(t)

		_, err := q.UpdateSemesterEndDate(context.Background(), UpdateSemesterEndDateParams{
			IDSemester: uuid.New(),
			EndDate:    time.Now(),
		})
		if !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("err = %v, want sql.ErrNoRows", err)
		}
	})
}

// ---------------------------------------------------------------------------
// DeleteSemester
// ---------------------------------------------------------------------------

func TestDeleteSemester(t *testing.T) {
	t.Run("borra la cursada existente", func(t *testing.T) {
		q := withTx(t)
		ctx := context.Background()

		creada := crearCursadaVigente(t, q)

		filas, err := q.DeleteSemester(ctx, creada.IDSemester)
		if err != nil {
			t.Fatalf("DeleteSemester devolvió error: %v", err)
		}
		if filas != 1 {
			t.Errorf("filas afectadas = %d, want 1", filas)
		}

		if _, err := q.GetSemesterById(ctx, creada.IDSemester); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("la cursada sigue existiendo tras el delete (err = %v)", err)
		}
	})

	t.Run("id inexistente afecta cero filas", func(t *testing.T) {
		q := withTx(t)

		filas, err := q.DeleteSemester(context.Background(), uuid.New())
		if err != nil {
			t.Fatalf("se esperaba nil, se obtuvo: %v", err)
		}
		if filas != 0 {
			t.Errorf("filas afectadas = %d, want 0", filas)
		}
	})

	t.Run("no se puede borrar una cursada con inscripciones", func(t *testing.T) {
		q := withTx(t)
		ctx := context.Background()

		cursada := crearCursadaVigente(t, q)

		if _, err := q.CreateInscription(ctx, CreateInscriptionParams{
			IDSemester: cursada.IDSemester,
			Email:      emailUnico("alumno"),
		}); err != nil {
			t.Fatalf("setup: %v", err)
		}

		if _, err := q.DeleteSemester(ctx, cursada.IDSemester); err == nil {
			t.Fatal("se borró una cursada con inscripciones; revisar la FK inscription_semester")
		}
	})
}
