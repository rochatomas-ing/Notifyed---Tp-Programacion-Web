-- name: CreateCourse :one
INSERT INTO courses (name)
VALUES ($1)
RETURNING *;

-- name: GetCourseById :one
SELECT * FROM courses
WHERE id_course = $1;

-- name: GetCourseByName :one
SELECT * FROM courses
WHERE name = $1;

-- name: ListCourses :many
SELECT * FROM courses
ORDER BY name;

-- name: UpdateCourseName :one
UPDATE courses
SET name = $2
WHERE id_course = $1
RETURNING *;

-- name: DeleteCourse :execrows
DELETE FROM courses
WHERE id_course = $1;