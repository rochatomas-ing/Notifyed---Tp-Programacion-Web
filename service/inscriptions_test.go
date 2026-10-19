package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	db "servidor-NotifyEd/db/sqlc"

	"github.com/google/uuid"
)

// ---- Fake del repositorio: InscriptionQueries en memoria, sin base de datos ----

type fakeInscriptionQueries struct {
	getSemesterById                  func(context.Context, uuid.UUID) (db.Semester, error)
	getSemesterByToken               func(context.Context, string) (db.GetSemesterByTokenRow, error)
	createInscription                func(context.Context, db.CreateInscriptionParams) (db.Inscription, error)
	getInscriptionById               func(context.Context, uuid.UUID) (db.Inscription, error)
	getInscriptionByEmailAndSemester func(context.Context, db.GetInscriptionByEmailAndSemesterParams) (db.Inscription, error)
	listInscriptionsBySemester       func(context.Context, uuid.UUID) ([]db.Inscription, error)
	setInscriptionActive             func(context.Context, db.SetInscriptionActiveParams) (int64, error)
	deleteInscription                func(context.Context, uuid.UUID) (int64, error)
	countActiveBySemester            func(context.Context, uuid.UUID) (int64, error)
}

func (f *fakeInscriptionQueries) GetSemesterById(ctx context.Context, id uuid.UUID) (db.Semester, error) {
	return f.getSemesterById(ctx, id)
}

func (f *fakeInscriptionQueries) GetSemesterByToken(ctx context.Context, token string) (db.GetSemesterByTokenRow, error) {
	return f.getSemesterByToken(ctx, token)
}

func (f *fakeInscriptionQueries) CreateInscription(ctx context.Context, arg db.CreateInscriptionParams) (db.Inscription, error) {
	return f.createInscription(ctx, arg)
}

func (f *fakeInscriptionQueries) GetInscriptionById(ctx context.Context, id uuid.UUID) (db.Inscription, error) {
	return f.getInscriptionById(ctx, id)
}

func (f *fakeInscriptionQueries) GetInscriptionByEmailAndSemester(ctx context.Context, arg db.GetInscriptionByEmailAndSemesterParams) (db.Inscription, error) {
	return f.getInscriptionByEmailAndSemester(ctx, arg)
}

func (f *fakeInscriptionQueries) ListInscriptionsBySemester(ctx context.Context, id uuid.UUID) ([]db.Inscription, error) {
	return f.listInscriptionsBySemester(ctx, id)
}

func (f *fakeInscriptionQueries) SetInscriptionActive(ctx context.Context, arg db.SetInscriptionActiveParams) (int64, error) {
	return f.setInscriptionActive(ctx, arg)
}

func (f *fakeInscriptionQueries) DeleteInscription(ctx context.Context, id uuid.UUID) (int64, error) {
	return f.deleteInscription(ctx, id)
}

func (f *fakeInscriptionQueries) CountActiveBySemester(ctx context.Context, id uuid.UUID) (int64, error) {
	return f.countActiveBySemester(ctx, id)
}

const tokenDePrueba = "sub-token-de-prueba"

// cursadaAbierta arma una cursada vigente; los tests de cursada cerrada le
// pisan la fecha.
func cursadaAbierta() db.Semester {
	return db.Semester{
		IDSemester: uuid.New(),
		SubToken:   tokenDePrueba,
		EndDate:    time.Now().UTC().AddDate(0, 0, 30),
	}
}

