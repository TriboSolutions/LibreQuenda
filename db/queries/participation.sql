-- Participation: the council's announcements, its quick polls, and the public
-- side of the maintenance list (what is known to be broken, what was fixed).

-- ============================================================
--  Announcements (comunicados)
-- ============================================================

-- name: CreateCouncilNotice :one
-- Row-level security: runs as the council (db.WithTenant).
INSERT INTO council_notice (tenant_id, facility_id, title, body, expires_at, notify, created_by)
VALUES (
    sqlc.arg('tenant_id'), sqlc.narg('facility_id'), sqlc.arg('title'), sqlc.arg('body'),
    sqlc.arg('expires_at'), sqlc.arg('notify'), sqlc.narg('created_by')
)
RETURNING
    id, created_at;

-- name: NotifyNoticeRecipients :execrows
-- In the bell (and the email queue) of who books there: for a facility, who had
-- a booking there in the last 30 days or has one coming, the owner or who joined
-- with an account; for the whole council, the same at any of its facilities over
-- 90 days, and its verified residents.
INSERT INTO notification (user_id, notice_id, type, payload)
SELECT DISTINCT
    people.user_id,
    n.id,
    'council_notice'::notification_type,
    jsonb_build_object(
        'title', n.title,
        'tenant_name', t.name, 'tenant_slug', t.slug,
        'facility_name', COALESCE(f.name, ''), 'facility_slug', COALESCE(f.slug, '')
    )
FROM
    council_notice n
    JOIN tenant t ON t.id = n.tenant_id
    LEFT JOIN facility f ON f.id = n.facility_id
    JOIN LATERAL (
        SELECT b.user_id
        FROM booking b JOIN facility bf ON bf.id = b.facility_id
        WHERE b.status = 'confirmed'
          AND b.user_id IS NOT NULL
          AND bf.tenant_id = n.tenant_id
          AND (n.facility_id IS NULL OR b.facility_id = n.facility_id)
          AND b.ends_at > NOW() - CASE WHEN n.facility_id IS NULL THEN INTERVAL '90 days' ELSE INTERVAL '30 days' END
        UNION
        SELECT p.user_id
        FROM booking_participant p JOIN booking b ON b.id = p.booking_id JOIN facility bf ON bf.id = b.facility_id
        WHERE b.status = 'confirmed'
          AND p.user_id IS NOT NULL
          AND bf.tenant_id = n.tenant_id
          AND (n.facility_id IS NULL OR b.facility_id = n.facility_id)
          AND b.ends_at > NOW() - CASE WHEN n.facility_id IS NULL THEN INTERVAL '90 days' ELSE INTERVAL '30 days' END
        UNION
        SELECT r.user_id
        FROM tenant_resident r
        WHERE n.facility_id IS NULL AND r.tenant_id = n.tenant_id
    ) people ON TRUE
WHERE
    n.id = sqlc.arg('notice_id');

-- name: SetNoticeRecipients :exec
-- Row-level security: runs as the council (db.WithTenant).
UPDATE council_notice
SET
    recipients = sqlc.arg('recipients')
WHERE
    id = sqlc.arg('id');

-- name: ListCouncilNotices :many
-- The council's announcements for its administrators, the current ones first.
SELECT
    n.id,
    n.facility_id,
    f.name AS facility_name,
    n.title,
    n.body,
    n.expires_at,
    n.notify,
    n.recipients,
    n.created_at
FROM
    council_notice n
    LEFT JOIN facility f ON f.id = n.facility_id
WHERE
    n.tenant_id = sqlc.arg('tenant_id')
ORDER BY
    (n.expires_at > NOW()) DESC,
    n.created_at DESC
LIMIT 200;

-- name: GetCouncilNotice :one
SELECT
    n.id,
    n.title
FROM
    council_notice n
WHERE
    n.id = sqlc.arg('id')
    AND n.tenant_id = sqlc.arg('tenant_id');

-- name: DeleteCouncilNotice :execrows
-- Withdrawn: off the pages and out of the bells (emails already sent stay sent).
-- Row-level security: runs as the council (db.WithTenant).
DELETE FROM council_notice
WHERE
    id = sqlc.arg('id')
    AND tenant_id = sqlc.arg('tenant_id');

