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
// CreateUser
// ---------------------------------------------------------------------------

func TestCreateUser(t *testing.T) {
	casos := []struct {
		nombre       string
		fullname     string
		email        string
		passwordHash string
		wantErr      bool
	}{
		{
			nombre:       "usuario válido",
			fullname:     "Ana García",
			email:        "ana@test.com",
			passwordHash: "$2a$10$abcdefghijklmnopqrstuvwxyz",
		},
		{
			nombre:       "nombre con acentos y ñ",
			fullname:     "Iñaki Núñez",
			email:        "inaki@test.com",
			passwordHash: "hash",
		},
		{
			nombre:       "hash de bcrypt completo (60 chars)",
			fullname:     "Test Bcrypt",
			email:        "bcrypt@test.com",
			passwordHash: strings.Repeat("x", 60),
		},
		{
			nombre:       "fullname más largo que 50 chars",
			fullname:     strings.Repeat("a", 51),
			email:        "largo@test.com",
			passwordHash: "hash",
			wantErr:      true,
		},
		{
			nombre:       "email más largo que 80 chars",
			fullname:     "Test",
			email:        strings.Repeat("a", 75) + "@test.com",
			passwordHash: "hash",
			wantErr:      true,
		},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			q := withTx(t)

			got, err := q.CreateUser(context.Background(), CreateUserParams{
				Fullname:     c.fullname,
				Email:        c.email,
				PasswordHash: c.passwordHash,
			})

			if c.wantErr {
				if err == nil {
					t.Fatal("se esperaba un error, no hubo ninguno")
				}
				return
			}
			if err != nil {
				t.Fatalf("CreateUser devolvió error: %v", err)
			}

			// Lo que controlamos: vuelve tal cual lo mandamos.
			if got.Fullname != c.fullname {
				t.Errorf("Fullname = %q, want %q", got.Fullname, c.fullname)
			}
			if got.Email != c.email {
				t.Errorf("Email = %q, want %q", got.Email, c.email)
			}
			// El hash no se transforma al guardarlo: si esto falla,
			// la columna es demasiado corta y lo está truncando.
			if got.PasswordHash != c.passwordHash {
				t.Errorf("el hash se guardó truncado: %d chars, want %d",
					len(got.PasswordHash), len(c.passwordHash))
			}
			// Lo que genera la base: no sabemos el valor, pero no puede ser el cero.
			if got.IDUser == uuid.Nil {
				t.Error("IDUser quedó en el UUID cero; revisar DEFAULT gen_random_uuid()")
			}
		})
	}
}

// Necesita dos llamadas, así que no entra en la tabla de arriba.
func TestCreateUserEmailDuplicado(t *testing.T) {
	q := withTx(t)
	ctx := context.Background()

	const email = "repetido@test.com"

	if _, err := q.CreateUser(ctx, CreateUserParams{
		Fullname: "Primero", Email: email, PasswordHash: "hash",
	}); err != nil {
		t.Fatalf("el primer usuario falló: %v", err)
	}

	_, err := q.CreateUser(ctx, CreateUserParams{
		Fullname: "Segundo", Email: email, PasswordHash: "otro",
	})
	if err == nil {
		t.Fatal("se crearon dos usuarios con el mismo email; revisar la UNIQUE email_uk")
	}
}

// ---------------------------------------------------------------------------
// GetUserByEmail — la query del login
// ---------------------------------------------------------------------------

