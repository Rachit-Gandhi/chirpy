-- name: GetUserByRefreshToken :one
SELECT * FROM refresh_tokens
where token = $1;