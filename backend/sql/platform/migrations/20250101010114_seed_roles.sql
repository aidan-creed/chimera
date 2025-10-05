
-- +goose Up
-- Seed the RBAC tables with a granular, scoped role structure

-- 1. Create the roles from your model
INSERT INTO "roles" (name, description) VALUES
('super_admin', 'Has all permissions, including managing other users and assigning admin roles.'),
('admin', 'Can manage data, uploads, and assign non-admin roles to any user.'),
('business_line_admin', 'Can assign analyst and viewer roles to users within their own business line(s).'),
('maintainer', 'Can upload reports and edit data within their assigned scope.'),
('analyst', 'Can view and edit data within their assigned scope.'),
('viewer', 'Has read-only access to dashboards and data within their assigned scope.');

-- 2. Assign existing permissions to the new roles
-- Super Admin gets everything
INSERT INTO "role_permissions" (role_id, permission_id)
SELECT (SELECT id FROM roles WHERE name = 'super_admin'), p.id FROM permissions p;

-- Admin gets everything EXCEPT managing other admins
INSERT INTO "role_permissions" (role_id, permission_id)
SELECT (SELECT id FROM roles WHERE name = 'admin'), p.id FROM permissions p WHERE p.action != 'roles:manage_admins';

-- Business Line Admin gets scoped user management and view rights
INSERT INTO "role_permissions" (role_id, permission_id)
SELECT (SELECT id FROM roles WHERE name = 'business_line_admin'), p.id FROM permissions p WHERE p.action IN ('roles:assign_scoped', 'users:view_scoped', 'items:view_scoped');

-- Maintainer can upload, edit, and view scoped items
INSERT INTO "role_permissions" (role_id, permission_id)
SELECT (SELECT id FROM roles WHERE name = 'maintainer'), p.id FROM permissions p WHERE p.action IN ('reports:upload', 'items:edit_scoped', 'items:view_scoped');

-- Analyst can edit and view scoped items
INSERT INTO "role_permissions" (role_id, permission_id)
SELECT (SELECT id FROM roles WHERE name = 'analyst'), p.id FROM permissions p WHERE p.action IN ('items:edit_scoped', 'items:view_scoped');

-- Viewer can only view scoped items
INSERT INTO "role_permissions" (role_id, permission_id)
SELECT (SELECT id FROM roles WHERE name = 'viewer'), p.id FROM permissions p WHERE p.action = 'items:view_scoped';


-- +goose Down
-- Clear out all the seeded data in reverse order of creation
DELETE FROM "role_permissions";
DELETE FROM "roles";
