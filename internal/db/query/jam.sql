-- name: CreateJam :one
INSERT INTO jam (name, bpm, capacity)
    VALUES ($1, $2, $3)
RETURNING
    *;

-- name: GetJam :one
SELECT
    *
FROM
    jam
WHERE
    id = $1
LIMIT 1;

-- name: ListJams :many
SELECT
    *
FROM
    jam
ORDER BY
    OWNER
LIMIT $1 OFFSET $2;

-- name: UpdateJam :one
UPDATE
    jam
SET
    bpm = $2
WHERE
    id = $1
RETURNING
    *;

-- name: DeleteJam :exec
DELETE FROM jam
WHERE id = $1;

