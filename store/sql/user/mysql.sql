CREATE TABLE "users" (
    id text NOT NULL PRIMARY KEY,
    username text NOT NULL,
    email text NOT NULL,
    password text NOT NULL,
    created_at timestamp NOT NULL DEFAULT NOW(),
    UNIQUE (email)
);

-- name: SelectByID :one
SELECT
    *
FROM
    users
WHERE
    id = $1
LIMIT 1;

-- name: SelectByEmail :one
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
    id;

-- name: CreateUser :execresult
INSERT INTO users (username, email, PASSWORD, created_at)
    VALUES ($1, $2, $3, $4);

-- name: UpdateUser :execresult
UPDATE
    users
SET
    username = $1
WHERE
    id = $2;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;

