-- name: CreateUser :one
INSERT INTO "user" (username, email)
    VALUES ($1, $2)
RETURNING
    *;

-- name: GetUserByID :one
SELECT
    *
FROM
    "user"
WHERE
    id = $1
LIMIT 1;

-- name: GetUserByEmail :one
SELECT
    *
FROM
    "user"
WHERE
    email = $1
LIMIT 1;

-- name: ListUsers :many
SELECT
    *
FROM
    "user"
ORDER BY
    "name"
LIMIT $1 OFFSET $2;

-- name: UpdateUserByID :one
UPDATE
    "user"
SET
    username = $2
WHERE
    id = $1
RETURNING
    *;

-- name: UpdateUserByEmail :one
UPDATE
    "user"
SET
    username = $2
WHERE
    email = $1
RETURNING
    *;

-- name: DeleteUserByID :exec
DELETE FROM "user"
WHERE id = $1;

-- name: DeleteUserByEmail :exec
DELETE FROM "user"
WHERE email = $1;