-- name: ListActiveFacilityNotices :many
-- On a facility's page: its own announcements and the council's, newest first.
SELECT
    n.id,
    n.facility_id,
    n.title,
    n.body,
    n.expires_at,
    n.created_at
FROM
    council_notice n
WHERE
    n.tenant_id = sqlc.arg('tenant_id')
    AND (n.facility_id IS NULL OR n.facility_id = sqlc.arg('facility_id'))
    AND n.expires_at > NOW()
ORDER BY
    n.created_at DESC
LIMIT 10;

-- name: ListActiveCouncilNotices :many
-- On the council's page: every current announcement, with the facility it is about.
SELECT
    n.id,
    n.facility_id,
    COALESCE(f.name, '')::TEXT AS facility_name,
    COALESCE(f.slug, '')::TEXT AS facility_slug,
    n.title,
    n.body,
    n.expires_at,
    n.created_at
FROM
    council_notice n
    LEFT JOIN facility f ON f.id = n.facility_id
WHERE
    n.tenant_id = sqlc.arg('tenant_id')
    AND n.expires_at > NOW()
    AND (f.id IS NULL OR f.is_active = TRUE)
ORDER BY
    n.created_at DESC
LIMIT 20;

-- name: ListPendingNoticeEmails :many
-- The email queue for announcements: through the council's relay or else the environment's.
SELECT
    n.id,
    c.title,
    c.body,
    c.expires_at,
    u.email,
    u.full_name,
    u.preferred_language,
    u.email_notifications,
    COALESCE(f.name, '')::TEXT AS facility_name,
    COALESCE(f.slug, '')::TEXT AS facility_slug,
    t.name AS tenant_name,
    t.slug AS tenant_slug,
    t.id   AS tenant_id,
    s.config AS smtp_config
FROM
    notification n
    JOIN app_user u ON u.id = n.user_id
    JOIN council_notice c ON c.id = n.notice_id
    JOIN tenant t ON t.id = c.tenant_id
    LEFT JOIN facility f ON f.id = c.facility_id
    LEFT JOIN tenant_smtp s ON s.tenant_id = t.id
WHERE
    n.emailed_at IS NULL
    AND n.type = 'council_notice'
    AND (sqlc.arg('env_relay')::BOOLEAN OR s.tenant_id IS NOT NULL)
ORDER BY
    n.created_at ASC
LIMIT sqlc.arg('max_rows');

-- name: ExportNotices :many
-- The council's announcements for its data export.
SELECT
    id,
    facility_id,
    title,
    body,
    expires_at,
    notify,
    recipients,
    created_by,
    created_at
FROM
    council_notice
WHERE
    tenant_id = sqlc.arg('tenant_id')
ORDER BY
    created_at ASC,
    id ASC;


-- ============================================================
--  Public maintenance list (incidencias públicas)
-- ============================================================

-- name: ListOpenFacilityIssues :many
-- On a facility's page, without what anyone wrote or who: the problems known
-- and not fixed yet, one line per category, oldest first.
SELECT
    i.category,
    COUNT(*)::INT AS reports,
    MIN(i.created_at)::TIMESTAMPTZ AS first_reported_at,
    BOOL_OR(i.status = 'in_progress')::BOOLEAN AS in_progress,
    BOOL_OR(i.urgent)::BOOLEAN AS urgent
FROM
    facility_issue i
WHERE
    i.facility_id = sqlc.arg('facility_id')
    AND i.status IN ('new', 'in_progress')
GROUP BY
    i.category
ORDER BY
    MIN(i.created_at) ASC;

-- name: ListRecentlySolvedFacilityIssues :many
-- …and what was fixed in the last 30 days, the latest first.
SELECT
    i.category,
    COUNT(*)::INT AS reports,
    MAX(i.resolved_at)::TIMESTAMPTZ AS solved_at
FROM
    facility_issue i
WHERE
    i.facility_id = sqlc.arg('facility_id')
    AND i.status = 'solved'
    AND i.resolved_at > NOW() - INTERVAL '30 days'
GROUP BY
    i.category
ORDER BY
    MAX(i.resolved_at) DESC;

