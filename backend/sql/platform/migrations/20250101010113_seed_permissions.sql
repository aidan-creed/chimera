
-- +goose Up
-- Seed the generic permissions that the Chimera platform understands.

INSERT INTO "permissions" (action, description) VALUES
('roles:manage_admins', 'Ability to assign the admin or super_admin roles.'),
('roles:assign_global', 'Ability to assign non-admin roles to any user.'),
('roles:assign_scoped', 'Ability to assign non-admin roles to users within the same business line(s).'),
('users:edit', 'Ability to edit a user''s details (e.g., active status).'),
('users:view_scoped', 'Ability to view users within the same business line(s).'),
('reports:upload', 'Ability to upload documents.'),
('items:edit_all', 'Ability to edit items of any type across all scopes.'),
('items:edit_scoped', 'Ability to edit items only within their assigned scope(s).'),
('items:view_all', 'Ability to view all item data across all scopes.'),
('items:view_scoped', 'Ability to view items only within their assigned scope(s).');

-- +goose Down
-- Clear out all the seeded data
DELETE FROM "permissions";
