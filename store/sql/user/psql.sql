CREATE TEMP TABLE IF NOT EXISTS "users" (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4 (),
    username text UNIQUE NOT NULL CHECK (username <> ''),
    email citext UNIQUE NOT NULL CHECK (email ~ '^[a-zA-Z0-9.!#$%&’*+/=?^_\x60{|}~-]+@[a-zA-Z0-9-]+(?:\.[a-zA-Z0-9-]+)*$'),
    PASSWORD citext NOT NULL CHECK (PASSWORD <> ''),
    created_at timestamp DEFAULT now()
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

-- name: SelectMany :many
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

