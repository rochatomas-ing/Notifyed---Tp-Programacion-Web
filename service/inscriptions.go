package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	db "servidor-NotifyEd/db/sqlc"

	"github.com/google/uuid"
)

// InscriptionQueries es la parte del repositorio sqlc que necesita el service.
// Incluye las cursadas porque toda inscripción se resuelve primero contra una
// cursada: por UUID (panel del profesor) o por sub_token (QR del alumno).
type InscriptionQueries interface {
	GetSemesterById(context.Context, uuid.UUID) (db.Semester, error)
	GetSemesterByToken(context.Context, string) (db.GetSemesterByTokenRow, error)
	CreateInscription(context.Context, db.CreateInscriptionParams) (db.Inscription, error)
	GetInscriptionById(context.Context, uuid.UUID) (db.Inscription, error)
	GetInscriptionByEmailAndSemester(context.Context, db.GetInscriptionByEmailAndSemesterParams) (db.Inscription, error)
	ListInscriptionsBySemester(context.Context, uuid.UUID) ([]db.Inscription, error)
	SetInscriptionActive(context.Context, db.SetInscriptionActiveParams) (int64, error)
	DeleteInscription(context.Context, uuid.UUID) (int64, error)
	CountActiveBySemester(context.Context, uuid.UUID) (int64, error)
}

type InscriptionService struct {
	queries InscriptionQueries
}

func NewInscriptionService(queries InscriptionQueries) *InscriptionService {
	return &InscriptionService{queries: queries}
}

// Subscribe da de alta al alumno en la cursada. Si ya estaba inscripto,
// CreateInscription lo reactiva (ON CONFLICT) y no hay 409. Una cursada
// cerrada (end_date pasada) no admite altas: ErrSemesterClosed.
func (s *InscriptionService) Subscribe(ctx context.Context, reference string, email string) (db.Inscription, error) {
	email, err := validateEmail(email)
	if err != nil {
		return db.Inscription{}, err
	}
	semester, err := s.resolveSemester(ctx, reference)
	if err != nil {
		return db.Inscription{}, err
	}
	if semesterClosed(semester.EndDate) {
		return db.Inscription{}, ErrSemesterClosed
	}
	inscription, err := s.queries.CreateInscription(ctx, db.CreateInscriptionParams{
		IDSemester: semester.IDSemester,
		Email:      email,
	})
	if err != nil {
		return db.Inscription{}, mapDatabaseError(err)
	}
	return inscription, nil
}

// Unsubscribe es la baja lógica: la fila queda con active = false. Es
// idempotente: si ya estaba inactiva, también termina sin error.
func (s *InscriptionService) Unsubscribe(ctx context.Context, reference string, email string) error {
	email, err := validateEmail(email)
	if err != nil {
		return err
	}
	semester, err := s.resolveSemester(ctx, reference)
	if err != nil {
		return err
	}
	rowsAffected, err := s.queries.SetInscriptionActive(ctx, db.SetInscriptionActiveParams{
		IDSemester: semester.IDSemester,
		Email:      email,
		Active:     false,
	})
	if err != nil {
		return mapDatabaseError(err)
	}
	if rowsAffected == 0 {
		return ErrInscriptionNotFound
	}
	return nil
}

// GetByEmail resuelve qué mostrarle al alumno que escanea el QR: si nunca se
// inscribió, ErrInscriptionNotFound (404).
func (s *InscriptionService) GetByEmail(ctx context.Context, reference string, email string) (db.Inscription, error) {
	email, err := validateEmail(email)
	if err != nil {
		return db.Inscription{}, err
	}
	semester, err := s.resolveSemester(ctx, reference)
	if err != nil {
		return db.Inscription{}, err
	}
	inscription, err := s.queries.GetInscriptionByEmailAndSemester(ctx, db.GetInscriptionByEmailAndSemesterParams{
		IDSemester: semester.IDSemester,
		Email:      email,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return db.Inscription{}, ErrInscriptionNotFound
		}
		return db.Inscription{}, mapDatabaseError(err)
	}
	return inscription, nil
}

// ListBySemester es el listado del profesor: todos los inscriptos, activos e
// inactivos, del más viejo al más nuevo.
func (s *InscriptionService) ListBySemester(ctx context.Context, id uuid.UUID) ([]db.Inscription, error) {
	inscriptions, err := s.queries.ListInscriptionsBySemester(ctx, id)
	if inscriptions == nil && err == nil {
		inscriptions = []db.Inscription{}
	}
	return inscriptions, err
}

// CountActive devuelve el contador de suscriptos activos de una cursada.
func (s *InscriptionService) CountActive(ctx context.Context, id uuid.UUID) (int64, error) {
	return s.queries.CountActiveBySemester(ctx, id)
}

// Toggle invierte active. SetInscriptionActive filtra por (id_semester,
// email), así que primero se busca la fila por id de inscripción.
func (s *InscriptionService) Toggle(ctx context.Context, id uuid.UUID) (db.Inscription, error) {
	inscription, err := s.queries.GetInscriptionById(ctx, id)
	if err != nil {
		return db.Inscription{}, err
	}
	rowsAffected, err := s.queries.SetInscriptionActive(ctx, db.SetInscriptionActiveParams{
		IDSemester: inscription.IDSemester,
		Email:      inscription.Email,
		Active:     !inscription.Active,
	})
	if err != nil {
		return db.Inscription{}, mapDatabaseError(err)
	}
	if rowsAffected == 0 {
		return db.Inscription{}, sql.ErrNoRows
	}
	inscription.Active = !inscription.Active
	return inscription, nil
}

// Delete es el borrado duro del panel del profesor. El alumno, en cambio, se
// da de baja lógica con Unsubscribe.
func (s *InscriptionService) Delete(ctx context.Context, id uuid.UUID) error {
	rowsAffected, err := s.queries.DeleteInscription(ctx, id)
	if err != nil {
		return mapDatabaseError(err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// resolveSemester acepta lo que venga en la URL: un UUID (panel del profesor)
// o un sub_token (QR del alumno). Es la decisión #4 del contrato.
func (s *InscriptionService) resolveSemester(ctx context.Context, reference string) (db.Semester, error) {
	if id, err := uuid.Parse(reference); err == nil {
		return s.queries.GetSemesterById(ctx, id)
	}
	row, err := s.queries.GetSemesterByToken(ctx, reference)
	if err != nil {
		return db.Semester{}, err
	}
	return db.Semester{
		IDSemester:  row.IDSemester,
		IDCourse:    row.IDCourse,
		ProfessorID: row.ProfessorID,
		Year:        row.Year,
		SubToken:    row.SubToken,
		EndDate:     row.EndDate,
	}, nil
}

// semesterClosed replica la regla de is_open de las queries (end_date >= hoy).
func semesterClosed(endDate time.Time) bool {
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	return endDate.Before(today)
}

// validateEmail normaliza y valida el email del alumno (decisión #6): es
// obligatorio, admite hasta 80 caracteres (varchar(80)) y tiene que tener
// formato válido.
func validateEmail(email string) (string, error) {
	email = strings.TrimSpace(email)
	if email == "" {
		return "", fmt.Errorf("%w: el email es obligatorio", ErrInvalidInput)
	}
	if len([]rune(email)) > 80 {
		return "", fmt.Errorf("%w: el email admite hasta 80 caracteres", ErrInvalidInput)
	}
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		return "", fmt.Errorf("%w: el email no tiene un formato válido", ErrInvalidInput)
	}
	return email, nil
}
