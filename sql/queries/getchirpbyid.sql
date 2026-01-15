-- name: GetChirp :one
SELECT * from chirps
WHERE id = $1;