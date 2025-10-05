-- name: CreateComment :one
INSERT INTO comments (
	item_id,
	comment,
	user_id
) VALUES (
	$1, $2, $3
)
RETURNING id, item_id, comment, user_id, created_at, updated_at;


-- name: AddMentionToComment :exec
INSERT INTO comment_mentions (
	comment_id,
	user_id
) VALUES (
	$1, $2
) ON CONFLICT DO NOTHING;

-- name: ListCommentsForItem :many
SELECT
	c.id,
	c.comment,
	c.created_at,
	u.display_name,
	-- Aggregate mentioned user IDs and names into JSON array
	(
		SELECT COALESCE(json_agg(json_build_object('user_id', mu.id, 'display_name', mu.display_name)), '[]')
		FROM comment_mentions cm
		JOIN users mu ON cm.user_id = mu.id
		WHERE cm.comment_id = c.id
	) AS mentioned_users
FROM
	comments c
JOIN
	users u ON c.user_id = u.id
WHERE
	c.item_id = $1
ORDER BY
	c.created_at ASC;


-- name: SetCommentEmbedding :exec
-- Sets the embedding for a specific comment after its been created
UPDATE comments
SET
	embedding = $2
WHERE
	id = $1;

