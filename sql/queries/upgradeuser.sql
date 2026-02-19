-- name: UpgradeUserById :one
UPDATE users
SET
    is_chirpy_red = true
WHERE id = $1
RETURNING *;