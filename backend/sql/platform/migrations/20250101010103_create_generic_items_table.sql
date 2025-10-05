-- +goose Up
-- +goose StatementBegin

-- Create function to set updated_at timestamp
CREATE OR REPLACE FUNCTION trigger_set_timestamp()
RETURNS TRIGGER AS $$
BEGIN
	NEW.updated_at = NOW();
	RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Enabling pgvector extension 
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE "items" (
	"id" BIGSERIAL PRIMARY KEY,
	"item_type" item_type NOT NULL,
	"scope" VARCHAR(100),
	"business_key" VARCHAR(100),
	"status" item_status NOT NULL DEFAULT 'active',
	"custom_properties" JSONB NOT NULL DEFAULT '{}'::jsonb,

	-- RAG PREPARATION on TABLE
	"embedding" vector(384),

	"created_at" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	"updated_at" TIMESTAMPTZ NOT NULL DEFAULT NOW(),

	-- Define the business key constraint
	CONSTRAINT items_business_key UNIQUE (item_type, business_key)
);

-- Indexes 
CREATE INDEX idx_items_item_type ON items(item_type);
CREATE INDEX idx_items_scope ON items(scope);
CREATE INDEX idx_items_items_custom_properties ON items USING GIN (custom_properties);

-- RAG PREPARATION: Vector Index
CREATE INDEX idx_items_embedding ON items USING HNSW (embedding vector_cosine_ops);

-- Add trigger to items table
CREATE TRIGGER set_timestamp
BEFORE UPDATE ON items
FOR EACH ROW
EXECUTE PROCEDURE trigger_set_timestamp();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "items";
DROP FUNCTION IF EXISTS trigger_set_timestamp();
-- +goose StatementEnd