-- name: CouncilIssueResolution :one
-- How long the council takes to fix what residents report: the average, in
-- days, over what it solved in the last 180 days.
SELECT
    COUNT(*)::INT AS solved,
    COALESCE(AVG(EXTRACT(EPOCH FROM (i.resolved_at - i.created_at)) / 86400), 0)::FLOAT8 AS average_days
FROM
    facility_issue i
    JOIN facility f ON f.id = i.facility_id
WHERE
    f.tenant_id = sqlc.arg('tenant_id')
    AND i.status = 'solved'
    AND i.resolved_at > NOW() - INTERVAL '180 days';


-- ============================================================
--  Polls (enquisas)
-- ============================================================

-- name: CreatePoll :one
-- Row-level security: runs as the council (db.WithTenant).
INSERT INTO poll (tenant_id, facility_id, question, residents_only, closes_at, created_by)
VALUES (
    sqlc.arg('tenant_id'), sqlc.narg('facility_id'), sqlc.arg('question'),
    sqlc.arg('residents_only'), sqlc.arg('closes_at'), sqlc.narg('created_by')
)
RETURNING
    id;

-- name: CreatePollOption :exec
-- Row-level security: runs as the council (db.WithTenant).
INSERT INTO poll_option (poll_id, position, label)
VALUES (sqlc.arg('poll_id'), sqlc.arg('position'), sqlc.arg('label'));

-- name: NotifyPollAudience :execrows
-- Asks the council's people, in the bell and the email queue: its verified
-- residents and administrators and, unless residents only, who booked there in
-- the last year (as ListPollsForUser shows it to).
INSERT INTO notification (user_id, poll_id, type, payload)
SELECT DISTINCT
    people.user_id,
    p.id,
    'poll_opened'::notification_type,
    jsonb_build_object(
        'title', p.question,
        'tenant_name', t.name, 'tenant_slug', t.slug,
        'facility_name', COALESCE(f.name, ''), 'facility_slug', COALESCE(f.slug, '')
    )
FROM
    poll p
    JOIN tenant t ON t.id = p.tenant_id
    LEFT JOIN facility f ON f.id = p.facility_id
    JOIN LATERAL (
        SELECT r.user_id FROM tenant_resident r WHERE r.tenant_id = p.tenant_id
        UNION
        SELECT m.user_id FROM tenant_membership m WHERE m.tenant_id = p.tenant_id
        UNION
        SELECT b.user_id
        FROM booking b JOIN facility bf ON bf.id = b.facility_id
        WHERE NOT p.residents_only AND bf.tenant_id = p.tenant_id AND b.status = 'confirmed'
          AND b.user_id IS NOT NULL AND b.ends_at > NOW() - INTERVAL '365 days'
        UNION
        SELECT bp.user_id
        FROM booking_participant bp JOIN booking b ON b.id = bp.booking_id JOIN facility bf ON bf.id = b.facility_id
        WHERE NOT p.residents_only AND bf.tenant_id = p.tenant_id AND b.status = 'confirmed'
          AND bp.user_id IS NOT NULL AND b.ends_at > NOW() - INTERVAL '365 days'
    ) people ON TRUE
WHERE
    p.id = sqlc.arg('poll_id');

-- name: NotifyPollResults :execrows
-- Once a poll closes (its day ended, or the council closed it early), who was
-- asked is told the results are out, once.
WITH closed AS (
    UPDATE poll
    SET
        results_sent_at = NOW()
    WHERE
        results_sent_at IS NULL
        AND (closed_at IS NOT NULL OR closes_at <= NOW())
    RETURNING
        id
)
INSERT INTO notification (user_id, poll_id, type, payload)
SELECT
    n.user_id,
    n.poll_id,
    'poll_closed'::notification_type,
    n.payload
FROM
    notification n
    JOIN closed c ON c.id = n.poll_id
WHERE
    n.type = 'poll_opened';

-- name: SetPollRecipients :exec
-- Row-level security: runs as the council (db.WithTenant).
UPDATE poll
SET
    recipients = sqlc.arg('recipients')
WHERE
    id = sqlc.arg('id');

