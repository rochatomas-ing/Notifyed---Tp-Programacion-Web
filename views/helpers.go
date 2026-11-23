package views

import (
	db "servidor-NotifyEd/db/sqlc"

	"github.com/google/uuid"
)

func filterSemestersByCourse(semesters []db.ListSemestersWithCourseByProfessorRow, courseID uuid.UUID) []db.ListSemestersWithCourseByProfessorRow {
	var filtered []db.ListSemestersWithCourseByProfessorRow
	for _, s := range semesters {
		if s.IDCourse == courseID {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