// nuevoFakeInscriptionQueries arma un fake con comportamiento feliz por
// defecto; cada test pisa sólo los métodos que necesita.
func nuevoFakeInscriptionQueries() *fakeInscriptionQueries {
	return &fakeInscriptionQueries{
		getSemesterById: func(context.Context, uuid.UUID) (db.Semester, error) {
			return cursadaAbierta(), nil
		},
		getSemesterByToken: func(_ context.Context, token string) (db.GetSemesterByTokenRow, error) {
			semester := cursadaAbierta()
			return db.GetSemesterByTokenRow{
				IDSemester: semester.IDSemester,
				IDCourse:   semester.IDCourse,
				Year:       semester.Year,
				SubToken:   token,
				EndDate:    semester.EndDate,
				IsOpen:     true,
			}, nil
		},
		createInscription: func(_ context.Context, arg db.CreateInscriptionParams) (db.Inscription, error) {
			return db.Inscription{
				IDInscription: uuid.New(),
				IDSemester:    arg.IDSemester,
				Email:         arg.Email,
				Active:        true,
			}, nil
		},
		getInscriptionById: func(_ context.Context, id uuid.UUID) (db.Inscription, error) {
			return db.Inscription{IDInscription: id, IDSemester: uuid.New(), Email: "alumno@test.com", Active: true}, nil
		},
		getInscriptionByEmailAndSemester: func(_ context.Context, arg db.GetInscriptionByEmailAndSemesterParams) (db.Inscription, error) {
			return db.Inscription{
				IDInscription: uuid.New(),
				IDSemester:    arg.IDSemester,
				Email:         arg.Email,
				Active:        true,
			}, nil
		},
		listInscriptionsBySemester: func(context.Context, uuid.UUID) ([]db.Inscription, error) {
			return nil, nil
		},
		setInscriptionActive: func(context.Context, db.SetInscriptionActiveParams) (int64, error) {
			return 1, nil
		},
		deleteInscription: func(context.Context, uuid.UUID) (int64, error) {
			return 1, nil
		},
		countActiveBySemester: func(context.Context, uuid.UUID) (int64, error) {
			return 0, nil
		},
	}
}

// ---- Subscribe ----

func TestInscriptionServiceSubscribe(t *testing.T) {
	errBase := errors.New("error de base")

	casos := []struct {
		nombre        string
		referencia    string
		email         string
		configurar    func(*fakeInscriptionQueries)
		wantErr       error
		emailEsperado string
	}{
		{
			nombre:        "token de la cursada válido",
			referencia:    tokenDePrueba,
			email:         "alumno@test.com",
			emailEsperado: "alumno@test.com",
		},
		{
			nombre:        "referencia por UUID (panel del profesor)",
			referencia:    uuid.New().String(),
			email:         "alumno@test.com",
			emailEsperado: "alumno@test.com",
		},
		{
			nombre:        "email con espacios se recorta",
			referencia:    tokenDePrueba,
			email:         "  alumno@test.com  ",
			emailEsperado: "alumno@test.com",
		},
		{
			nombre:     "email vacío",
			referencia: tokenDePrueba,
			email:      "   ",
			wantErr:    ErrInvalidInput,
		},
		{
			nombre:     "email sin formato válido",
			referencia: tokenDePrueba,
			email:      "no-es-un-email",
			wantErr:    ErrInvalidInput,
		},
		{
			nombre:     "email de más de 80 caracteres",
			referencia: tokenDePrueba,
			email:      strings.Repeat("a", 72) + "@test.com",
			wantErr:    ErrInvalidInput,
		},
		{
			nombre:     "cursada inexistente",
			referencia: "token-que-no-existe",
			email:      "alumno@test.com",
			configurar: func(f *fakeInscriptionQueries) {
				f.getSemesterByToken = func(context.Context, string) (db.GetSemesterByTokenRow, error) {
					return db.GetSemesterByTokenRow{}, sql.ErrNoRows
				}
			},
			wantErr: sql.ErrNoRows,
		},
		{
			nombre:     "cursada cerrada",
			referencia: tokenDePrueba,
			email:      "alumno@test.com",
			configurar: func(f *fakeInscriptionQueries) {
				f.getSemesterByToken = func(_ context.Context, token string) (db.GetSemesterByTokenRow, error) {
					return db.GetSemesterByTokenRow{
						IDSemester: uuid.New(),
						SubToken:   token,
						EndDate:    time.Now().UTC().AddDate(0, 0, -1),
						IsOpen:     false,
					}, nil
				}
			},
			wantErr: ErrSemesterClosed,
		},
		{
			nombre:     "error del repositorio",
			referencia: tokenDePrueba,
			email:      "alumno@test.com",
			configurar: func(f *fakeInscriptionQueries) {
				f.createInscription = func(context.Context, db.CreateInscriptionParams) (db.Inscription, error) {
					return db.Inscription{}, errBase
				}
			},
			wantErr: errBase,
		},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			fake := nuevoFakeInscriptionQueries()
			if caso.configurar != nil {
				caso.configurar(fake)
			}
			servicio := NewInscriptionService(fake)

			inscription, err := servicio.Subscribe(context.Background(), caso.referencia, caso.email)

			if caso.wantErr != nil {
				if err == nil {
					t.Fatalf("se esperaba error %v y no hubo ninguno", caso.wantErr)
				}
				if !errors.Is(err, caso.wantErr) {
					t.Fatalf("se esperaba error %v, se obtuvo %v", caso.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("no se esperaba error, se obtuvo %v", err)
			}
			if inscription.Email != caso.emailEsperado {
				t.Errorf("se esperaba el email %q, se recibió %q", caso.emailEsperado, inscription.Email)
			}
			if !inscription.Active {
				t.Errorf("la inscripción nueva debería quedar activa: %+v", inscription)
			}
		})
	}
}

