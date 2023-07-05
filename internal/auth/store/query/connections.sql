-- name: CreateConnection :one
INSERT INTO connections (provider_id, user_id)
    VALUES ($1, $2)
RETURNING
    *;

-- name: GetConnection :one
SELECT
    *
FROM
    connections
WHERE
    provider_id = $1
LIMIT 1;

-- name: ListUserConnections :many
SELECT
    *
FROM
    connections
WHERE
    user_id = $1
ORDER BY
    provider_id
LIMIT $2 OFFSET $3;

-- name: DeleteConnection :exec
DELETE FROM connections
WHERE provider_id = $1;


-- name: DeleteUserConnections :exec
DELETE FROM connections
WHERE user_id = $1;
