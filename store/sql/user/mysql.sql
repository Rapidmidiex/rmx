CREATE TABLE users (
    id text NOT NULL PRIMARY KEY,
    username text NOT NULL,
    email text NOT NULL,
    password text NOT NULL,
    created_at timestamp NOT NULL DEFAULT NOW(),
    updated_at timestamp NULL DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
    deleted_at timestamp NULL DEFAULT NULL,
    UNIQUE (email)
);

-- name: GetUserByID :one
SELECT
    *
FROM
    users
WHERE
    id = ?
LIMIT 1;

-- name: GetUserByEmail :one
SELECT
    *
FROM
    users
WHERE
    email = ?
LIMIT 1;

-- name: ListUsers :many
SELECT
    *
FROM
    users
ORDER BY
    id;

-- name: CreateUser :execresult
INSERT INTO users (username, email, PASSWORD, created_at, updated_at, deleted_at)
    VALUES (?, ?, ?, ?, ?, ?);

-- name: UpdateUser :execresult
UPDATE
    users
SET
    username = ?
WHERE
    id = ?;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = ?;

