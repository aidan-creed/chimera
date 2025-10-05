-- +goose Up
-- Create all custom ENUM types used in schema

-- A generic status for item in any application
CREATE TYPE item_status AS ENUM (
	'active',
	'inactive',
	'archived'
);

-- Define the 'application' or type of data stored in the items table
CREATE TYPE item_type AS ENUM (
	'KNOWLEDGE_CHUNK'
);

-- Hint for SQLC: Creating empty placeholders for views that have complex dependencies

-- +goose Down
DROP TYPE IF EXISTS item_type;
DROP TYPE IF EXISTS item_status;

