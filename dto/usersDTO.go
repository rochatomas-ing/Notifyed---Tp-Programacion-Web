package dto

import (
	"github.com/google/uuid"
)

type UserDTO struct {
	IDUser   uuid.UUID `json:"id_user"`
	Fullname string    `json:"fullname"`
	Email    string    `json:"email"`
}

func ToUserDTO(id uuid.UUID, fullname string, email string) UserDTO {
	return UserDTO{
		IDUser:   id,
		Fullname: fullname,
		Email:    email,
	}
}

// CreateUserRequest modela el body de POST /users. La password llega en texto
// plano; el service la hashea con bcrypt antes de tocar la base de datos.
type CreateUserRequest struct {
	Fullname string `json:"fullname"`
	Email    string `json:"email"`
	Password string `json:"password"`
}
