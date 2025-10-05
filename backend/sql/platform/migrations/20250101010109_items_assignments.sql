-- +goose Up
-- Create the "item_assignments" table

CREATE TABLE "item_assignments" (
    "id" BIGSERIAL PRIMARY KEY,
    "item_id" BIGINT NOT NULL REFERENCES "items"("id") ON DELETE CASCADE,
    "user_id" BIGINT NOT NULL REFERENCES "users"("id") ON DELETE CASCADE,
    "assigned_as_role" VARCHAR(100) NOT NULL, 
    "assigned_at" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "is_active" BOOLEAN NOT NULL DEFAULT TRUE,
    UNIQUE ("item_id", "user_id", "assigned_as_role")
);

-- +goose Down
-- Drop the "item_assignments" table

DROP TABLE IF EXISTS "item_assignments";
