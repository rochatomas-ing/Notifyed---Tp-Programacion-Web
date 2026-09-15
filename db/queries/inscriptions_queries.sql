-- name: CreateInscription :one
-- Si el email ya estaba inscripto en esa cursada, lo reactiva en lugar de
-- fallar por la UNIQUE. Cubre el caso de volver a escanear el QR después de
-- haberse dado de baja.
INSERT INTO inscriptions (id_semester, email)
VALUES ($1, $2)
ON CONFLICT (id_semester, email)
DO UPDATE SET active = true
RETURNING *;

-- name: GetInscriptionById :one
SELECT * FROM inscriptions
WHERE id_inscription = $1;

-- name: GetInscriptionByEmailAndSemester :one
-- Decide qué mostrar en la pantalla del link: alta, baja o reactivación.
SELECT * FROM inscriptions
WHERE id_semester = $1 AND email = $2;

-- name: ListInscriptionsBySemester :many
SELECT * FROM inscriptions
WHERE id_semester = $1
ORDER BY created_at;

-- name: ListActiveEmailsBySemester :many
-- Destinatarios de un aviso. Filtra por suscripción activa y por cursada
-- todavía vigente: el cierre por fecha se resuelve acá, sin proceso aparte.
SELECT i.email
FROM inscriptions i
JOIN semesters s ON s.id_semester = i.id_semester
WHERE i.id_semester = $1
  AND i.active
  AND s.end_date >= CURRENT_DATE
ORDER BY i.email;

-- name: CountActiveBySemester :one
SELECT COUNT(*) FROM inscriptions
WHERE id_semester = $1 AND active;

-- name: SetInscriptionActive :execrows
-- Un solo update para baja (false) y reactivación (true).
UPDATE inscriptions
SET active = $3
WHERE id_semester = $1 AND email = $2;

-- name: DeleteInscription :execrows
DELETE FROM inscriptions
WHERE id_inscription = $1;