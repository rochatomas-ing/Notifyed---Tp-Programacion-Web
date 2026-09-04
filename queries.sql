
-- name: CreateNotice :one
INSERT INTO notices (id_semester, id_sender, title, message)
VALUES($1, $2 $3, $4)
RETURNING id_notice, id_semester, id_sender, title, message, created_at;

-- name: GetNoticeById :one
SELECT id_notice, id_sender, id_semester, title, message, created_at
FROM notices
WHERE id_notice = $1;

-- name: ListNotices :many
SELECT id_notice, id_sender, id_semester, title, message, created_at
FROM notices
ORDER BY created_at; 

-- name: UpdateNoticeMessage :exec
UPDATE notices
SET message = $2
WHERE id_notice = $1;

-- name: DeleteNotice :one
DELETE FROM notices
WHERE id_notice = $1;


