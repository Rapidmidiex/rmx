-- name: GetUserByID :one
SELECT *
FROM users
WHERE id = ?
LIMIT 1;

-- name: GetUserByEmail :one
SELECT *
FROM users
WHERE email = ?
LIMIT 1;

-- name: ListUsers :many
SELECT *
FROM users
ORDER BY id;

-- name: CreateUser :execresult
INSERT INTO users (
        username,
        email,
        password,
        created_at,
        updated_at,
        deleted_at
    )
VALUES (?, ?, ?, ?, ?, ?);

-- name: UpdateUser :execresult
UPDATE users
    SET username = ?
    WHERE id = ?;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = ?;
