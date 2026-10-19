package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	db "servidor-NotifyEd/db/sqlc"

	"github.com/google/uuid"
)

// ---- Fake del repositorio: NoticeQueries en memoria, sin base de datos ----

type fakeNoticeQueries struct {
	createNotice        func(context.Context, db.CreateNoticeParams) (db.Notice, error)
	getNoticeById       func(context.Context, uuid.UUID) (db.Notice, error)
	listNotices         func(context.Context) ([]db.Notice, error)
	listNoticesWSender  func(context.Context, uuid.UUID) ([]db.ListNoticesWithSenderRow, error)
	updateNoticeMessage func(context.Context, db.UpdateNoticeMessageParams) (db.Notice, error)
	deleteNotice        func(context.Context, uuid.UUID) (int64, error)
	getUserById         func(context.Context, uuid.UUID) (db.User, error)
}

func (f *fakeNoticeQueries) CreateNotice(ctx context.Context, arg db.CreateNoticeParams) (db.Notice, error) {
	return f.createNotice(ctx, arg)
}

func (f *fakeNoticeQueries) GetNoticeById(ctx context.Context, id uuid.UUID) (db.Notice, error) {
	return f.getNoticeById(ctx, id)
}

func (f *fakeNoticeQueries) ListNotices(ctx context.Context) ([]db.Notice, error) {
	return f.listNotices(ctx)
}

func (f *fakeNoticeQueries) ListNoticesWithSender(ctx context.Context, id uuid.UUID) ([]db.ListNoticesWithSenderRow, error) {
	return f.listNoticesWSender(ctx, id)
}

func (f *fakeNoticeQueries) UpdateNoticeMessage(ctx context.Context, arg db.UpdateNoticeMessageParams) (db.Notice, error) {
	return f.updateNoticeMessage(ctx, arg)
}

func (f *fakeNoticeQueries) DeleteNotice(ctx context.Context, id uuid.UUID) (int64, error) {
	return f.deleteNotice(ctx, id)
}

func (f *fakeNoticeQueries) GetUserById(ctx context.Context, id uuid.UUID) (db.User, error) {
	return f.getUserById(ctx, id)
}

// nuevoFakeNoticeQueries arma un fake con comportamiento feliz por defecto;
// cada test pisa sólo los métodos que necesita.
func nuevoFakeNoticeQueries() *fakeNoticeQueries {
	return &fakeNoticeQueries{
		createNotice: func(_ context.Context, arg db.CreateNoticeParams) (db.Notice, error) {
			return db.Notice{
				IDNotice:   uuid.New(),
				IDSemester: arg.IDSemester,
				IDSender:   arg.IDSender,
				Title:      arg.Title,
				Message:    arg.Message,
			}, nil
		},
		getNoticeById: func(_ context.Context, id uuid.UUID) (db.Notice, error) {
			return db.Notice{IDNotice: id}, nil
		},
		listNotices: func(context.Context) ([]db.Notice, error) {
			return nil, nil
		},
		listNoticesWSender: func(context.Context, uuid.UUID) ([]db.ListNoticesWithSenderRow, error) {
			return nil, nil
		},
		updateNoticeMessage: func(_ context.Context, arg db.UpdateNoticeMessageParams) (db.Notice, error) {
			return db.Notice{IDNotice: arg.IDNotice, Message: arg.Message}, nil
		},
		deleteNotice: func(context.Context, uuid.UUID) (int64, error) {
			return 1, nil
		},
		getUserById: func(_ context.Context, id uuid.UUID) (db.User, error) {
			return db.User{IDUser: id}, nil
		},
	}
}

func avisoValido() db.CreateNoticeParams {
	return db.CreateNoticeParams{
		IDSemester: uuid.New(),
		IDSender:   uuid.New(),
		Title:      "Aviso de prueba",
		Message:    "Se toma parcial el lunes",
	}
}

// ---- Create ----

