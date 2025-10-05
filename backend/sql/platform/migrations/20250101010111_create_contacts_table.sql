-- +goose Up
-- The contacts table holds information for individuals who are not system users
CREATE TABLE "contacts" (
	"id" BIGSERIAL PRIMARY KEY,
	"display_name" VARCHAR(255) NOT NULL,
	"first_name" VARCHAR(100),
	"last_name" VARCHAR(100),
	"email" VARCHAR(255) UNIQUE,
	"phone" VARCHAR(50),
	"last_seen_at" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	"created_at" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Add index for faster searching by name
CREATE INDEX idx_contacts_display_name ON "contacts" ("display_name");

-- The item_contacts table links contacts to items.
CREATE TABLE "item_contacts" (
	"id" BIGSERIAL PRIMARY KEY,
	"item_id" BIGINT NOT NULL REFERENCES "items"("id") ON DELETE CASCADE,
	"contact_id" BIGINT NOT NULL REFERENCES "contacts"("id") ON DELETE CASCADE,
	"association_type" VARCHAR(100), 
	UNIQUE ("item_id", "contact_id", "association_type")
);

-- +goose Down
DROP TABLE IF EXISTS "item_contacts";
DROP TABLE IF EXISTS "contacts";
