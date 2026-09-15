-- name: CreateSemester :one
INSERT INTO semesters (id_course, professor_id, year, sub_token, end_date)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetSemesterById :one
SELECT * FROM semesters
WHERE id_semester = $1;

-- name: GetSemesterByToken :one
-- Resuelve el QR/link de suscripción. Devuelve también el nombre de la materia
-- y si la cursada sigue abierta, que es lo que la pantalla necesita mostrar.
SELECT
    s.id_semester,
    s.id_course,
    s.professor_id,
    s.year,
    s.sub_token,
    s.end_date,
    c.name AS course_name,
    (s.end_date >= CURRENT_DATE) AS is_open
FROM semesters s
JOIN courses c ON c.id_course = s.id_course
WHERE s.sub_token = $1;

-- name: ListSemestersByProfessor :many
-- Panel del profesor: sus cursadas con el nombre de la materia y el conteo
-- de suscriptos activos.
SELECT
    s.id_semester,
    s.year,
    s.sub_token,
    s.end_date,
    c.name AS course_name,
    COUNT(i.id_inscription) FILTER (WHERE i.active) AS active_subscribers
FROM semesters s
JOIN courses c ON c.id_course = s.id_course
LEFT JOIN inscriptions i ON i.id_semester = s.id_semester
WHERE s.professor_id = $1
GROUP BY s.id_semester, c.name
ORDER BY s.year DESC, c.name;

-- name: ListSemestersByCourse :many
SELECT * FROM semesters
WHERE id_course = $1
ORDER BY year DESC;

-- name: UpdateSemesterEndDate :one
-- Sirve tanto para corregir la fecha como para cerrar la cursada antes de
-- tiempo, poniendo la fecha de hoy.
UPDATE semesters
SET end_date = $2
WHERE id_semester = $1
RETURNING *;

-- name: DeleteSemester :execrows
DELETE FROM semesters
WHERE id_semester = $1;