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
// CreateNotice
// ---------------------------------------------------------------------------

func TestCreateNotice(t *testing.T) {
	casos := []struct {
		nombre  string
		title   string
		message string
		// badSemester / badSender fuerzan un UUID inexistente para probar las FK.
		badSemester bool
		badSender   bool
		wantErr     bool
	}{
		{
			nombre:  "aviso válido",
			title:   "Cambio de aula",
			message: "La clase de hoy se dicta en el aula 12.",
		},
		{
			nombre:  "mensaje largo (la columna es TEXT)",
			title:   "Suspensión de clase",
			message: strings.Repeat("texto largo. ", 500),
		},
		{
			nombre:  "título en el límite de 255 chars",
			title:   strings.Repeat("a", 255),
			message: "cuerpo",
		},
		{
			nombre:  "título más largo que 255 chars",
			title:   strings.Repeat("a", 256),
			message: "cuerpo",
			wantErr: true,
		},
		{
			nombre:      "cursada inexistente",
			title:       "Título",
			message:     "cuerpo",
			badSemester: true,
			wantErr:     true,
		},
		{
			nombre:    "autor inexistente",
			title:     "Título",
			message:   "cuerpo",
			badSender: true,
			wantErr:   true,
		},
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
			senderID := cursada.ProfessorID
			if c.badSender {
				senderID = uuid.New()
			}

			got, err := q.CreateNotice(ctx, CreateNoticeParams{
				IDSemester: semID,
				IDSender:   senderID,
				Title:      c.title,
				Message:    c.message,
			})
			if checkErr(t, err, c.wantErr) {
				return
			}

			if got.IDNotice == uuid.Nil {
				t.Error("IDNotice quedó en el UUID cero")
			}
			if got.Title != c.title {
				t.Errorf("Title = %q, want %q", got.Title, c.title)
			}
			if got.Message != c.message {
				t.Error("Message no coincide con el insertado")
			}
			if got.IDSemester != semID {
				t.Errorf("IDSemester = %v, want %v", got.IDSemester, semID)
			}
			if got.IDSender != senderID {
				t.Errorf("IDSender = %v, want %v", got.IDSender, senderID)
			}
			if got.CreatedAt.IsZero() {
				t.Error("CreatedAt quedó vacío; revisar DEFAULT now()")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// GetNoticeById
// ---------------------------------------------------------------------------

func TestGetNoticeById(t *testing.T) {
	t.Run("devuelve el aviso creado", func(t *testing.T) {
		q := withTx(t)
		ctx := context.Background()

		cursada := crearCursadaVigente(t, q)

		creado, err := q.CreateNotice(ctx, CreateNoticeParams{
			IDSemester: cursada.IDSemester,
			IDSender:   cursada.ProfessorID,
			Title:      "Parcial reprogramado",
			Message:    "Pasa al viernes 20.",
		})
		if err != nil {
			t.Fatalf("setup: %v", err)
		}

		got, err := q.GetNoticeById(ctx, creado.IDNotice)
		if err != nil {
			t.Fatalf("GetNoticeById devolvió error: %v", err)
		}

		if got.IDNotice != creado.IDNotice {
			t.Errorf("IDNotice = %v, want %v", got.IDNotice, creado.IDNotice)
		}
		if got.Title != creado.Title {
			t.Errorf("Title = %q, want %q", got.Title, creado.Title)
		}
		// Verifica que las columnas no estén cruzadas en el SELECT.
		if got.IDSemester != cursada.IDSemester {
			t.Errorf("IDSemester = %v, want %v", got.IDSemester, cursada.IDSemester)
		}
		if got.IDSender != cursada.ProfessorID {
			t.Errorf("IDSender = %v, want %v", got.IDSender, cursada.ProfessorID)
		}
	})

	t.Run("id inexistente devuelve ErrNoRows", func(t *testing.T) {
		q := withTx(t)

		_, err := q.GetNoticeById(context.Background(), uuid.New())
		if !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("err = %v, want sql.ErrNoRows", err)
		}
	})
}

// ---------------------------------------------------------------------------
// ListNoticesBySemester
// ---------------------------------------------------------------------------

func TestListNoticesBySemester(t *testing.T) {
	t.Run("cursada sin avisos devuelve lista vacía", func(t *testing.T) {
		q := withTx(t)

		cursada := crearCursadaVigente(t, q)

		got, err := q.ListNoticesBySemester(context.Background(), cursada.IDSemester)
		if err != nil {
			t.Fatalf("ListNoticesBySemester devolvió error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("se esperaba lista vacía, hay %d avisos", len(got))
		}
	})

	t.Run("devuelve solo los avisos de esa cursada", func(t *testing.T) {
		q := withTx(t)
		ctx := context.Background()

		cursadaA := crearCursadaVigente(t, q)
		for i := 0; i < 2; i++ {
			if _, err := q.CreateNotice(ctx, CreateNoticeParams{
				IDSemester: cursadaA.IDSemester,
				IDSender:   cursadaA.ProfessorID,
				Title:      "Aviso de A",
				Message:    "cuerpo",
			}); err != nil {
				t.Fatalf("setup: %v", err)
			}
		}

		// Una cursada distinta con su propio aviso.
		cursadaB := crearCursadaVigente(t, q)
		if _, err := q.CreateNotice(ctx, CreateNoticeParams{
			IDSemester: cursadaB.IDSemester,
			IDSender:   cursadaB.ProfessorID,
			Title:      "Aviso de B",
			Message:    "cuerpo",
		}); err != nil {
			t.Fatalf("setup: %v", err)
		}

		got, err := q.ListNoticesBySemester(ctx, cursadaA.IDSemester)
		if err != nil {
			t.Fatalf("ListNoticesBySemester devolvió error: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("se devolvieron %d avisos, want 2", len(got))
		}
		for _, n := range got {
			if n.IDSemester != cursadaA.IDSemester {
				t.Error("apareció un aviso de otra cursada")
			}
		}
	})

	t.Run("ordena del más reciente al más viejo", func(t *testing.T) {
		q := withTx(t)
		ctx := context.Background()

		cursada := crearCursadaVigente(t, q)
		for i := 0; i < 3; i++ {
			if _, err := q.CreateNotice(ctx, CreateNoticeParams{
				IDSemester: cursada.IDSemester,
				IDSender:   cursada.ProfessorID,
				Title:      "Aviso",
				Message:    "cuerpo",
			}); err != nil {
				t.Fatalf("setup: %v", err)
			}
		}

		got, err := q.ListNoticesBySemester(ctx, cursada.IDSemester)
		if err != nil {
			t.Fatalf("ListNoticesBySemester devolvió error: %v", err)
		}

		for i := 1; i < len(got); i++ {
			if got[i].CreatedAt.After(got[i-1].CreatedAt) {
				t.Errorf("el orden descendente por created_at está roto entre %d y %d", i-1, i)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// ListNoticesWithSender
// ---------------------------------------------------------------------------

func TestListNoticesWithSender(t *testing.T) {
	t.Run("incluye el nombre del profesor", func(t *testing.T) {
		q := withTx(t)
		ctx := context.Background()

		cursada := crearCursadaVigente(t, q)
		profesor, err := q.GetUserById(ctx, cursada.ProfessorID)
		if err != nil {
			t.Fatalf("setup: %v", err)
		}

		if _, err := q.CreateNotice(ctx, CreateNoticeParams{
			IDSemester: cursada.IDSemester,
			IDSender:   cursada.ProfessorID,
			Title:      "Aviso con autor",
			Message:    "cuerpo",
		}); err != nil {
			t.Fatalf("setup: %v", err)
		}

		got, err := q.ListNoticesWithSender(ctx, cursada.IDSemester)
		if err != nil {
			t.Fatalf("ListNoticesWithSender devolvió error: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("se devolvieron %d avisos, want 1", len(got))
		}
		if got[0].SenderName != profesor.Fullname {
			t.Errorf("SenderName = %q, want %q; revisar el JOIN con users",
				got[0].SenderName, profesor.Fullname)
		}
	})
}

// ---------------------------------------------------------------------------
// CountNoticesBySemester
// ---------------------------------------------------------------------------

func TestCountNoticesBySemester(t *testing.T) {
	t.Run("cuenta los avisos de la cursada", func(t *testing.T) {
		q := withTx(t)
		ctx := context.Background()

		cursada := crearCursadaVigente(t, q)
		for i := 0; i < 4; i++ {
			if _, err := q.CreateNotice(ctx, CreateNoticeParams{
				IDSemester: cursada.IDSemester,
				IDSender:   cursada.ProfessorID,
				Title:      "Aviso",
				Message:    "cuerpo",
			}); err != nil {
				t.Fatalf("setup: %v", err)
			}
		}

		got, err := q.CountNoticesBySemester(ctx, cursada.IDSemester)
		if err != nil {
			t.Fatalf("CountNoticesBySemester devolvió error: %v", err)
		}
		if got != 4 {
			t.Errorf("count = %d, want 4", got)
		}
	})

	t.Run("cursada sin avisos devuelve cero", func(t *testing.T) {
		q := withTx(t)

		cursada := crearCursadaVigente(t, q)

		got, err := q.CountNoticesBySemester(context.Background(), cursada.IDSemester)
		if err != nil {
			t.Fatalf("CountNoticesBySemester devolvió error: %v", err)
		}
		if got != 0 {
			t.Errorf("count = %d, want 0", got)
		}
	})
}

// ---------------------------------------------------------------------------
// UpdateNoticeMessage
// ---------------------------------------------------------------------------

func TestUpdateNoticeMessage(t *testing.T) {
	t.Run("actualiza el mensaje sin tocar el título", func(t *testing.T) {
		q := withTx(t)
		ctx := context.Background()

		cursada := crearCursadaVigente(t, q)

		creado, err := q.CreateNotice(ctx, CreateNoticeParams{
			IDSemester: cursada.IDSemester,
			IDSender:   cursada.ProfessorID,
			Title:      "Título original",
			Message:    "Mensaje original",
		})
		if err != nil {
			t.Fatalf("setup: %v", err)
		}

		got, err := q.UpdateNoticeMessage(ctx, UpdateNoticeMessageParams{
			IDNotice: creado.IDNotice,
			Message:  "Mensaje corregido",
		})
		if err != nil {
			t.Fatalf("UpdateNoticeMessage devolvió error: %v", err)
		}

		if got.Message != "Mensaje corregido" {
			t.Errorf("Message = %q, want %q", got.Message, "Mensaje corregido")
		}
		if got.Title != creado.Title {
			t.Errorf("el update pisó el título: %q, want %q", got.Title, creado.Title)
		}
	})

	t.Run("id inexistente devuelve ErrNoRows", func(t *testing.T) {
		q := withTx(t)

		_, err := q.UpdateNoticeMessage(context.Background(), UpdateNoticeMessageParams{
			IDNotice: uuid.New(),
			Message:  "no afecta a nadie",
		})
		if !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("err = %v, want sql.ErrNoRows", err)
		}
	})
}

// ---------------------------------------------------------------------------
// DeleteNotice
// ---------------------------------------------------------------------------

func TestDeleteNotice(t *testing.T) {
	t.Run("borra el aviso existente", func(t *testing.T) {
		q := withTx(t)
		ctx := context.Background()

		cursada := crearCursadaVigente(t, q)

		creado, err := q.CreateNotice(ctx, CreateNoticeParams{
			IDSemester: cursada.IDSemester,
			IDSender:   cursada.ProfessorID,
			Title:      "A borrar",
			Message:    "cuerpo",
		})
		if err != nil {
			t.Fatalf("setup: %v", err)
		}

		filas, err := q.DeleteNotice(ctx, creado.IDNotice)
		if err != nil {
			t.Fatalf("DeleteNotice devolvió error: %v", err)
		}
		if filas != 1 {
			t.Errorf("filas afectadas = %d, want 1", filas)
		}

		if _, err := q.GetNoticeById(ctx, creado.IDNotice); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("el aviso sigue existiendo tras el delete (err = %v)", err)
		}
	})

	t.Run("id inexistente afecta cero filas", func(t *testing.T) {
		q := withTx(t)

		filas, err := q.DeleteNotice(context.Background(), uuid.New())
		if err != nil {
			t.Fatalf("se esperaba nil, se obtuvo: %v", err)
		}
		if filas != 0 {
			t.Errorf("filas afectadas = %d, want 0", filas)
		}
	})
}
