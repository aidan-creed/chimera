-- +goose Up
-- Create the audit table for user changes
CREATE TABLE audit.users_changes (
    audit_id BIGSERIAL PRIMARY KEY,
    target_id BIGINT NOT NULL,
    operation CHAR(1) NOT NULL,
    changed_by BIGINT,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    old_data JSONB,
    new_data JSONB
);

-- Add index for efficient lookup by user ID
CREATE INDEX idx_audit_user_target_id ON audit.users_changes (target_id);
-- Add index for chronological ordering
CREATE INDEX idx_audit_user_changed_at ON audit.users_changes (changed_at DESC);

-- Add the audit trigger to the user table
CREATE TRIGGER user_audit_trigger
AFTER INSERT OR UPDATE OR DELETE ON "users"
FOR EACH ROW EXECUTE FUNCTION audit.if_modified_func();

-- +goose Down
-- Drop the audit table for user changes

DROP TRIGGER IF EXISTS user_audit_trigger ON "users";
DROP TABLE IF EXISTS audit.users_changes;

