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
// CreateInscription
// ---------------------------------------------------------------------------

func TestCreateInscription(t *testing.T) {
	casos := []struct {
		nombre string
		email  string
		// badSemester fuerza un UUID inexistente para probar la FK.
		badSemester bool
		wantErr     bool
	}{
		{nombre: "inscripción válida", email: "alumno@test.com"},
		{nombre: "email con punto", email: "ana.garcia@test.com"},
		{nombre: "cursada inexistente", email: "alumno@test.com", badSemester: true, wantErr: true},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			q := withTx(t)
			ctx := context.Background()

			cursada := crearCursadaVigente(t, q)

			semID := cursada.IDSemester
			if c.badSemester {
				semID = uuid.New()
			}

			got, err := q.CreateInscription(ctx, CreateInscriptionParams{
				IDSemester: semID,
				Email:      c.email,
			})
			if checkErr(t, err, c.wantErr) {
				return
			}

			if got.IDInscription == uuid.Nil {
				t.Error("IDInscription quedó en el UUID cero")
			}
			if got.Email != c.email {
				t.Errorf("Email = %q, want %q", got.Email, c.email)
			}
			if got.IDSemester != semID {
				t.Errorf("IDSemester = %v, want %v", got.IDSemester, semID)
			}
			// active tiene DEFAULT true: la inscripción nace activa.
			if !got.Active {
				t.Error("la inscripción debería nacer activa; revisar DEFAULT true")
			}
			if got.CreatedAt.IsZero() {
				t.Error("CreatedAt quedó vacío; revisar DEFAULT now()")
			}
		})
	}
}

// El ciclo completo del link único: alta, baja, reactivación.
// Necesita varias llamadas encadenadas, así que va fuera de la tabla.
func TestCreateInscriptionReactivaEnLugarDeDuplicar(t *testing.T) {
	q := withTx(t)
	ctx := context.Background()

	cursada := crearCursadaVigente(t, q)
	const email = "alumno@test.com"

	// 1. Alta
	primera, err := q.CreateInscription(ctx, CreateInscriptionParams{
		IDSemester: cursada.IDSemester,
		Email:      email,
	})
	if err != nil {
		t.Fatalf("el alta falló: %v", err)
	}

	// 2. Baja
	filas, err := q.SetInscriptionActive(ctx, SetInscriptionActiveParams{
		IDSemester: cursada.IDSemester,
		Email:      email,
		Active:     false,
	})
	if err != nil {
		t.Fatalf("la baja falló: %v", err)
	}
	if filas != 1 {
		t.Fatalf("filas afectadas en la baja = %d, want 1", filas)
	}

	// 3. Vuelve a escanear el QR: debe reactivar la misma fila, no crear otra.
	segunda, err := q.CreateInscription(ctx, CreateInscriptionParams{
		IDSemester: cursada.IDSemester,
		Email:      email,
	})
	if err != nil {
		t.Fatalf("la reactivación falló; revisar el ON CONFLICT: %v", err)
	}

	if segunda.IDInscription != primera.IDInscription {
		t.Errorf("se creó una inscripción nueva (%v) en vez de reactivar la existente (%v)",
			segunda.IDInscription, primera.IDInscription)
	}
	if !segunda.Active {
		t.Error("la inscripción quedó inactiva después de reactivarla")
	}
}

// ---------------------------------------------------------------------------
// GetInscriptionById
// ---------------------------------------------------------------------------

func TestGetInscriptionById(t *testing.T) {
	t.Run("devuelve la inscripción creada", func(t *testing.T) {
		q := withTx(t)
		ctx := context.Background()

		cursada := crearCursadaVigente(t, q)
		creada, err := q.CreateInscription(ctx, CreateInscriptionParams{
			IDSemester: cursada.IDSemester,
			Email:      "alumno@test.com",
		})
		if err != nil {
			t.Fatalf("setup: %v", err)
		}

		got, err := q.GetInscriptionById(ctx, creada.IDInscription)
		if err != nil {
			t.Fatalf("GetInscriptionById devolvió error: %v", err)
		}
		if got.Email != creada.Email {
			t.Errorf("Email = %q, want %q", got.Email, creada.Email)
		}
	})

	t.Run("id inexistente devuelve ErrNoRows", func(t *testing.T) {
		q := withTx(t)

		_, err := q.GetInscriptionById(context.Background(), uuid.New())
		if !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("err = %v, want sql.ErrNoRows", err)
		}
	})
}

