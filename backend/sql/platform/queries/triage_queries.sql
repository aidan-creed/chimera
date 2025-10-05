-- name: CreateIngestionError :one
-- Inserts a new ingestion error record for a row that failed processing.
INSERT INTO ingestion_errors (
    id,
    job_id,
    original_row_data,
    reason_for_failure
) VALUES (
    $1, $2, $3, $4
)
RETURNING *;

-- name: ItemExistsByBusinessKey :one
-- Checks for the existence of an item by its type and business key. Returns 1 if it exists, 0 otherwise.
SELECT EXISTS(SELECT 1 FROM items WHERE item_type = $1 AND business_key = $2)::int;

-- name: CreateIngestionJob :one
-- Inserts a new file ingestion job record.
INSERT INTO ingestion_jobs (
	id, 
	source_type,
	source_details,
	item_type,
	status, 
	user_id,
	source_uri
) VALUES (
	$1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: CreateTempItemsStagingTable :exec
-- Creates a temporary table for staging items during ingest
-- This table is dropped on commit
CREATE TEMP TABLE temp_items_staging (LIKE items INCLUDING DEFAULTS) ON COMMIT DROP;

-- name: UpdateIngestionJobStatus :exec
-- Updates the status and details of an ingestion job
UPDATE ingestion_jobs
SET
	status = $2,
	completed_at = NOW(),
	error_details = $3,
	processed_rows = $4,
	initial_error_count = $5
WHERE
	id = $1;

-- name: IncrementIngestionJobResolvedRows :exec
UPDATE ingestion_jobs
SET
	resolved_rows_count = resolved_rows_count + 1
WHERE
	id = (SELECT job_id FROM ingestion_errors WHERE ingestion_errors.id = $1);

-- name: ListIngestionJobs :many
-- Lists ingestion jobs with pagination support
SELECT 
	id,
	source_type,
	source_details,
	item_type,
	status,
	user_id,
	source_uri,
	started_at,
	completed_at,
	error_details,
	processed_rows,
	initial_error_count,
	resolved_rows_count,
	total_rows
FROM 
	ingestion_jobs
ORDER BY 
	started_at DESC
LIMIT $1 OFFSET $2;

-- name: GetIngestionErrorsByJobID :many
-- Retrieves ingestion errors associated with a specific job ID, with pagination support
SELECT
	id,
	job_id,
	"timestamp",
	original_row_data,
	reason_for_failure,
	status,
	corrected_data,
	resolved_at,
	resolved_by
FROM
	ingestion_errors
WHERE
	job_id = $1 AND status IN ('new', 'pending_revalidation')
ORDER BY
	"timestamp" ASC;

-- name: UpdateIngestionErrorWithCorrection :one
UPDATE ingestion_errors
SET
    corrected_data = $2,
    status = 'pending_revalidation',
    resolved_by = $3,
    resolved_at = NOW()
WHERE
    id = $1
RETURNING *;

