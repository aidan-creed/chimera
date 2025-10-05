-- +goose Up

-- The "ingestion_jobs" table tracks the status and metadata of each job
CREATE TABLE "ingestion_jobs" (
	"id" UUID PRIMARY KEY,
	"source_type" VARCHAR(50) NOT NULL,
	"source_details" JSONB,
	"item_type" TEXT NOT NULL,
	"status" VARCHAR(50) NOT NULL,
	"started_at" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	"completed_at" TIMESTAMPTZ,
	"error_details" TEXT,
	"user_id" BIGINT REFERENCES "users"("id"),
	"source_uri" TEXT,
	"total_rows" INTEGER,
	"processed_rows" INTEGER,
	"initial_error_count" INTEGER,
	"resolved_rows_count" INTEGER DEFAULT 0
);

COMMENT ON TABLE "ingestion_jobs" IS 'Tracks the metadata and status of a single data upload job.';
COMMENT ON COLUMN "ingestion_jobs"."source_type" IS 'The type of source from which data is being ingested, e.g., csv, api, etc.';
COMMENT ON COLUMN "ingestion_jobs"."item_type" IS 'The type of item being ingested, corresponds to the items.item_type column.';
COMMENT ON COLUMN "ingestion_jobs"."initial_error_count" IS 'A counter for how many errored rows have been successfully corrected by a user.';



-- The ingestion_errors table stores details about rows that fail validation
CREATE TABLE "ingestion_errors" (
	"id" UUID PRIMARY KEY,
	"job_id" UUID NOT NULL REFERENCES "ingestion_jobs"("id") ON DELETE CASCADE,
	"timestamp" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	"original_row_data" JSONB NOT NULL,
	"reason_for_failure" TEXT NOT NULL,
	"status" VARCHAR(50) NOT NULL DEFAULT 'new',
	"corrected_data" JSONB,
	"resolved_at" TIMESTAMPTZ,
	"resolved_by" BIGINT REFERENCES "users"("id")
);

COMMENT ON TABLE "ingestion_errors" IS 'Stores individual rows that failed validation during an ingestion job and tracks their triage status.';
COMMENT ON COLUMN "ingestion_errors"."status" IS 'The lifecycle of the error: new, pending_revalidation, resolved, ignored.';
COMMENT ON COLUMN "ingestion_errors"."corrected_data" IS 'The user-corrected version of the data, awaiting re-validation.';
COMMENT ON COLUMN "ingestion_errors"."resolved_by" IS 'The user who submitted the correction for this error.';

-- +goose Down
DROP TABLE IF EXISTS "ingestion_errors";
DROP TABLE IF EXISTS "ingestion_jobs";


