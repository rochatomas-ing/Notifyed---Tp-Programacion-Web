package service

import (
	"errors"
	"fmt"

	"github.com/lib/pq"
)

var (
	ErrInvalidInput = errors.New("datos de entrada inválidos")
	ErrConflict     = errors.New("conflicto con un recurso existente o relacionado")

	// ErrSemesterClosed: no se admite el alta de una inscripción en una
	// cursada cuya end_date ya pasó (decisión #1 del contrato).
	ErrSemesterClosed = errors.New("la cursada está cerrada")

	// ErrInscriptionNotFound: no hay inscripción con ese email en la cursada
	// (decisiones #2 y #3 del contrato).
	ErrInscriptionNotFound = errors.New("no hay inscripción con ese email")
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