func TestGetUserByEmail(t *testing.T) {
	t.Run("devuelve el usuario con su hash", func(t *testing.T) {
		q := withTx(t)
		ctx := context.Background()

		const hash = "$2a$10$hashCompletoDeBcryptParaVerificar"
		creado, err := q.CreateUser(ctx, CreateUserParams{
			Fullname:     "Login Test",
			Email:        "login@test.com",
			PasswordHash: hash,
		})
		if err != nil {
			t.Fatalf("setup: %v", err)
		}

		got, err := q.GetUserByEmail(ctx, "login@test.com")
		if err != nil {
			t.Fatalf("GetUserByEmail devolvió error: %v", err)
		}

		if got.IDUser != creado.IDUser {
			t.Errorf("IDUser = %v, want %v", got.IDUser, creado.IDUser)
		}
		// Sin el hash, el login no puede verificar nada.
		if got.PasswordHash != hash {
			t.Errorf("PasswordHash = %q, want %q", got.PasswordHash, hash)
		}
	})

	t.Run("email inexistente devuelve ErrNoRows", func(t *testing.T) {
		q := withTx(t)

		_, err := q.GetUserByEmail(context.Background(), "nadie@test.com")
		if !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("err = %v, want sql.ErrNoRows", err)
		}
	})

	// Documenta el comportamiento actual: la búsqueda distingue mayúsculas.
	// Si se quiere que el login sea case-insensitive, hay que normalizar el
	// email antes de guardarlo o usar CITEXT en el schema.
	t.Run("la búsqueda distingue mayúsculas", func(t *testing.T) {
		q := withTx(t)
		ctx := context.Background()

		if _, err := q.CreateUser(ctx, CreateUserParams{
			Fullname: "Case Test", Email: "minusculas@test.com", PasswordHash: "hash",
		}); err != nil {
			t.Fatalf("setup: %v", err)
		}

		_, err := q.GetUserByEmail(ctx, "MINUSCULAS@test.com")
		if !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("la búsqueda encontró el usuario con otra capitalización (err = %v)", err)
		}
	})
}

// ---------------------------------------------------------------------------
// GetUserById
// ---------------------------------------------------------------------------

func TestGetUserById(t *testing.T) {
	t.Run("devuelve el usuario creado", func(t *testing.T) {
		q := withTx(t)
		ctx := context.Background()

		creado := crearProfesor(t, q)

		got, err := q.GetUserById(ctx, creado.IDUser)
		if err != nil {
			t.Fatalf("GetUserById devolvió error: %v", err)
		}

		if got.IDUser != creado.IDUser {
			t.Errorf("IDUser = %v, want %v", got.IDUser, creado.IDUser)
		}
		if got.Email != creado.Email {
			t.Errorf("Email = %q, want %q", got.Email, creado.Email)
		}
	})

	t.Run("id inexistente devuelve ErrNoRows", func(t *testing.T) {
		q := withTx(t)

		_, err := q.GetUserById(context.Background(), uuid.New())
		if !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("err = %v, want sql.ErrNoRows", err)
		}
	})
}

// ---------------------------------------------------------------------------
// ListUsers
// ---------------------------------------------------------------------------

