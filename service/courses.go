package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	db "servidor-NotifyEd/db/sqlc"

	"github.com/google/uuid"
)

type CourseQueries interface {
	CreateCourse(context.Context, string) (db.Course, error)
	GetCourseById(context.Context, uuid.UUID) (db.Course, error)
	ListCourses(context.Context) ([]db.Course, error)
	UpdateCourseName(context.Context, db.UpdateCourseNameParams) (db.Course, error)
	DeleteCourse(context.Context, uuid.UUID) (int64, error)
}

type CourseService struct {
	queries CourseQueries
}

func NewCourseService(queries CourseQueries) *CourseService {
	return &CourseService{queries: queries}
}

func (s *CourseService) Create(ctx context.Context, name string) (db.Course, error) {
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > 40 {
		return db.Course{}, fmt.Errorf("%w: el nombre de la materia es obligatorio y debe tener hasta 40 caracteres", ErrInvalidInput)
	}
	course, err := s.queries.CreateCourse(ctx, name)
	if err != nil {
		return db.Course{}, mapDatabaseError(err)
	}
	return course, nil
}

func (s *CourseService) Get(ctx context.Context, id uuid.UUID) (db.Course, error) {
	return s.queries.GetCourseById(ctx, id)
}

func (s *CourseService) List(ctx context.Context) ([]db.Course, error) {
	courses, err := s.queries.ListCourses(ctx)
	if courses == nil && err == nil {
		courses = []db.Course{}
	}
	return courses, err
}

func (s *CourseService) UpdateName(ctx context.Context, id uuid.UUID, name string) (db.Course, error) {
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > 40 {
		return db.Course{}, fmt.Errorf("%w: el nombre de la materia es obligatorio y debe tener hasta 40 caracteres", ErrInvalidInput)
	}
	course, err := s.queries.UpdateCourseName(ctx, db.UpdateCourseNameParams{IDCourse: id, Name: name})
	if err != nil {
		if err == sql.ErrNoRows {
			return db.Course{}, err
		}
		return db.Course{}, mapDatabaseError(err)
	}
	return course, nil
}

func (s *CourseService) Delete(ctx context.Context, id uuid.UUID) error {
	rowsAffected, err := s.queries.DeleteCourse(ctx, id)
	if err != nil {
		return mapDatabaseError(err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}
