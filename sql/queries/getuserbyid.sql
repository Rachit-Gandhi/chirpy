-- name: GetUserById :one
SELECT * FROM users
where id = $1;