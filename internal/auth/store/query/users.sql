-- name: CreateUser :one
INSERT INTO users (username, email, email_verified, is_admin, picture, blocked)
    VALUES ($1, $2, $3, $4, $5, $6)
RETURNING
    *;

-- name: GetUserByID :one
SELECT
    *
FROM
    users
WHERE
    id = $1
LIMIT 1;

-- name: GetUserByUsername :one
SELECT
    *
FROM
    users
WHERE
    username = $1
LIMIT 1;

-- name: GetUserByEmail :one
SELECT
    *
FROM
    users
WHERE
    email = $1
LIMIT 1;

-- name: ListUsers :many
SELECT
    *
FROM
    users
ORDER BY
    username
LIMIT $1 OFFSET $2;

-- name: UpdateUserByID :one
UPDATE
    users
SET
    username = $2,
    email = $3,
    email_verified = $4,
    is_admin = $5,
    picture = $6,
    blocked = $7,
    last_login = $8
WHERE
    id = $1
RETURNING
    *;

-- name: UpdateUserByUsername :one
UPDATE
    users
SET
    email = $2,
    email_verified = $3,
    is_admin = $4,
    picture = $5,
    blocked = $6,
    last_login = $7
WHERE
    username = $1
RETURNING
    *;

-- name: UpdateUserByEmail :one
UPDATE
    users
SET
    username = $2,
    email_verified = $3,
    is_admin = $4,
    picture = $5,
    blocked = $6,
    last_login = $7
WHERE
    email = $1
RETURNING
    *;

-- name: DeleteUserByID :exec
DELETE FROM users
WHERE id = $1;

-- name: DeleteUserByUsername :exec
DELETE FROM users
WHERE username = $1;

-- name: DeleteUserByEmail :exec
DELETE FROM users
WHERE email = $1;

