package service

import (
	"context"
	"database/sql"
	"fmt"

	db "servidor-NotifyEd/db/sqlc"

	"github.com/google/uuid"
)

type UsersQueries interface {
	ListUsers(ctx context.Context) ([]db.ListUsersRow, error)
	GetUserById(ctx context.Context, idUser uuid.UUID) (db.User, error)
	GetUserByEmail(ctx context.Context, email string) (db.User, error)
	CreateUser(ctx context.Context, user db.CreateUserParams) (db.User, error)
	UpdateUserFullname(ctx context.Context, user db.UpdateUserFullnameParams) (db.User, error)
	UpdateUserPassword(ctx context.Context, user db.UpdateUserPasswordParams) (db.User, error)
	DeleteUser(ctx context.Context, idUser uuid.UUID) (int64, error)
}

type UsersService struct {
	queries UsersQueries
}

func NewUsersService(queries UsersQueries) *UsersService {
	return &UsersService{queries: queries}
}

func (s *UsersService) List(ctx context.Context) ([]db.ListUsersRow, error) {
	usuarios, err := s.queries.ListUsers(ctx)
	if usuarios == nil && err == nil {
		usuarios = []db.ListUsersRow{}
	}
	return usuarios, err
}

func (s *UsersService) Get(ctx context.Context, idUser uuid.UUID) (db.User, error) {
	usuario, err := s.queries.GetUserById(ctx, idUser)
	if err != nil {
		return db.User{}, mapDatabaseError(err)
	}
	return usuario, nil
}

func (s *UsersService) Create(ctx context.Context, user db.CreateUserParams) (db.User, error) {
	if user.Fullname == "" || user.Email == "" || user.PasswordHash == "" {
		return db.User{}, fmt.Errorf("%w: todos los campos son obligatorios", ErrInvalidInput)
	}
	usuario, err := s.queries.CreateUser(ctx, user)
	if err != nil {
		return db.User{}, mapDatabaseError(err)
	}
	return usuario, nil
}

func (s *UsersService) UpdateFullname(ctx context.Context, user db.UpdateUserFullnameParams) (db.User, error) {
	if user.Fullname == "" {
		return db.User{}, fmt.Errorf("%w: el nombre completo es obligatorio", ErrInvalidInput)
	}
	usuario, err := s.queries.UpdateUserFullname(ctx, user)
	if err != nil {
		return db.User{}, mapDatabaseError(err)
	}
	return usuario, nil
}

func (s *UsersService) UpdatePassword(ctx context.Context, user db.UpdateUserPasswordParams) (db.User, error) {
	if user.PasswordHash == "" {
		return db.User{}, fmt.Errorf("%w: la contraseña es obligatoria", ErrInvalidInput)
	}
	usuario, err := s.queries.UpdateUserPassword(ctx, user)
	if err != nil {
		return db.User{}, mapDatabaseError(err)
	}
	return usuario, nil
}

func (s *UsersService) Delete(ctx context.Context, idUser uuid.UUID) error {
	rowsAffected, err := s.queries.DeleteUser(ctx, idUser)
	if err != nil {
		return mapDatabaseError(err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}