// ---------------------------------------------------------------------------
// GetInscriptionByEmailAndSemester — decide qué mostrar en el link
// ---------------------------------------------------------------------------

func TestGetInscriptionByEmailAndSemester(t *testing.T) {
	t.Run("devuelve la inscripción existente", func(t *testing.T) {
		q := withTx(t)
		ctx := context.Background()

		cursada := crearCursadaVigente(t, q)
		creada, err := q.CreateInscription(ctx, CreateInscriptionParams{
			IDSemester: cursada.IDSemester,
			Email:      "alumno@test.com",
		})
		if err != nil {
			t.Fatalf("setup: %v", err)
		}

		got, err := q.GetInscriptionByEmailAndSemester(ctx, GetInscriptionByEmailAndSemesterParams{
			IDSemester: cursada.IDSemester,
			Email:      "alumno@test.com",
		})
		if err != nil {
			t.Fatalf("GetInscriptionByEmailAndSemester devolvió error: %v", err)
		}
		if got.IDInscription != creada.IDInscription {
			t.Errorf("IDInscription = %v, want %v", got.IDInscription, creada.IDInscription)
		}
	})

	t.Run("email no inscripto devuelve ErrNoRows", func(t *testing.T) {
		q := withTx(t)
		ctx := context.Background()

		cursada := crearCursadaVigente(t, q)

		_, err := q.GetInscriptionByEmailAndSemester(ctx, GetInscriptionByEmailAndSemesterParams{
			IDSemester: cursada.IDSemester,
			Email:      "nadie@test.com",
		})
		if !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("err = %v, want sql.ErrNoRows", err)
		}
	})
}

// ---------------------------------------------------------------------------
// ListInscriptionsBySemester
// ---------------------------------------------------------------------------

