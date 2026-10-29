-- ============================================================
--  FACILITY TAGS: one catalogue for every council, in categories
-- ============================================================

-- name: ListTagCategories :many
SELECT
    id,
    name,
    position
FROM
    tag_category
ORDER BY
    position,
    name;

-- name: ListTags :many
-- Every tag, and how many facilities carry it (the superadministrator sees what a delete takes away).
SELECT
    t.id,
    t.category_id,
    t.slug,
    t.name,
    (SELECT COUNT(*) FROM facility_tag ft WHERE ft.tag_id = t.id)::INT AS facility_count
FROM
    tag t
ORDER BY
    t.name;

-- name: NextTagCategoryPosition :one
SELECT COALESCE(MAX(position), 0)::INT + 1 FROM tag_category;

-- name: CreateTagCategory :one
INSERT INTO tag_category (name, position)
VALUES (sqlc.arg('name'), sqlc.arg('position'))
RETURNING id;

-- name: UpdateTagCategory :execrows
UPDATE tag_category
SET
    name     = sqlc.arg('name'),
    position = sqlc.arg('position')
WHERE
    id = sqlc.arg('id');

-- name: DeleteTagCategory :execrows
-- Its tags go with it, and so do they from every facility (ON DELETE CASCADE).
DELETE FROM tag_category WHERE id = sqlc.arg('id');

-- name: CreateTag :one
INSERT INTO tag (category_id, slug, name)
VALUES (sqlc.arg('category_id'), sqlc.arg('slug'), sqlc.arg('name'))
RETURNING id;

-- name: UpdateTag :execrows
-- The slug stays: it is in addresses people may have shared.
UPDATE tag
SET
    category_id = sqlc.arg('category_id'),
    name        = sqlc.arg('name')
WHERE
    id = sqlc.arg('id');

-- name: DeleteTag :execrows
DELETE FROM tag WHERE id = sqlc.arg('id');
