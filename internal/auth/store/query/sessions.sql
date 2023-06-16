-- name: CreateSession :one
INSERT INTO sessions (email, issuer)
    VALUES ($1, $2)
RETURNING
    *;

-- name: GetSessionByID :one
SELECT
    *
FROM
    sessions
WHERE
    id = $1
LIMIT 1;

-- name: GetSessionsByEmail :many
SELECT
    *
FROM
    sessions
WHERE
    email = $1
LIMIT 1;

-- name: DeleteSessionByID :exec
DELETE FROM sessions
WHERE id = $1;

-- name: DeleteSessionsByEmail :exec
DELETE FROM sessions
WHERE email = $1;