-- name: CreateUser :one
INSERT INTO users (fullname, email, password_hash)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetUserById :one
SELECT * FROM users
WHERE id_user = $1;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1;

-- name: ListUsers :many
SELECT id_user, fullname, email FROM users
ORDER BY fullname;

-- name: UpdateUserFullname :one
UPDATE users
SET fullname = $2
WHERE id_user = $1
RETURNING *;

-- name: UpdateUserPassword :execrows
UPDATE users
SET password_hash = $2
WHERE id_user = $1;

-- name: DeleteUser :execrows
DELETE FROM users
WHERE id_user = $1;