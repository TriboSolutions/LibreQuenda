-- Problems residents report at a facility, and the council's maintenance list.

-- name: GetFacilityForIssue :one
-- The facility a problem is reported at: it must be active, in an active council.
SELECT
    f.id,
    f.tenant_id,
    f.name
FROM
    facility f
    JOIN tenant t ON t.id = f.tenant_id
WHERE
    f.id = sqlc.arg('facility_id')
    AND f.is_active = TRUE
    AND t.is_active = TRUE;

-- name: GetBookingForIssue :one
-- A booking the person holds or joined, to report a problem from it.
SELECT
    b.id,
    b.facility_id,
    b.ends_at
FROM
    booking b
WHERE
    b.id = sqlc.arg('booking_id')
    AND b.status = 'confirmed'
    AND (
        b.user_id = sqlc.arg('user_id')
        OR EXISTS (SELECT 1 FROM booking_participant p WHERE p.booking_id = b.id AND p.user_id = sqlc.arg('user_id'))
    );

-- name: CountOpenIssuesByReporter :one
SELECT
    COUNT(*)
FROM
    facility_issue
WHERE
    reporter_id = sqlc.arg('reporter_id')
    AND status IN ('new', 'in_progress');

-- name: CreateFacilityIssue :one
-- Row-level security: runs as the facility's council (db.WithTenant).
INSERT INTO facility_issue (
    facility_id, reporter_id, booking_id, category, description, photo_ref, urgent, share_contact
) VALUES (
    sqlc.arg('facility_id'), sqlc.arg('reporter_id'), sqlc.narg('booking_id'), sqlc.arg('category'),
    sqlc.arg('description'), sqlc.narg('photo_ref'), sqlc.arg('urgent'), sqlc.arg('share_contact')
)
RETURNING
    id;

-- name: ListCouncilIssues :many
-- The council's maintenance list, newest first; facility_id and status narrow it.
-- The reporter's first name always; the API keeps their email only when they shared it.
SELECT
    i.id,
    i.facility_id,
    f.name AS facility_name,
    i.category,
    i.description,
    i.photo_ref,
    i.urgent,
    i.status,
    i.staff_note,
    i.created_at,
    i.resolved_at,
    b.starts_at AS booking_starts_at,
    split_part(COALESCE(u.full_name, ''), ' ', 1)::TEXT AS reporter_first_name,
    i.share_contact,
    u.email AS reporter_email -- the API drops it unless share_contact
FROM
    facility_issue i
    JOIN facility f ON f.id = i.facility_id
    LEFT JOIN app_user u ON u.id = i.reporter_id
    LEFT JOIN booking b ON b.id = i.booking_id
WHERE
    f.tenant_id = sqlc.arg('tenant_id')
    AND (sqlc.narg('facility_id')::UUID IS NULL OR i.facility_id = sqlc.narg('facility_id'))
    AND (sqlc.narg('status')::issue_status IS NULL OR i.status = sqlc.narg('status'))
ORDER BY
    i.created_at DESC
LIMIT 500;

-- name: CountNewCouncilIssues :one
-- The badge on the council administration's Maintenance tab.
SELECT
    COUNT(*)
FROM
    facility_issue i
    JOIN facility f ON f.id = i.facility_id
WHERE
    f.tenant_id = sqlc.arg('tenant_id')
    AND i.status = 'new';

-- name: GetCouncilIssue :one
SELECT
    i.id,
    i.status,
    i.reporter_id,
    f.name AS facility_name,
    f.slug AS facility_slug,
    t.name AS tenant_name,
    t.slug AS tenant_slug
FROM
    facility_issue i
    JOIN facility f ON f.id = i.facility_id
    JOIN tenant t ON t.id = f.tenant_id
WHERE
    i.id = sqlc.arg('id')
    AND f.tenant_id = sqlc.arg('tenant_id');

-- name: UpdateFacilityIssue :exec
-- Row-level security: runs as the council (db.WithTenant). resolved_* follow
-- the status: set on solved or rejected, cleared when it is opened again.
UPDATE facility_issue
SET
    status      = sqlc.arg('status'),
    staff_note  = sqlc.narg('staff_note'),
    resolved_by = CASE WHEN sqlc.arg('status')::issue_status IN ('solved', 'rejected') THEN sqlc.narg('staff_id')::UUID END,
    resolved_at = CASE WHEN sqlc.arg('status')::issue_status IN ('solved', 'rejected') THEN COALESCE(resolved_at, NOW()) END,
    updated_at  = NOW()
WHERE
    id = sqlc.arg('id');

-- name: CreateIssueNotification :exec
-- In the reporter's bell, and in the email queue (ListPendingIssueEmails).
INSERT INTO notification (user_id, issue_id, type, payload)
VALUES (sqlc.arg('user_id'), sqlc.arg('issue_id'), sqlc.arg('type'), sqlc.arg('payload'));

-- name: ListPendingIssueEmails :many
-- The email queue for issue notices (no booking): through the council's relay
-- or else the environment's.
SELECT
    n.id,
    n.type,
    n.payload,
    u.email,
    u.full_name,
    u.preferred_language,
    f.name AS facility_name,
    t.name AS tenant_name,
    t.id   AS tenant_id,
    s.config AS smtp_config
FROM
    notification n
    JOIN app_user u ON u.id = n.user_id
    JOIN facility_issue i ON i.id = n.issue_id
    JOIN facility f ON f.id = i.facility_id
    JOIN tenant   t ON t.id = f.tenant_id
    LEFT JOIN tenant_smtp s ON s.tenant_id = t.id
WHERE
    n.emailed_at IS NULL
    AND n.type IN ('issue_solved', 'issue_rejected')
    AND (sqlc.arg('env_relay')::BOOLEAN OR s.tenant_id IS NOT NULL)
ORDER BY
    n.created_at ASC
LIMIT sqlc.arg('max_rows');

-- name: ListIssuesByReporter :many
-- The person's own reports, for their data export.
SELECT
    i.id,
    f.name AS facility_name,
    i.category,
    i.description,
    i.urgent,
    i.share_contact,
    i.status,
    i.staff_note,
    i.created_at,
    i.resolved_at
FROM
    facility_issue i
    JOIN facility f ON f.id = i.facility_id
WHERE
    i.reporter_id = sqlc.arg('reporter_id')
ORDER BY
    i.created_at ASC;

-- name: ExportIssues :many
-- The council's maintenance list for its data export.
SELECT
    i.id,
    i.facility_id,
    i.reporter_id,
    i.booking_id,
    i.category,
    i.description,
    i.photo_ref,
    i.urgent,
    i.share_contact,
    i.status,
    i.staff_note,
    i.resolved_by,
    i.resolved_at,
    i.created_at,
    i.updated_at
FROM
    facility_issue i
    JOIN facility f ON f.id = i.facility_id
WHERE
    f.tenant_id = sqlc.arg('tenant_id')
ORDER BY
    i.created_at ASC,
    i.id ASC;
