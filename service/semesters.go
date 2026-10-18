package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	db "servidor-NotifyEd/db/sqlc"

	"github.com/google/uuid"
)

type SemesterQueries interface {
	CreateSemester(context.Context, db.CreateSemesterParams) (db.Semester, error)
	GetSemesterById(context.Context, uuid.UUID) (db.Semester, error)
	GetSemesterByToken(context.Context, string) (db.GetSemesterByTokenRow, error)
	ListSemestersByProfessor(context.Context, uuid.UUID) ([]db.ListSemestersByProfessorRow, error)
	ListSemestersByCourse(context.Context, uuid.UUID) ([]db.Semester, error)
	UpdateSemesterEndDate(context.Context, db.UpdateSemesterEndDateParams) (db.Semester, error)
	DeleteSemester(context.Context, uuid.UUID) (int64, error)
}

type SemesterService struct {
	queries SemesterQueries
}

func NewSemesterService(queries SemesterQueries) *SemesterService {
	return &SemesterService{queries: queries}
}

func (s *SemesterService) Create(ctx context.Context, params db.CreateSemesterParams) (db.Semester, error) {
	if params.IDCourse == uuid.Nil || params.ProfessorID == uuid.Nil || params.Year <= 0 {
		return db.Semester{}, fmt.Errorf("%w: materia, profesor y año son obligatorios", ErrInvalidInput)
	}
	if params.EndDate.IsZero() {
		return db.Semester{}, fmt.Errorf("%w: la fecha de cierre es obligatoria", ErrInvalidInput)
	}
	if strings.TrimSpace(params.SubToken) == "" {
		tokenBytes := make([]byte, 16)
		if _, err := rand.Read(tokenBytes); err != nil {
			return db.Semester{}, fmt.Errorf("generar token de suscripción: %w", err)
		}
		params.SubToken = base64.RawURLEncoding.EncodeToString(tokenBytes)
	} else {
		params.SubToken = strings.TrimSpace(params.SubToken)
		if len(params.SubToken) > 32 {
			return db.Semester{}, fmt.Errorf("%w: el token admite hasta 32 caracteres", ErrInvalidInput)
		}
	}
	semester, err := s.queries.CreateSemester(ctx, params)
	if err != nil {
		return db.Semester{}, mapDatabaseError(err)
	}
	return semester, nil
}

func (s *SemesterService) Get(ctx context.Context, id uuid.UUID) (db.Semester, error) {
	return s.queries.GetSemesterById(ctx, id)
}

func (s *SemesterService) GetByToken(ctx context.Context, token string) (db.GetSemesterByTokenRow, error) {
	return s.queries.GetSemesterByToken(ctx, token)
}

func (s *SemesterService) ListByProfessor(ctx context.Context, professorID uuid.UUID) ([]db.ListSemestersByProfessorRow, error) {
	semesters, err := s.queries.ListSemestersByProfessor(ctx, professorID)
	if semesters == nil && err == nil {
		semesters = []db.ListSemestersByProfessorRow{}
	}
	return semesters, err
}

func (s *SemesterService) ListByCourse(ctx context.Context, courseID uuid.UUID) ([]db.Semester, error) {
	semesters, err := s.queries.ListSemestersByCourse(ctx, courseID)
	if semesters == nil && err == nil {
		semesters = []db.Semester{}
	}
	return semesters, err
}

func (s *SemesterService) UpdateEndDate(ctx context.Context, id uuid.UUID, endDate time.Time) (db.Semester, error) {
	if endDate.IsZero() {
		return db.Semester{}, fmt.Errorf("%w: la fecha de cierre es obligatoria", ErrInvalidInput)
	}
	semester, err := s.queries.UpdateSemesterEndDate(ctx, db.UpdateSemesterEndDateParams{IDSemester: id, EndDate: endDate})
	if err != nil {
		return db.Semester{}, err
	}
	return semester, nil
}

func (s *SemesterService) Delete(ctx context.Context, id uuid.UUID) error {
	rowsAffected, err := s.queries.DeleteSemester(ctx, id)
	if err != nil {
		return mapDatabaseError(err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}