// ---- Unsubscribe ----

func TestInscriptionServiceUnsubscribe(t *testing.T) {
	errBase := errors.New("error de base")

	casos := []struct {
		nombre     string
		email      string
		configurar func(*fakeInscriptionQueries)
		wantErr    error
	}{
		{nombre: "fila existente (queda inactiva)", email: "alumno@test.com"},
		{
			nombre: "nunca inscripto",
			email:  "nunca@test.com",
			configurar: func(f *fakeInscriptionQueries) {
				f.setInscriptionActive = func(context.Context, db.SetInscriptionActiveParams) (int64, error) {
					return 0, nil
				}
			},
			wantErr: ErrInscriptionNotFound,
		},
		{nombre: "email vacío", email: " ", wantErr: ErrInvalidInput},
		{
			nombre: "error del repositorio",
			email:  "alumno@test.com",
			configurar: func(f *fakeInscriptionQueries) {
				f.setInscriptionActive = func(context.Context, db.SetInscriptionActiveParams) (int64, error) {
					return 0, errBase
				}
			},
			wantErr: errBase,
		},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			fake := nuevoFakeInscriptionQueries()
			if caso.configurar != nil {
				caso.configurar(fake)
			}
			servicio := NewInscriptionService(fake)

			err := servicio.Unsubscribe(context.Background(), tokenDePrueba, caso.email)

			if caso.wantErr != nil {
				if !errors.Is(err, caso.wantErr) {
					t.Fatalf("se esperaba error %v, se obtuvo %v", caso.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("no se esperaba error, se obtuvo %v", err)
			}
		})
	}
}

func TestInscriptionServiceUnsubscribeDesactiva(t *testing.T) {
	fake := nuevoFakeInscriptionQueries()
	var recibido db.SetInscriptionActiveParams
	fake.setInscriptionActive = func(_ context.Context, arg db.SetInscriptionActiveParams) (int64, error) {
		recibido = arg
		return 1, nil
	}
	servicio := NewInscriptionService(fake)

	if err := servicio.Unsubscribe(context.Background(), tokenDePrueba, "alumno@test.com"); err != nil {
		t.Fatalf("no se esperaba error, se obtuvo %v", err)
	}
	if recibido.Active {
		t.Errorf("la baja debe setear active = false: %+v", recibido)
	}
}

// ---- GetByEmail ----

