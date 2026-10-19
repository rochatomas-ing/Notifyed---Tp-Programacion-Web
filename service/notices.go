package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	db "servidor-NotifyEd/db/sqlc"

	"github.com/google/uuid"
)

// NoticeQueries es la parte del repositorio sqlc que necesita el service.
type NoticeQueries interface {
	CreateNotice(context.Context, db.CreateNoticeParams) (db.Notice, error)
	GetNoticeById(context.Context, uuid.UUID) (db.Notice, error)
	ListNotices(context.Context) ([]db.Notice, error)
	UpdateNoticeMessage(context.Context, db.UpdateNoticeMessageParams) (db.Notice, error)
	DeleteNotice(context.Context, uuid.UUID) (int64, error)
	GetUserById(context.Context, uuid.UUID) (db.User, error)
}

type NoticeService struct {
	queries NoticeQueries
}

func NewNoticeService(queries NoticeQueries) *NoticeService {
	return &NoticeService{queries: queries}
}

// Create valida el aviso y lo persiste. El alumno no tiene cuenta, así que
// id_sender siempre apunta a un profesor: se verifica que exista antes de
// insertar.
func (s *NoticeService) Create(ctx context.Context, params db.CreateNoticeParams) (db.Notice, error) {
	params.Title = strings.TrimSpace(params.Title)
	params.Message = strings.TrimSpace(params.Message)
	if params.Title == "" || params.Message == "" {
		return db.Notice{}, fmt.Errorf("%w: el título y el mensaje son obligatorios", ErrInvalidInput)
	}
	if params.IDSemester == uuid.Nil {
		return db.Notice{}, fmt.Errorf("%w: id_semester es obligatorio", ErrInvalidInput)
	}
	if _, err := s.queries.GetUserById(ctx, params.IDSender); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return db.Notice{}, fmt.Errorf("%w: el profesor indicado no existe", ErrInvalidInput)
		}
		return db.Notice{}, err
	}
	notice, err := s.queries.CreateNotice(ctx, params)
	if err != nil {
		return db.Notice{}, mapDatabaseError(err)
	}
	return notice, nil
}

func (s *NoticeService) Get(ctx context.Context, id uuid.UUID) (db.Notice, error) {
	return s.queries.GetNoticeById(ctx, id)
}

func (s *NoticeService) List(ctx context.Context) ([]db.Notice, error) {
	notices, err := s.queries.ListNotices(ctx)
	if notices == nil && err == nil {
		notices = []db.Notice{}
	}
	return notices, err
}

// UpdateMessage sólo actualiza el mensaje: UpdateNoticeMessage no toca el
// título (decisión documentada del contrato del TP).
func (s *NoticeService) UpdateMessage(ctx context.Context, id uuid.UUID, message string) (db.Notice, error) {
	message = strings.TrimSpace(message)
	if message == "" {
		return db.Notice{}, fmt.Errorf("%w: el mensaje no puede estar vacío", ErrInvalidInput)
	}
	notice, err := s.queries.UpdateNoticeMessage(ctx, db.UpdateNoticeMessageParams{
		IDNotice: id,
		Message:  message,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return db.Notice{}, err
		}
		return db.Notice{}, mapDatabaseError(err)
	}
	return notice, nil
}

func (s *NoticeService) Delete(ctx context.Context, id uuid.UUID) error {
	rowsAffected, err := s.queries.DeleteNotice(ctx, id)
	if err != nil {
		return mapDatabaseError(err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}
