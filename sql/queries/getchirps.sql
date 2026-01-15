-- name: GetChirps :many
SELECT * from chirps
ORDER BY created_at;