func TestInscriptionServiceGetByEmail(t *testing.T) {
	casos := []struct {
		nombre     string
		configurar func(*fakeInscriptionQueries)
		wantErr    error
	}{
		{nombre: "inscripción existente"},
		{
			nombre: "nunca inscripto",
			configurar: func(f *fakeInscriptionQueries) {
				f.getInscriptionByEmailAndSemester = func(context.Context, db.GetInscriptionByEmailAndSemesterParams) (db.Inscription, error) {
					return db.Inscription{}, sql.ErrNoRows
				}
			},
			wantErr: ErrInscriptionNotFound,
		},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			fake := nuevoFakeInscriptionQueries()
			if caso.configurar != nil {
				caso.configurar(fake)
			}
			servicio := NewInscriptionService(fake)

			inscription, err := servicio.GetByEmail(context.Background(), tokenDePrueba, "alumno@test.com")

			if caso.wantErr != nil {
				if !errors.Is(err, caso.wantErr) {
					t.Fatalf("se esperaba error %v, se obtuvo %v", caso.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("no se esperaba error, se obtuvo %v", err)
			}
			if inscription.Email != "alumno@test.com" {
				t.Errorf("el email devuelto no coincide: %+v", inscription)
			}
		})
	}
}

// ---- Toggle ----

func TestInscriptionServiceToggle(t *testing.T) {
	fake := nuevoFakeInscriptionQueries()
	var recibido db.SetInscriptionActiveParams
	fake.setInscriptionActive = func(_ context.Context, arg db.SetInscriptionActiveParams) (int64, error) {
		recibido = arg
		return 1, nil
	}
	servicio := NewInscriptionService(fake)

	inscription, err := servicio.Toggle(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("no se esperaba error, se obtuvo %v", err)
	}
	if inscription.Active {
		t.Errorf("una inscripción activa debería pasar a inactiva: %+v", inscription)
	}
	if recibido.Active {
		t.Errorf("el update debería setear active = false: %+v", recibido)
	}
}

func TestInscriptionServiceToggleInexistente(t *testing.T) {
	fake := nuevoFakeInscriptionQueries()
	fake.getInscriptionById = func(context.Context, uuid.UUID) (db.Inscription, error) {
		return db.Inscription{}, sql.ErrNoRows
	}
	servicio := NewInscriptionService(fake)

	_, err := servicio.Toggle(context.Background(), uuid.New())
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("se esperaba sql.ErrNoRows, se obtuvo %v", err)
	}
}

// ---- Delete ----

func TestInscriptionServiceDelete(t *testing.T) {
	casos := []struct {
		nombre  string
		filas   int64
		wantErr error
	}{
		{nombre: "inscripción existente", filas: 1},
		{nombre: "inscripción inexistente", filas: 0, wantErr: sql.ErrNoRows},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			fake := nuevoFakeInscriptionQueries()
			fake.deleteInscription = func(context.Context, uuid.UUID) (int64, error) {
				return caso.filas, nil
			}
			servicio := NewInscriptionService(fake)

			err := servicio.Delete(context.Background(), uuid.New())

			if caso.wantErr != nil {
				if !errors.Is(err, caso.wantErr) {
					t.Fatalf("se esperaba error %v, se obtuvo %v", caso.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("no se esperaba error, se obtuvo %v", err)
			}
		})
	}
}

// ---- ListBySemester y CountActive ----

func TestInscriptionServiceListDevuelveArregloVacio(t *testing.T) {
	servicio := NewInscriptionService(nuevoFakeInscriptionQueries())

	inscriptions, err := servicio.ListBySemester(context.Background(), uuid.New())

	if err != nil {
		t.Fatalf("no se esperaba error, se obtuvo %v", err)
	}
	if inscriptions == nil {
		t.Fatal("se esperaba un arreglo vacío, se obtuvo nil (JSON null)")
	}
	if len(inscriptions) != 0 {
		t.Errorf("se esperaba un arreglo vacío, se obtuvo %+v", inscriptions)
	}
}

func TestInscriptionServiceCountActive(t *testing.T) {
	fake := nuevoFakeInscriptionQueries()
	fake.countActiveBySemester = func(context.Context, uuid.UUID) (int64, error) {
		return 7, nil
	}
	servicio := NewInscriptionService(fake)

	count, err := servicio.CountActive(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("no se esperaba error, se obtuvo %v", err)
	}
	if count != 7 {
		t.Errorf("se esperaban 7 suscriptos activos, se obtuvieron %d", count)
	}
}