-- name: ListPendingPollEmails :many
-- The email queue for polls: through the council's relay or else the
-- environment's; the job skips the invitations to vote of the ones closed meanwhile.
SELECT
    n.id,
    n.type,
    p.id AS poll_id,
    p.question,
    p.closes_at,
    (p.closed_at IS NULL AND p.closes_at > NOW())::BOOLEAN AS is_open,
    u.email,
    u.full_name,
    u.preferred_language,
    u.email_notifications,
    COALESCE(f.name, '')::TEXT AS facility_name,
    t.name AS tenant_name,
    t.slug AS tenant_slug,
    t.id   AS tenant_id,
    t.timezone,
    s.config AS smtp_config
FROM
    notification n
    JOIN app_user u ON u.id = n.user_id
    JOIN poll p ON p.id = n.poll_id
    JOIN tenant t ON t.id = p.tenant_id
    LEFT JOIN facility f ON f.id = p.facility_id
    LEFT JOIN tenant_smtp s ON s.tenant_id = t.id
WHERE
    n.emailed_at IS NULL
    AND n.type IN ('poll_opened', 'poll_closed')
    AND (sqlc.arg('env_relay')::BOOLEAN OR s.tenant_id IS NOT NULL)
ORDER BY
    n.created_at ASC
LIMIT sqlc.arg('max_rows');

-- name: ListCouncilPolls :many
-- The council's polls for its administrators, the open ones first.
SELECT
    p.id,
    p.recipients,
    p.facility_id,
    f.name AS facility_name,
    p.question,
    p.residents_only,
    p.closes_at,
    p.closed_at,
    p.created_at,
    (p.closed_at IS NULL AND p.closes_at > NOW())::BOOLEAN AS is_open,
    (SELECT COUNT(*) FROM poll_vote v WHERE v.poll_id = p.id)::INT AS votes
FROM
    poll p
    LEFT JOIN facility f ON f.id = p.facility_id
WHERE
    p.tenant_id = sqlc.arg('tenant_id')
ORDER BY
    (p.closed_at IS NULL AND p.closes_at > NOW()) DESC,
    p.created_at DESC
LIMIT 200;

-- name: ListPollOptions :many
-- The options of some polls, in their order, with how many chose each.
SELECT
    o.id,
    o.poll_id,
    o.position,
    o.label,
    (SELECT COUNT(*) FROM poll_vote v WHERE v.option_id = o.id)::INT AS votes
FROM
    poll_option o
WHERE
    o.poll_id = ANY(sqlc.arg('poll_ids')::UUID[])
ORDER BY
    o.poll_id,
    o.position;

-- name: GetCouncilPoll :one
SELECT
    p.id,
    p.question,
    (p.closed_at IS NULL AND p.closes_at > NOW())::BOOLEAN AS is_open
FROM
    poll p
WHERE
    p.id = sqlc.arg('id')
    AND p.tenant_id = sqlc.arg('tenant_id');

-- name: ClosePoll :execrows
-- Closed early: the results show from now on.
-- Row-level security: runs as the council (db.WithTenant).
UPDATE poll
SET
    closed_at = NOW()
WHERE
    id = sqlc.arg('id')
    AND tenant_id = sqlc.arg('tenant_id')
    AND closed_at IS NULL
    AND closes_at > NOW();

-- name: DeletePoll :execrows
-- Row-level security: runs as the council (db.WithTenant).
DELETE FROM poll
WHERE
    id = sqlc.arg('id')
    AND tenant_id = sqlc.arg('tenant_id');

-- name: ListPollsForUser :many
-- /enquisas: the polls of the councils the person is part of (a verified
-- resident, its administrator, or who booked there in the last year), while
-- open and for a year after they close, to follow the results; residents-only
-- ones only for who may vote. my_option: their vote, if any.
SELECT
    p.id,
    p.question,
    p.residents_only,
    p.closes_at,
    p.closed_at,
    (p.closed_at IS NULL AND p.closes_at > NOW())::BOOLEAN AS is_open,
    t.name AS tenant_name,
    t.slug AS tenant_slug,
    COALESCE(f.name, '')::TEXT AS facility_name,
    COALESCE(f.slug, '')::TEXT AS facility_slug,
    v.option_id AS my_option,
    (SELECT COUNT(*) FROM poll_vote c WHERE c.poll_id = p.id)::INT AS votes
FROM
    poll p
    JOIN tenant t ON t.id = p.tenant_id
    LEFT JOIN facility f ON f.id = p.facility_id
    LEFT JOIN poll_vote v ON v.poll_id = p.id AND v.user_id = sqlc.arg('user_id')
