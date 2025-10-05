-- +goose Up
-- Create the audit table for items changes

CREATE TABLE audit.items_changes (
    audit_id BIGSERIAL PRIMARY KEY,
    target_id BIGINT NOT NULL,
    operation CHAR(1) NOT NULL,
    changed_by BIGINT,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    old_data JSONB,
    new_data JSONB
);

-- Add index for efficient lookup by item ID
CREATE INDEX idx_audit_items_target_id ON audit.items_changes (target_id);
-- Add index for chronological ordering
CREATE INDEX idx_audit_items_changed_at ON audit.items_changes (changed_at DESC);

-- Audit trigger to the items table
CREATE TRIGGER items_audit_trigger
AFTER INSERT OR UPDATE OR DELETE ON "items"
FOR EACH ROW EXECUTE FUNCTION audit.if_modified_func();

-- +goose Down
DROP TRIGGER IF EXISTS items_audit_trigger ON "items";
DROP TABLE IF EXISTS audit.items_changes;
