-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = $1;

-- name: CreateUser :one
-- source 由调用方决定（admin/builtIn/LDAP）；DB CHECK 约束限定取值。
INSERT INTO users (id, username, password_hash, display_name, email, phone, carrier, status, source)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: UpdateUser :one
UPDATE users
SET display_name = COALESCE($2, display_name),
    email = COALESCE($3, email),
    phone = COALESCE($4, phone),
    carrier = COALESCE($5, carrier),
    status = COALESCE($6, status),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdatePassword :exec
UPDATE users
SET password_hash = $2, updated_at = NOW()
WHERE id = $1;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- name: UpdateLastLogin :exec
UPDATE users
SET last_login_at = NOW(), failed_login_attempts = 0, locked_until = NULL
WHERE id = $1;

-- name: ListUsers :many
SELECT * FROM users
WHERE (carrier = $1 OR $1 IS NULL)
  AND (status = $2 OR $2 IS NULL)
  AND (username ILIKE '%' || $3 || '%' OR $3 = '')
ORDER BY created_at DESC
LIMIT $4 OFFSET $5;

-- name: CountUsers :one
SELECT COUNT(*) FROM users
WHERE (carrier = $1 OR $1 IS NULL)
  AND (status = $2 OR $2 IS NULL)
  AND (username ILIKE '%' || $3 || '%' OR $3 = '');
