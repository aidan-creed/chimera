-- name: ListRoles :many
-- Fetch all available roles in system
SELECT id, name, description FROM "roles" ORDER BY id;

-- name: AssignRoleToUser :exec
-- Assign a specific role to a user
INSERT INTO "user_roles" (user_id, role_id) VALUES ($1, $2)
ON CONFLICT (user_id, role_id) DO NOTHING;

-- name: RemoveRoleFromUser :exec
-- Removes a specific role from a user
DELETE FROM "user_roles" WHERE user_id = $1 AND role_id = $2;

-- name: AssignScopeToUser :exec
-- Grants a user access to a specific scope
INSERT INTO "user_scope_access" (user_id, scope) VALUES ($1, $2)
ON CONFLICT (user_id, scope) DO NOTHING;

-- name: RemoveScopeFromUser :exec
--Revokes a user's access from a specific scope.
DELETE FROM "user_scope_access" WHERE user_id = $1 AND scope = $2;

-- name: RemoveAllRolesFromUser :exec
-- Removes all roles from a user. Useful when completely re-assigning roles
DELETE FROM "user_roles" WHERE user_id = $1;

-- name: RemoveAllScopesFromUser :exec
-- Removes all scope access from a user
DELETE FROM "user_scope_access" WHERE user_id = $1;



-- name: SetUserAdminStatus :one
-- Updates only the is_admin status of a specific user
-- This is a priviliged action and should be protected at API layer
UPDATE "users"
SET
	is_admin = $2
WHERE
	id = $1
RETURNING *;

-- name: CreateUserFromAuthProvider :one
-- Creates a new user record from the authentication provider's details
INSERT INTO "users" (
	auth_provider_subject,
	email,
	display_name,
	is_active,
	is_admin
) VALUES (
	$1, $2, $3, TRUE, FALSE
)
RETURNING *;

-- name: GetUserByAuthProviderSubject :one
-- Fetch a single user by their external auth provider ID
SELECT * FROM "users" WHERE auth_provider_subject = $1;

-- name: UpdateUser :one
-- Updates a user's mutable details
UPDATE "users"
SET
	display_name = $2,
	is_active = $3
WHERE
	id = $1
RETURNING *;


