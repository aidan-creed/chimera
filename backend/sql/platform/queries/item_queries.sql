-- name: CreateItem :one
-- Inserts a new item record into database
-- Go is responsible for constructing the custom_properties JSONB
INSERT INTO items (
	item_type, 
	scope,
	business_key,
	status,
	custom_properties,
	embedding
) VALUES (
	$1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: CreateItemEvent :one
-- Inserts a new event record for a specific time
INSERT INTO items_events (
	item_id,
	event_type,
	event_data,
	created_by
) VALUES (
	$1, $2, $3, $4
)
RETURNING *;

-- name: DeactivateItemsBySource :exec
UPDATE items SET status = 'inactive'
WHERE item_type= $1 AND custom_properties->>'reporting_source' = $2;

-- name: UpsertItems :execrows
--Insert new records from staging, or update existing ones based on business key
INSERT INTO items (
	item_type, scope, business_key, status, custom_properties, embedding
)
SELECT
	item_type,
	scope, 
	business_key,
	'active',
	custom_properties,
	embedding
FROM temp_items_staging
ON CONFLICT (item_type, business_key) DO UPDATE SET 
	status = EXCLUDED.status,
	scope = EXCLUDED.scope,
	custom_properties = items.custom_properties || EXCLUDED.custom_properties,
	embedding = EXCLUDED.embedding,
	updated_at = NOW();


-- name: GetEventsForItem :many
-- Fetch the event history for a specific item, newest first
SELECT * FROM "items_events"
WHERE item_id = $1
ORDER BY created_at DESC;

-- name: GetItemForUpdate :one
-- Fetch a single item for update
SELECT * FROM "items"
WHERE id = $1 LIMIT 1;

-- name: UpdateItem :one
-- Updates the mutable fields of a specific item
UPDATE items
SET
	scope = $2,
	status = $3,
	custom_properties = $4,
	updated_at = NOW()
WHERE
	id = $1
RETURNING *;


