-- +goose Up
-- This table creates a many to many relationship between comments and users
-- enabling a robust @mention and notification system.
CREATE TABLE "comment_mentions" (
	"comment_id" BIGINT NOT NULL REFERENCES "comments"("id") ON DELETE CASCADE,
	"user_id" BIGINT NOT NULL REFERENCES "users"("id") ON DELETE CASCADE,
	PRIMARY KEY ("comment_id", "user_id")
);

-- +goose Down
DROP TABLE IF EXISTS "comment_mentions";

