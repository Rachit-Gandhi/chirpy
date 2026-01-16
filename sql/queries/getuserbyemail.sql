-- name: GetUserByEmail :one
SELECT * FROM users
where email = $1;