func TestNoticeServiceCreate(t *testing.T) {
	errBase := errors.New("error de base")

	casos := []struct {
		nombre     string
		params     db.CreateNoticeParams
		configurar func(*fakeNoticeQueries)
		wantErr    error
	}{
		{
			nombre: "aviso válido",
			params: avisoValido(),
		},
		{
			nombre: "título vacío",
			params: func() db.CreateNoticeParams {
				p := avisoValido()
				p.Title = "   "
				return p
			}(),
			wantErr: ErrInvalidInput,
		},
		{
			nombre: "mensaje vacío",
			params: func() db.CreateNoticeParams {
				p := avisoValido()
				p.Message = ""
				return p
			}(),
			wantErr: ErrInvalidInput,
		},
		{
			nombre: "id_semester nulo",
			params: func() db.CreateNoticeParams {
				p := avisoValido()
				p.IDSemester = uuid.Nil
				return p
			}(),
			wantErr: ErrInvalidInput,
		},
		{
			nombre: "profesor inexistente",
			params: avisoValido(),
			configurar: func(f *fakeNoticeQueries) {
				f.getUserById = func(context.Context, uuid.UUID) (db.User, error) {
					return db.User{}, sql.ErrNoRows
				}
			},
			wantErr: ErrInvalidInput,
		},
		{
			nombre: "error del repositorio",
			params: avisoValido(),
			configurar: func(f *fakeNoticeQueries) {
				f.createNotice = func(context.Context, db.CreateNoticeParams) (db.Notice, error) {
					return db.Notice{}, errBase
				}
			},
			wantErr: errBase,
		},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			fake := nuevoFakeNoticeQueries()
			if caso.configurar != nil {
				caso.configurar(fake)
			}
			servicio := NewNoticeService(fake)

			aviso, err := servicio.Create(context.Background(), caso.params)

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
			if aviso.Title != caso.params.Title || aviso.Message != caso.params.Message {
				t.Errorf("el aviso devuelto no coincide con los datos enviados: %+v", aviso)
			}
		})
	}
}

func TestNoticeServiceCreateRecortaEspacios(t *testing.T) {
	fake := nuevoFakeNoticeQueries()
	var recibido db.CreateNoticeParams
	fake.createNotice = func(_ context.Context, arg db.CreateNoticeParams) (db.Notice, error) {
		recibido = arg
		return db.Notice{IDNotice: uuid.New()}, nil
	}
	servicio := NewNoticeService(fake)

	params := avisoValido()
	params.Title = "  Aviso con espacios  "
	params.Message = "  mensaje  "
	if _, err := servicio.Create(context.Background(), params); err != nil {
		t.Fatalf("no se esperaba error, se obtuvo %v", err)
	}

	if recibido.Title != "Aviso con espacios" || recibido.Message != "mensaje" {
		t.Errorf("se esperaba recortar los espacios, se recibió %+v", recibido)
	}
}

// ---- UpdateMessage ----

func TestNoticeServiceUpdateMessage(t *testing.T) {
	casos := []struct {
		nombre  string
		message string
		wantErr error
	}{
		{nombre: "mensaje válido", message: "Mensaje editado"},
		{nombre: "mensaje vacío", message: "   ", wantErr: ErrInvalidInput},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			servicio := NewNoticeService(nuevoFakeNoticeQueries())

			aviso, err := servicio.UpdateMessage(context.Background(), uuid.New(), caso.message)

			if caso.wantErr != nil {
				if !errors.Is(err, caso.wantErr) {
					t.Fatalf("se esperaba error %v, se obtuvo %v", caso.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("no se esperaba error, se obtuvo %v", err)
			}
			if aviso.Message != "Mensaje editado" {
				t.Errorf("se esperaba el mensaje editado, se obtuvo %+v", aviso)
			}
		})
	}
}

func TestNoticeServiceUpdateMessageInexistente(t *testing.T) {
	fake := nuevoFakeNoticeQueries()
	fake.updateNoticeMessage = func(context.Context, db.UpdateNoticeMessageParams) (db.Notice, error) {
		return db.Notice{}, sql.ErrNoRows
	}
	servicio := NewNoticeService(fake)

	_, err := servicio.UpdateMessage(context.Background(), uuid.New(), "mensaje")

	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("se esperaba sql.ErrNoRows, se obtuvo %v", err)
	}
}

// ---- Delete ----

func TestNoticeServiceDelete(t *testing.T) {
	casos := []struct {
		nombre  string
		filas   int64
		wantErr error
	}{
		{nombre: "aviso existente", filas: 1},
		{nombre: "aviso inexistente", filas: 0, wantErr: sql.ErrNoRows},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			fake := nuevoFakeNoticeQueries()
			fake.deleteNotice = func(context.Context, uuid.UUID) (int64, error) {
				return caso.filas, nil
			}
			servicio := NewNoticeService(fake)

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

// ---- List ----

func TestNoticeServiceListDevuelveArregloVacio(t *testing.T) {
	fake := nuevoFakeNoticeQueries()
	fake.listNotices = func(context.Context) ([]db.Notice, error) {
		return nil, nil
	}
	servicio := NewNoticeService(fake)

	avisos, err := servicio.List(context.Background())

	if err != nil {
		t.Fatalf("no se esperaba error, se obtuvo %v", err)
	}
	if avisos == nil {
		t.Fatal("se esperaba un arreglo vacío, se obtuvo nil (JSON null)")
	}
	if len(avisos) != 0 {
		t.Errorf("se esperaba un arreglo vacío, se obtuvo %+v", avisos)
	}
}