WHERE
    t.is_active = TRUE
    AND COALESCE(p.closed_at, p.closes_at) > NOW() - INTERVAL '365 days'
    AND (
        EXISTS (SELECT 1 FROM tenant_resident r WHERE r.tenant_id = p.tenant_id AND r.user_id = sqlc.arg('user_id'))
        OR EXISTS (SELECT 1 FROM tenant_membership m WHERE m.tenant_id = p.tenant_id AND m.user_id = sqlc.arg('user_id'))
        OR (NOT p.residents_only AND EXISTS (
            SELECT 1 FROM booking b JOIN facility bf ON bf.id = b.facility_id
            WHERE bf.tenant_id = p.tenant_id AND b.status = 'confirmed' AND b.ends_at > NOW() - INTERVAL '365 days'
              AND (b.user_id = sqlc.arg('user_id')
                   OR EXISTS (SELECT 1 FROM booking_participant bp WHERE bp.booking_id = b.id AND bp.user_id = sqlc.arg('user_id')))
        ))
    )
ORDER BY
    (p.closed_at IS NULL AND p.closes_at > NOW()) DESC,
    CASE WHEN p.closed_at IS NULL AND p.closes_at > NOW() THEN p.closes_at END ASC,
    COALESCE(p.closed_at, p.closes_at) DESC
LIMIT 100;

-- name: GetPollForVote :one
-- Whether the person may vote: the poll open, its council active, and, when
-- residents only, the person verified there (as IsTenantResident).
SELECT
    p.id,
    p.tenant_id,
    p.residents_only,
    (p.closed_at IS NULL AND p.closes_at > NOW() AND t.is_active)::BOOLEAN AS is_open,
    (
        EXISTS (SELECT 1 FROM tenant_resident r WHERE r.tenant_id = p.tenant_id AND r.user_id = sqlc.arg('user_id'))
        OR EXISTS (SELECT 1 FROM tenant_membership m WHERE m.tenant_id = p.tenant_id AND m.user_id = sqlc.arg('user_id') AND m.role = 'admin')
        OR EXISTS (SELECT 1 FROM platform_admin a WHERE a.user_id = sqlc.arg('user_id'))
    )::BOOLEAN AS is_resident,
    u.email_verified
FROM
    poll p
    JOIN tenant t ON t.id = p.tenant_id
    JOIN app_user u ON u.id = sqlc.arg('user_id')
WHERE
    p.id = sqlc.arg('poll_id');

-- name: CastPollVote :execrows
-- One vote per account, changed while the poll is open. Nothing when the
-- option is not one of the poll's.
INSERT INTO poll_vote (poll_id, option_id, user_id)
SELECT o.poll_id, o.id, sqlc.arg('user_id')
FROM poll_option o
WHERE o.id = sqlc.arg('option_id') AND o.poll_id = sqlc.arg('poll_id')
ON CONFLICT (poll_id, user_id) DO UPDATE
SET
    option_id  = EXCLUDED.option_id,
    updated_at = NOW();

-- name: ListVotesByUser :many
-- The person's own votes, for their data export.
SELECT
    p.question,
    o.label,
    t.name AS tenant_name,
    v.created_at,
    v.updated_at
FROM
    poll_vote v
    JOIN poll p ON p.id = v.poll_id
    JOIN poll_option o ON o.id = v.option_id
    JOIN tenant t ON t.id = p.tenant_id
WHERE
    v.user_id = sqlc.arg('user_id')
ORDER BY
    v.created_at ASC;

-- name: ExportPolls :many
-- The council's polls and their results, for its data export (no voters).
SELECT
    p.id,
    p.facility_id,
    p.question,
    p.residents_only,
    p.closes_at,
    p.closed_at,
    p.created_by,
    p.created_at,
    o.position,
    o.label,
    (SELECT COUNT(*) FROM poll_vote v WHERE v.option_id = o.id)::INT AS votes
FROM
    poll p
    JOIN poll_option o ON o.poll_id = p.id
WHERE
    p.tenant_id = sqlc.arg('tenant_id')
ORDER BY
    p.created_at ASC,
    p.id ASC,
    o.position ASC;
