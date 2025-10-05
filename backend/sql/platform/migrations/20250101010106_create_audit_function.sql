-- +goose Up
-- +goose StatementBegin
-- Create a generic audit trigger function that captures old and new row data

CREATE OR REPLACE FUNCTION audit.if_modified_func() RETURNS TRIGGER AS $$
DECLARE
    audit_table_name TEXT;
    target_id_column TEXT;
    target_id_value BIGINT;
    -- For UPDATE operations, these variables will hold the old and new row data
    old_row_jsonb JSONB := NULL;
    new_row_jsonb JSONB := NULL;
    -- Optional: Variable to store the user who made the change
    current_user_id BIGINT := NULL; -- Adjust to BIGINT if user.id is BIGINT
BEGIN
    audit_table_name := TG_TABLE_NAME || '_changes';
    target_id_column := 'id'; -- Assuming all audited tables have an 'id' column as PK

    IF TG_OP = 'UPDATE' THEN
        old_row_jsonb := to_jsonb(OLD);
        new_row_jsonb := to_jsonb(NEW);
        target_id_value := NEW.id;
    ELSIF TG_OP = 'INSERT' THEN
        new_row_jsonb := to_jsonb(NEW);
        target_id_value := NEW.id;
    ELSIF TG_OP = 'DELETE' THEN
        old_row_jsonb := to_jsonb(OLD);
        target_id_value := OLD.id;
    END IF;

    -- Attempt to get the current user ID if it's set in the session
    -- Ensure 'app.user_id' is set in your application session: SET app.user_id = 'your-user-uuid';
    BEGIN
        current_user_id := current_setting('app.user_id', true)::BIGINT; -- Adjust to BIGINT if user.id is BIGINT
    EXCEPTION WHEN OTHERS THEN
        -- If app.user_id is not set or not a valid BIGINT, current_user_id remains NULL
        current_user_id := NULL;
    END;

    -- Dynamically insert into the correct audit table
    EXECUTE format('INSERT INTO audit.%I ('
                   'target_id, operation, changed_by, changed_at, old_data, new_data)'
                   ' VALUES ($1, $2, $3, $4, $5, $6)', audit_table_name)
    USING target_id_value,
          substring(TG_OP, 1, 1), -- 'I', 'U', 'D'
          current_user_id,
          NOW(),
          old_row_jsonb,
          new_row_jsonb;

    RETURN NEW;
END;
$$
LANGUAGE plpgsql
SECURITY DEFINER;

-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
-- Drop the generic audit trigger function

DROP FUNCTION IF EXISTS audit.if_modified_func() CASCADE;
-- +goose StatementEnd
