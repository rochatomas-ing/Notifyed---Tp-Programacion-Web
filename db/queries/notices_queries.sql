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