package views

import (
	db "servidor-NotifyEd/db/sqlc"

	"github.com/google/uuid"
)

// FilterSemestersByCourse filtra las cursadas pertenecientes a una materia específica.
func FilterSemestersByCourse(semesters []db.ListSemestersWithCourseByProfessorRow, courseID uuid.UUID) []db.ListSemestersWithCourseByProfessorRow {
	var filtered []db.ListSemestersWithCourseByProfessorRow
	for _, s := range semesters {
		if s.IDCourse == courseID {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

// ExtractCoursesFromSemesters devuelve la lista de materias únicas (sin repetidos)
// a partir de las cursadas del profesor, preservando el orden recibido.
func ExtractCoursesFromSemesters(semesters []db.ListSemestersWithCourseByProfessorRow) []db.Course {
	seen := make(map[uuid.UUID]bool)
	var courses []db.Course
	for _, s := range semesters {
		if !seen[s.IDCourse] {
			seen[s.IDCourse] = true
			courses = append(courses, db.Course{
				IDCourse: s.IDCourse,
				Name:     s.CourseName,
			})
		}
	}
	return courses
}