func TestListInscriptionsBySemester(t *testing.T) {
	t.Run("cursada sin inscriptos devuelve lista vacía", func(t *testing.T) {
		q := withTx(t)

		cursada := crearCursadaVigente(t, q)

		got, err := q.ListInscriptionsBySemester(context.Background(), cursada.IDSemester)
		if err != nil {
			t.Fatalf("ListInscriptionsBySemester devolvió error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("se esperaba lista vacía, hay %d inscripciones", len(got))
		}
	})

	t.Run("incluye activas e inactivas", func(t *testing.T) {
		q := withTx(t)
		ctx := context.Background()

		cursada := crearCursadaVigente(t, q)

		for i := 0; i < 3; i++ {
			if _, err := q.CreateInscription(ctx, CreateInscriptionParams{
				IDSemester: cursada.IDSemester,
				Email:      emailUnico("alumno"),
			}); err != nil {
				t.Fatalf("setup: %v", err)
			}
		}

		// Una se da de baja: igual debe aparecer en este listado.
		primeras, err := q.ListInscriptionsBySemester(ctx, cursada.IDSemester)
		if err != nil {
			t.Fatalf("setup: %v", err)
		}
		if _, err := q.SetInscriptionActive(ctx, SetInscriptionActiveParams{
			IDSemester: cursada.IDSemester,
			Email:      primeras[0].Email,
			Active:     false,
		}); err != nil {
			t.Fatalf("setup baja: %v", err)
		}

		got, err := q.ListInscriptionsBySemester(ctx, cursada.IDSemester)
		if err != nil {
			t.Fatalf("ListInscriptionsBySemester devolvió error: %v", err)
		}
		if len(got) != 3 {
			t.Errorf("se devolvieron %d inscripciones, want 3 (activas e inactivas)", len(got))
		}
	})
}

// ---------------------------------------------------------------------------
// ListActiveEmailsBySemester — la query que arma los destinatarios
// ---------------------------------------------------------------------------

func TestListActiveEmailsBySemester(t *testing.T) {
	t.Run("devuelve los emails activos", func(t *testing.T) {
		q := withTx(t)
		ctx := context.Background()

		cursada := crearCursadaVigente(t, q)

		for i := 0; i < 3; i++ {
			if _, err := q.CreateInscription(ctx, CreateInscriptionParams{
				IDSemester: cursada.IDSemester,
				Email:      emailUnico("alumno"),
			}); err != nil {
				t.Fatalf("setup: %v", err)
			}
		}

		got, err := q.ListActiveEmailsBySemester(ctx, cursada.IDSemester)
		if err != nil {
			t.Fatalf("ListActiveEmailsBySemester devolvió error: %v", err)
		}
		if len(got) != 3 {
			t.Errorf("se devolvieron %d emails, want 3", len(got))
		}
	})

	t.Run("excluye los dados de baja", func(t *testing.T) {
		q := withTx(t)
		ctx := context.Background()

		cursada := crearCursadaVigente(t, q)
		const emailBaja = "sebaja@test.com"

		if _, err := q.CreateInscription(ctx, CreateInscriptionParams{
			IDSemester: cursada.IDSemester, Email: emailBaja,
		}); err != nil {
			t.Fatalf("setup: %v", err)
		}
		if _, err := q.CreateInscription(ctx, CreateInscriptionParams{
			IDSemester: cursada.IDSemester, Email: "sigue@test.com",
		}); err != nil {
			t.Fatalf("setup: %v", err)
		}

		if _, err := q.SetInscriptionActive(ctx, SetInscriptionActiveParams{
			IDSemester: cursada.IDSemester, Email: emailBaja, Active: false,
		}); err != nil {
			t.Fatalf("setup baja: %v", err)
		}

		got, err := q.ListActiveEmailsBySemester(ctx, cursada.IDSemester)
		if err != nil {
			t.Fatalf("ListActiveEmailsBySemester devolvió error: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("se devolvieron %d emails, want 1", len(got))
		}
		if got[0] == emailBaja {
			t.Error("apareció el email de un alumno dado de baja")
		}
	})

	// El cierre automático de cursada: la prueba de que funciona sin proceso aparte.
	t.Run("cursada vencida no devuelve destinatarios", func(t *testing.T) {
		q := withTx(t)
		ctx := context.Background()

		// Terminó hace un mes.
		vencida := crearCursada(t, q, time.Now().AddDate(0, -1, 0))

		for i := 0; i < 3; i++ {
			if _, err := q.CreateInscription(ctx, CreateInscriptionParams{
				IDSemester: vencida.IDSemester,
				Email:      emailUnico("alumno"),
			}); err != nil {
				t.Fatalf("setup: %v", err)
			}
		}

		got, err := q.ListActiveEmailsBySemester(ctx, vencida.IDSemester)
		if err != nil {
			t.Fatalf("ListActiveEmailsBySemester devolvió error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("se devolvieron %d emails de una cursada vencida, want 0", len(got))
		}
	})
}

// ---------------------------------------------------------------------------
// CountActiveBySemester
// ---------------------------------------------------------------------------

func TestCountActiveBySemester(t *testing.T) {
	t.Run("cuenta solo las activas", func(t *testing.T) {
		q := withTx(t)
		ctx := context.Background()

		cursada := crearCursadaVigente(t, q)
		const emailBaja = "sebaja@test.com"

		for _, email := range []string{emailBaja, "uno@test.com", "dos@test.com"} {
			if _, err := q.CreateInscription(ctx, CreateInscriptionParams{
				IDSemester: cursada.IDSemester, Email: email,
			}); err != nil {
				t.Fatalf("setup: %v", err)
			}
		}

		if _, err := q.SetInscriptionActive(ctx, SetInscriptionActiveParams{
			IDSemester: cursada.IDSemester, Email: emailBaja, Active: false,
		}); err != nil {
			t.Fatalf("setup baja: %v", err)
		}

		got, err := q.CountActiveBySemester(ctx, cursada.IDSemester)
		if err != nil {
			t.Fatalf("CountActiveBySemester devolvió error: %v", err)
		}
		if got != 2 {
			t.Errorf("count = %d, want 2", got)
		}
	})

	t.Run("cursada sin inscriptos devuelve cero", func(t *testing.T) {
		q := withTx(t)

		cursada := crearCursadaVigente(t, q)

		got, err := q.CountActiveBySemester(context.Background(), cursada.IDSemester)
		if err != nil {
			t.Fatalf("CountActiveBySemester devolvió error: %v", err)
		}
		if got != 0 {
			t.Errorf("count = %d, want 0", got)
		}
	})
}

// ---------------------------------------------------------------------------
// SetInscriptionActive
// ---------------------------------------------------------------------------

func TestSetInscriptionActive(t *testing.T) {
	casos := []struct {
		nombre      string
		activar     bool
		wantActivas int64
	}{
		{nombre: "dar de baja", activar: false, wantActivas: 0},
		{nombre: "dejar activa", activar: true, wantActivas: 1},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			q := withTx(t)
			ctx := context.Background()

			cursada := crearCursadaVigente(t, q)
			const email = "alumno@test.com"

			if _, err := q.CreateInscription(ctx, CreateInscriptionParams{
				IDSemester: cursada.IDSemester, Email: email,
			}); err != nil {
				t.Fatalf("setup: %v", err)
			}

			filas, err := q.SetInscriptionActive(ctx, SetInscriptionActiveParams{
				IDSemester: cursada.IDSemester,
				Email:      email,
				Active:     c.activar,
			})
			if err != nil {
				t.Fatalf("SetInscriptionActive devolvió error: %v", err)
			}
			if filas != 1 {
				t.Errorf("filas afectadas = %d, want 1", filas)
			}

			activas, err := q.CountActiveBySemester(ctx, cursada.IDSemester)
			if err != nil {
				t.Fatalf("CountActiveBySemester devolvió error: %v", err)
			}
			if activas != c.wantActivas {
				t.Errorf("activas = %d, want %d", activas, c.wantActivas)
			}
		})
	}

	t.Run("email no inscripto afecta cero filas", func(t *testing.T) {
		q := withTx(t)
		ctx := context.Background()

		cursada := crearCursadaVigente(t, q)

		filas, err := q.SetInscriptionActive(ctx, SetInscriptionActiveParams{
			IDSemester: cursada.IDSemester,
			Email:      "nadie@test.com",
			Active:     false,
		})
		if err != nil {
			t.Fatalf("se esperaba nil, se obtuvo: %v", err)
		}
		if filas != 0 {
			t.Errorf("filas afectadas = %d, want 0", filas)
		}
	})
}

// ---------------------------------------------------------------------------
// DeleteInscription
// ---------------------------------------------------------------------------

func TestDeleteInscription(t *testing.T) {
	t.Run("borra la inscripción existente", func(t *testing.T) {
		q := withTx(t)
		ctx := context.Background()

		cursada := crearCursadaVigente(t, q)
		creada, err := q.CreateInscription(ctx, CreateInscriptionParams{
			IDSemester: cursada.IDSemester,
			Email:      "alumno@test.com",
		})
		if err != nil {
			t.Fatalf("setup: %v", err)
		}

		filas, err := q.DeleteInscription(ctx, creada.IDInscription)
		if err != nil {
			t.Fatalf("DeleteInscription devolvió error: %v", err)
		}
		if filas != 1 {
			t.Errorf("filas afectadas = %d, want 1", filas)
		}

		if _, err := q.GetInscriptionById(ctx, creada.IDInscription); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("la inscripción sigue existiendo tras el delete (err = %v)", err)
		}
	})

	t.Run("id inexistente afecta cero filas", func(t *testing.T) {
		q := withTx(t)

		filas, err := q.DeleteInscription(context.Background(), uuid.New())
		if err != nil {
			t.Fatalf("se esperaba nil, se obtuvo: %v", err)
		}
		if filas != 0 {
			t.Errorf("filas afectadas = %d, want 0", filas)
		}
	})
}
