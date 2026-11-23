-- name: CreateNotice :one
INSERT INTO notices (id_semester, id_sender, title, message)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetNoticeById :one
SELECT * FROM notices
WHERE id_notice = $1;

-- name: ListNoticesBySemester :many
-- La que usa la pantalla real: historial de avisos de una cursada,
-- del más reciente al más viejo.
SELECT * FROM notices
WHERE id_semester = $1
ORDER BY created_at DESC;

-- name: ListNoticesWithSender :many
-- Historial con el nombre del profesor que publicó cada aviso.
SELECT
    n.id_notice,
    n.id_semester,
    n.title,
    n.message,
    n.created_at,
    u.fullname AS sender_name
FROM notices n
JOIN users u ON u.id_user = n.id_sender
WHERE n.id_semester = $1
ORDER BY n.created_at DESC;

-- name: ListNotices :many
SELECT * FROM notices
ORDER BY created_at DESC;

-- name: CountNoticesBySemester :one
SELECT COUNT(*) FROM notices
WHERE id_semester = $1;

-- name: UpdateNoticeMessage :one
UPDATE notices
SET message = $2
WHERE id_notice = $1
RETURNING *;

-- name: DeleteNotice :execrows
DELETE FROM notices
WHERE id_notice = $1;

-- name: ListNoticesFiltered :many
SELECT
    n.id_notice,
    n.id_semester,
    n.id_sender,
    n.title,
    n.message,
    n.created_at,
    c.name AS course_name,
    c.id_course,
    s.year AS semester_year
FROM notices n
JOIN semesters s ON s.id_semester = n.id_semester
JOIN courses c ON c.id_course = s.id_course
WHERE n.id_sender = $1
  AND (sqlc.narg('id_semester')::uuid IS NULL OR n.id_semester = sqlc.narg('id_semester'))
  AND (sqlc.narg('id_course')::uuid IS NULL OR c.id_course = sqlc.narg('id_course'))
ORDER BY n.created_at DESC;

