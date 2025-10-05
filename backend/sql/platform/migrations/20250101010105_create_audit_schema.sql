-- +goose Up
-- Create the audit schema to house audit-related tables and functions

CREATE SCHEMA audit;

-- +goose Down
-- Drop the audit schema and all its contents

DROP SCHEMA IF EXISTS audit CASCADE;