func TestListUsers(t *testing.T) {
	t.Run("sin usuarios devuelve lista vacía", func(t *testing.T) {
		q := withTx(t)

		got, err := q.ListUsers(context.Background())
		if err != nil {
			t.Fatalf("ListUsers devolvió error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("se esperaba lista vacía, hay %d usuarios", len(got))
		}
	})

	t.Run("devuelve todos ordenados por fullname", func(t *testing.T) {
		q := withTx(t)
		ctx := context.Background()

		// Se crean desordenados a propósito.
		for _, nombre := range []string{"Zoe Vega", "Ana Díaz", "Marco Ruiz"} {
			if _, err := q.CreateUser(ctx, CreateUserParams{
				Fullname: nombre, Email: emailUnico("list"), PasswordHash: "hash",
			}); err != nil {
				t.Fatalf("setup %q: %v", nombre, err)
			}
		}

		got, err := q.ListUsers(ctx)
		if err != nil {
			t.Fatalf("ListUsers devolvió error: %v", err)
		}
		if len(got) != 3 {
			t.Fatalf("se devolvieron %d usuarios, want 3", len(got))
		}

		for i := 1; i < len(got); i++ {
			if got[i].Fullname < got[i-1].Fullname {
				t.Errorf("el orden por fullname está roto: %q viene después de %q",
					got[i].Fullname, got[i-1].Fullname)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// UpdateUserFullname
// ---------------------------------------------------------------------------

func TestUpdateUserFullname(t *testing.T) {
	t.Run("actualiza el nombre sin tocar el resto", func(t *testing.T) {
		q := withTx(t)
		ctx := context.Background()

		creado := crearProfesor(t, q)

		got, err := q.UpdateUserFullname(ctx, UpdateUserFullnameParams{
			IDUser:   creado.IDUser,
			Fullname: "Nombre Actualizado",
		})
		if err != nil {
			t.Fatalf("UpdateUserFullname devolvió error: %v", err)
		}

		if got.Fullname != "Nombre Actualizado" {
			t.Errorf("Fullname = %q, want %q", got.Fullname, "Nombre Actualizado")
		}
		if got.Email != creado.Email {
			t.Errorf("el update pisó el email: %q, want %q", got.Email, creado.Email)
		}
		if got.PasswordHash != creado.PasswordHash {
			t.Error("el update pisó el password_hash")
		}
	})

	t.Run("id inexistente devuelve ErrNoRows", func(t *testing.T) {
		q := withTx(t)

		_, err := q.UpdateUserFullname(context.Background(), UpdateUserFullnameParams{
			IDUser:   uuid.New(),
			Fullname: "No existe",
		})
		if !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("err = %v, want sql.ErrNoRows", err)
		}
	})
}

// ---------------------------------------------------------------------------
// UpdateUserPassword
// ---------------------------------------------------------------------------

func TestUpdateUserPassword(t *testing.T) {
	t.Run("cambia el hash y afecta una fila", func(t *testing.T) {
		q := withTx(t)
		ctx := context.Background()

		creado := crearProfesor(t, q)
		const nuevoHash = "$2a$10$hashNuevoDespuesDelCambio"

		filas, err := q.UpdateUserPassword(ctx, UpdateUserPasswordParams{
			IDUser:       creado.IDUser,
			PasswordHash: nuevoHash,
		})
		if err != nil {
			t.Fatalf("UpdateUserPassword devolvió error: %v", err)
		}
		if filas != 1 {
			t.Errorf("filas afectadas = %d, want 1", filas)
		}

		releido, err := q.GetUserById(ctx, creado.IDUser)
		if err != nil {
			t.Fatalf("no se pudo releer el usuario: %v", err)
		}
		if releido.PasswordHash != nuevoHash {
			t.Errorf("PasswordHash = %q, want %q", releido.PasswordHash, nuevoHash)
		}
	})

	// Acá se ve para qué sirve :execrows en lugar de :exec.
	t.Run("id inexistente afecta cero filas sin error", func(t *testing.T) {
		q := withTx(t)

		filas, err := q.UpdateUserPassword(context.Background(), UpdateUserPasswordParams{
			IDUser:       uuid.New(),
			PasswordHash: "hash",
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
// DeleteUser
// ---------------------------------------------------------------------------

func TestDeleteUser(t *testing.T) {
	t.Run("borra el usuario existente", func(t *testing.T) {
		q := withTx(t)
		ctx := context.Background()

		creado := crearProfesor(t, q)

		filas, err := q.DeleteUser(ctx, creado.IDUser)
		if err != nil {
			t.Fatalf("DeleteUser devolvió error: %v", err)
		}
		if filas != 1 {
			t.Errorf("filas afectadas = %d, want 1", filas)
		}

		if _, err := q.GetUserById(ctx, creado.IDUser); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("el usuario sigue existiendo tras el delete (err = %v)", err)
		}
	})

	t.Run("id inexistente afecta cero filas", func(t *testing.T) {
		q := withTx(t)

		filas, err := q.DeleteUser(context.Background(), uuid.New())
		if err != nil {
			t.Fatalf("se esperaba nil, se obtuvo: %v", err)
		}
		if filas != 0 {
			t.Errorf("filas afectadas = %d, want 0", filas)
		}
	})

	// La FK semester_user tiene que impedir borrar un profesor con cursadas.
	t.Run("no se puede borrar un profesor con cursadas", func(t *testing.T) {
		q := withTx(t)
		ctx := context.Background()

		cursada := crearCursadaVigente(t, q)

		sem, err := q.GetSemesterById(ctx, cursada.IDSemester)
		if err != nil {
			t.Fatalf("setup: %v", err)
		}

		_, err = q.DeleteUser(ctx, sem.ProfessorID)
		if err == nil {
			t.Fatal("se borró un profesor que tenía cursadas; revisar la FK semester_user")
		}
	})
}
