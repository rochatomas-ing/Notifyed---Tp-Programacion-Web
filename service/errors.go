package service

import (
	"errors"
	"fmt"

	"github.com/lib/pq"
)

var (
	ErrInvalidInput = errors.New("datos de entrada inválidos")
	ErrConflict     = errors.New("conflicto con un recurso existente o relacionado")
)

func mapDatabaseError(err error) error {
	var postgresError *pq.Error
	if errors.As(err, &postgresError) {
		switch postgresError.Code {
		case "23505", "23503": // UNIQUE o FK: clave duplicada o recurso relacionado.
			return fmt.Errorf("%w: %s", ErrConflict, postgresError.Constraint)
		}
	}
	return err
}
