-- ============================================================
--  ENUMS
-- ============================================================

CREATE TYPE tenant_role AS ENUM (
    'admin'         -- manages the council's facilities, hours, rules and indicators
);

CREATE TYPE booking_status AS ENUM (
    'confirmed',
    'cancelled'
);

CREATE TYPE booking_canceller AS ENUM (
    'user',     -- the resident who booked
    'council'   -- a closure, or the facility deactivated
);

CREATE TYPE booking_channel AS ENUM (
    'online',   -- the resident, in the app (or at a reader with the personal QR)
    'staff'     -- council staff, for someone who phoned or came to the desk
);

-- who can find and join a booking
CREATE TYPE booking_kind AS ENUM (
    'private',  -- only by the owner's invite link (the default)
    'public',   -- a resident's booking anyone signed in finds in «For you» and joins while spots are left
    'event'     -- the council's own activity, with a title: created by its administrators, joined like a public one
);

CREATE TYPE checkin_method AS ENUM (
    'reader',       -- a QR reader at the door read the booking or the personal QR
    'printed_qr',   -- the resident's phone read the QR printed on the door
    'entry_code',   -- the resident typed the facility's entry code
    'nfc',          -- the resident's phone read the NFC tag on the door
    'staff'         -- council staff confirmed it from the calendar
);

CREATE TYPE notification_type AS ENUM (
    'booking_confirmed',
    'booking_changed',
    'booking_cancelled',            -- cancelled by the user
    'booking_cancelled_by_council', -- closure, maintenance…
    'booking_reminder',
    'booking_joined',               -- someone joined the person's booking (payload: name, guest)
    'council_admin_granted',        -- made administrator of a council (no booking)
    'issue_reported',               -- new issue for the council administrators
    'issue_solved',                 -- a problem the person reported was fixed (issue_id)
    'issue_rejected',               -- …or turned down, with the council's reason
    'rating_requested',             -- a booking ended: rate the facility (and, if it was public, the others who played)
    'resident_requested',           -- someone asks the council administrators to verify them as a resident
    'resident_verified',            -- the council verified the person as its resident
    'resident_rejected',            -- …or could not, from its own census
    'resident_revoked'              -- the council took the verification back (payload: cancelled, how many bookings it cancelled)
);

-- A problem a resident reports at a facility, in the council's maintenance list.
CREATE TYPE issue_category AS ENUM (
    'equipment',      -- nets, goals, baskets, lane ropes…
    'lighting',
    'cleaning',
    'changing_rooms',
    'accessibility',
    'safety',
    'other'
);

CREATE TYPE issue_status AS ENUM (
    'new',
    'in_progress',
    'solved',
    'rejected'        -- with the council's reason in staff_note
);


-- ============================================================
--  TENANTS (concellos)
-- ============================================================

CREATE TABLE tenant (
    id            UUID         PRIMARY KEY DEFAULT uuidv7(),
    slug          VARCHAR(64)  NOT NULL UNIQUE,   -- e.g. 'lalin', used in URLs
    name          VARCHAR(255) NOT NULL,
    timezone      VARCHAR(64)  NOT NULL DEFAULT 'Europe/Madrid',
    contact_email VARCHAR(255) NULL,
    contact_phone VARCHAR(50)  NULL,
    municipality_code CHAR(5)  NULL CHECK (municipality_code ~ '^[0-9]{5}$'), -- INE code: MeteoGalicia's forecast for its outdoor facilities
    logo_ref      TEXT         NULL,              -- storage reference, never a URL
    is_active     BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);


-- A council's own SMTP relay for the emails LibreQuenda sends about it. Without
-- one, they go through the relay in the environment (SMTP_*); with neither, they
-- wait in the queue.
-- The council's booking window, the default of its facilities: each one
-- without custom_rules keeps a copy (see facility), refreshed when this changes.
CREATE TABLE tenant_booking_rules (
    tenant_id            UUID        PRIMARY KEY REFERENCES tenant(id) ON DELETE CASCADE,
    max_advance_hours    INT         NOT NULL DEFAULT 336 CHECK (max_advance_hours > 0),
    min_advance_minutes  INT         NOT NULL DEFAULT 0  CHECK (min_advance_minutes >= 0),
    min_cancel_hours     INT         NOT NULL DEFAULT 2  CHECK (min_cancel_hours >= 0),
    max_active_bookings_per_user INT NOT NULL DEFAULT 2  CHECK (max_active_bookings_per_user > 0),
    max_bookings_per_day INT         NOT NULL DEFAULT 0  CHECK (max_bookings_per_day >= 0),
    max_turns_in_a_row   INT         NOT NULL DEFAULT 0  CHECK (max_turns_in_a_row >= 0),
    max_bookings_per_week INT        NOT NULL DEFAULT 0  CHECK (max_bookings_per_week >= 0),
    max_active_bookings_council INT  NOT NULL DEFAULT 0  CHECK (max_active_bookings_council >= 0), -- upcoming bookings per person across the council; 0 = no limit
    weekly_opening       BOOLEAN     NOT NULL DEFAULT FALSE,
    opens_weekday        SMALLINT    NOT NULL DEFAULT 7  CHECK (opens_weekday BETWEEN 1 AND 7),
    opens_minute         INT         NOT NULL DEFAULT 1200 CHECK (opens_minute BETWEEN 0 AND 1439),
    no_show_penalty      BOOLEAN     NOT NULL DEFAULT FALSE,
    no_show_limit        INT         NOT NULL DEFAULT 3  CHECK (no_show_limit > 0),
    no_show_window_days  INT         NOT NULL DEFAULT 30 CHECK (no_show_window_days > 0),
    no_show_block_days   INT         NOT NULL DEFAULT 7  CHECK (no_show_block_days > 0),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE tenant_smtp (
    tenant_id  UUID        PRIMARY KEY REFERENCES tenant(id) ON DELETE CASCADE,
    config     BYTEA       NOT NULL,   -- host, port, username, password and sender, sealed with SECRETS_KEY (api/secrets)
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);


-- ============================================================
--  USERS
--  Mirror of the identity provider (Zitadel). No credentials here.
-- ============================================================

CREATE TABLE app_user (
    id          UUID         PRIMARY KEY DEFAULT uuidv7(),
    idp_subject VARCHAR(255) NOT NULL UNIQUE,     -- Zitadel "sub" claim
    email       VARCHAR(255) NOT NULL,
    full_name   VARCHAR(255) NULL,
    email_verified BOOLEAN   NOT NULL DEFAULT FALSE, -- from Zitadel; roles are only granted to verified emails
    -- profile kept by LibreQuenda (name and email belong to Zitadel)
    birth_date         DATE         NULL CHECK (birth_date > DATE '1900-01-01'), -- required before booking
    phone              VARCHAR(30)  NULL,
    avatar_ref         TEXT         NULL,              -- storage reference, never a URL
    preferred_language VARCHAR(5)   NULL,              -- gl, es, en, ca
    email_reminders    BOOLEAN      NOT NULL DEFAULT TRUE,
    email_notifications BOOLEAN     NOT NULL DEFAULT TRUE,             -- off: booking and problem notices stay in the app only, no email
    show_recommendations BOOLEAN    NOT NULL DEFAULT TRUE,             -- "For you" on the home page, from their own bookings
    resident_requested_at TIMESTAMPTZ NULL,                            -- the last request to be verified as a resident: one a week
    calendar_token     UUID         NULL UNIQUE,       -- secret in the .ics feed address; NULL = feed off, a new one revokes the old address
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Superadministrators: create and manage councils. The first one comes from
-- SUPERADMIN_EMAIL (granted on that person's first login with a verified email).
CREATE TABLE platform_admin (
    user_id    UUID        PRIMARY KEY REFERENCES app_user(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- How this installation presents itself (name and logo in the header, page
-- titles and emails). One row at most, edited by a superadministrator at /admin;
-- without it, or with NULLs, the app is called LibreQuenda and shows its own logo.
CREATE TABLE app_settings (
    id         BOOLEAN     PRIMARY KEY DEFAULT TRUE CHECK (id),   -- a single row
    app_name   VARCHAR(60) NULL,
    logo_ref   TEXT        NULL,                                  -- storage reference, never a URL
    -- a person holds upcoming bookings in one council at a time (own or joined)
    one_council_per_account BOOLEAN NOT NULL DEFAULT TRUE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Roles per council. Residents have no row: any logged-in user can book.
CREATE TABLE tenant_membership (
    user_id    UUID        NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,
    tenant_id  UUID        NOT NULL REFERENCES tenant(id)   ON DELETE CASCADE,
    role       tenant_role NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, tenant_id, role)
);

-- People a council's administrators (or a superadministrator) verified as its
-- residents: they book and join at its residents_only facilities. A row means
-- verified; removing it takes the verification back, without touching the
-- bookings already made. Gone with the account or the council. A person is a
-- resident of one council only (UNIQUE user_id).
CREATE TABLE tenant_resident (
    tenant_id   UUID        NOT NULL REFERENCES tenant(id)   ON DELETE CASCADE,
    user_id     UUID        NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,
    verified_by UUID        NULL REFERENCES app_user(id) ON DELETE SET NULL,
    verified_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, user_id),
    UNIQUE (user_id)
);

-- Someone asking the council to verify them as its resident (for its
-- residents_only facilities): what the council needs to look them up in its
-- own census (padrón). Identity data is kept only while it is useful: the row
-- is deleted when an administrator verifies or discards the request, and the
-- purge_resident_requests job deletes any left after 15 days. One pending
-- request per person and council; nothing of it is copied anywhere else.
CREATE TABLE resident_request (
    id          UUID         PRIMARY KEY DEFAULT uuidv7(),
    tenant_id   UUID         NOT NULL REFERENCES tenant(id)   ON DELETE CASCADE,
    user_id     UUID         NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,
    dni         VARCHAR(9)   NOT NULL,   -- DNI or NIE, upper case, checked letter
    full_name   VARCHAR(200) NOT NULL,
    postal_code CHAR(5)      NOT NULL,
    street      VARCHAR(200) NOT NULL,   -- street and number where they live
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, user_id)
);

-- A role waiting for its person: accepted the first time someone logs in with
-- this email verified (they may not have an account yet when invited).
CREATE TABLE tenant_invitation (
    id          UUID         PRIMARY KEY DEFAULT uuidv7(),
    tenant_id   UUID         NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
    email       VARCHAR(255) NOT NULL,
    role        tenant_role  NOT NULL,
    invited_by  UUID         NULL REFERENCES app_user(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    accepted_at TIMESTAMPTZ  NULL,
    emailed_at  TIMESTAMPTZ  NULL               -- NULL = the invitation email is still to send
);


-- ============================================================
--  FACILITIES (instalacións)
--  The bookable unit: a court, a pool lane, a room…
--  Usage rules ("normas de uso") live on the facility.
-- ============================================================

CREATE TABLE facility (
    id                           UUID         PRIMARY KEY DEFAULT uuidv7(),
    tenant_id                    UUID         NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
    slug                         VARCHAR(64)  NOT NULL,   -- its page: /<council>/<slug>
    name                         VARCHAR(255) NOT NULL,
    description                  TEXT         NULL,
    address                      TEXT         NULL,
    latitude                     DOUBLE PRECISION NULL CHECK (latitude BETWEEN -90 AND 90),     -- map pin
    longitude                    DOUBLE PRECISION NULL CHECK (longitude BETWEEN -180 AND 180),
    capacity                     INT          NULL CHECK (capacity > 0),
    accessibility_info           TEXT         NULL,
    contact_phone                VARCHAR(50)  NULL,       -- its own; without one, residents see the council's
    contact_email                VARCHAR(255) NULL,
    show_weather                 BOOLEAN      NOT NULL DEFAULT FALSE, -- explicit facility preference
    image_ref                    TEXT         NULL,       -- storage reference, never a URL
    is_active                    BOOLEAN      NOT NULL DEFAULT TRUE,

    -- usage rules
    slot_minutes                 INT          NOT NULL DEFAULT 60 CHECK (slot_minutes > 0),
    -- booking window: the council's (tenant_booking_rules), copied here while
    -- custom_rules is off, or the facility's own. Staff booking for someone
    -- skip the per-person limits (per day, active, in a row, no-show block).
    custom_rules                 BOOLEAN      NOT NULL DEFAULT FALSE,
    max_advance_hours            INT          NOT NULL DEFAULT 336 CHECK (max_advance_hours > 0),   -- how far ahead
    min_advance_minutes          INT          NOT NULL DEFAULT 0  CHECK (min_advance_minutes >= 0), -- how close to the start, from the app
    min_cancel_hours             INT          NOT NULL DEFAULT 2  CHECK (min_cancel_hours >= 0),
    max_active_bookings_per_user INT          NOT NULL DEFAULT 2  CHECK (max_active_bookings_per_user > 0),
    max_bookings_per_day         INT          NOT NULL DEFAULT 0  CHECK (max_bookings_per_day >= 0),  -- per person and day here; 0 = no limit
    max_turns_in_a_row           INT          NOT NULL DEFAULT 0  CHECK (max_turns_in_a_row >= 0),    -- back-to-back turns per person; 0 = no limit
    max_bookings_per_week        INT          NOT NULL DEFAULT 0  CHECK (max_bookings_per_week >= 0),  -- per person and ISO week here; 0 = no limit
    -- weekly opening instead of the rolling max_advance_hours: next week's turns
    -- open all at once on opens_weekday (ISO, 7 = Sunday) at opens_minute (local)
    weekly_opening               BOOLEAN      NOT NULL DEFAULT FALSE,
    opens_weekday                SMALLINT     NOT NULL DEFAULT 7  CHECK (opens_weekday BETWEEN 1 AND 7),
    opens_minute                 INT          NOT NULL DEFAULT 1200 CHECK (opens_minute BETWEEN 0 AND 1439),
    no_show_penalty              BOOLEAN      NOT NULL DEFAULT FALSE,  -- block bookings after too many no-shows here
    no_show_limit                INT          NOT NULL DEFAULT 3  CHECK (no_show_limit > 0),          -- no-shows…
    no_show_window_days          INT          NOT NULL DEFAULT 30 CHECK (no_show_window_days > 0),    -- …within these days…
    no_show_block_days           INT          NOT NULL DEFAULT 7  CHECK (no_show_block_days > 0),     -- …block booking for these days
    min_age                      INT          NULL CHECK (min_age BETWEEN 1 AND 80),  -- age control: NULL = none
    residents_only               BOOLEAN      NOT NULL DEFAULT FALSE, -- only people the council verified as its residents (tenant_resident) book or join here
    usage_rules_text             TEXT         NULL,       -- free-text rules shown to residents

    created_at                   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at                   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, slug),
    CHECK ((latitude IS NULL) = (longitude IS NULL))
);

-- Weekly schedule ("horarios"). Several rows per weekday allowed (e.g. morning + afternoon).
-- Facility tags: one catalogue for every council, kept by the superadministrators
-- and grouped in categories ("Deportes", "Instalación"…), so residents can look
-- for a sport across all councils. A facility carries as many as fit it.
CREATE TABLE tag_category (
    id         UUID        PRIMARY KEY DEFAULT uuidv7(),
    name       VARCHAR(60) NOT NULL,
    position   INT         NOT NULL DEFAULT 0,   -- order on screen
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE tag (
    id          UUID        PRIMARY KEY DEFAULT uuidv7(),
    category_id UUID        NOT NULL REFERENCES tag_category(id) ON DELETE CASCADE,
    slug        VARCHAR(64) NOT NULL UNIQUE,     -- in addresses (?etiqueta=padel); kept when renamed
    name        VARCHAR(40) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE facility_tag (
    facility_id UUID NOT NULL REFERENCES facility(id) ON DELETE CASCADE,
    tag_id      UUID NOT NULL REFERENCES tag(id) ON DELETE CASCADE,
    PRIMARY KEY (facility_id, tag_id)
);

-- A facility's sports as one line for lists ("Baloncesto, Fútbol sala"): the names of
-- its tags in the first category (Deportes, unless the superadministrators reorder
-- them). Worked out on reading, so renaming a tag shows everywhere at once.
CREATE FUNCTION facility_sport(target UUID) RETURNS TEXT
LANGUAGE sql STABLE AS $$
    SELECT COALESCE(STRING_AGG(t.name, ', ' ORDER BY t.name), '')
    FROM facility_tag ft
    JOIN tag t ON t.id = ft.tag_id
    WHERE ft.facility_id = target
      AND t.category_id = (SELECT c.id FROM tag_category c ORDER BY c.position, c.name LIMIT 1)
$$;

CREATE TABLE opening_hours (
    id          UUID     PRIMARY KEY DEFAULT uuidv7(),
    facility_id UUID     NOT NULL REFERENCES facility(id) ON DELETE CASCADE,
    weekday     SMALLINT NOT NULL CHECK (weekday BETWEEN 1 AND 7), -- ISO: 1 = Monday
    opens_at    TIME     NOT NULL,
    closes_at   TIME     NOT NULL,
    CHECK (closes_at > opens_at)
);

-- Exceptions to the schedule: holidays, maintenance…
CREATE TABLE closure (
    id          UUID         PRIMARY KEY DEFAULT uuidv7(),
    facility_id UUID         NOT NULL REFERENCES facility(id) ON DELETE CASCADE,
    starts_at   TIMESTAMPTZ  NOT NULL,
    ends_at     TIMESTAMPTZ  NOT NULL,
    reason      VARCHAR(255) NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CHECK (ends_at > starts_at)
);


-- ============================================================
--  CHECK-IN READERS (lectores QR)
--  An unattended device at a facility entrance (an IoT reader or a tablet)
--  that scans booking QR codes. It signs in with a key shown once.
-- ============================================================

CREATE TABLE checkin_device (
    id           UUID         PRIMARY KEY DEFAULT uuidv7(),
    facility_id  UUID         NOT NULL REFERENCES facility(id) ON DELETE CASCADE,
    name         VARCHAR(80)  NOT NULL,              -- e.g. "Main door"
    key_hash     BYTEA        NOT NULL UNIQUE,       -- SHA-256 of the device key; the key itself is never stored
    key_hint     VARCHAR(8)   NOT NULL,              -- last characters of the key, to tell devices apart
    last_seen_at TIMESTAMPTZ  NULL,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Check-in by reading the facility, from the resident's own phone: a QR printed
-- on the door (its address carries qr_token), or the facility's fixed entry
-- code (derived from code_secret; a new secret gives a new code). Both need the
-- phone's location within radius_m of the facility's pin; the location is never
-- stored. flash_booking also covers the personal QR at the readers. staff_confirm
-- lets council staff mark the entry from the bookings calendar. qr_reader turns
-- the facility's QR readers off without removing them. No row: the
-- defaults, with every way off.
CREATE TABLE facility_checkin (
    facility_id   UUID         PRIMARY KEY REFERENCES facility(id) ON DELETE CASCADE,
    printed_qr    BOOLEAN      NOT NULL DEFAULT FALSE,
    qr_token      UUID         NOT NULL DEFAULT gen_random_uuid(),  -- a new one voids the printed QRs
    entry_code    BOOLEAN      NOT NULL DEFAULT FALSE,
    code_secret   BYTEA        NOT NULL DEFAULT (uuid_send(gen_random_uuid()) || uuid_send(gen_random_uuid())),  -- a new one voids the printed code
    radius_m      INT          NOT NULL DEFAULT 100 CHECK (radius_m BETWEEN 25 AND 1000),
    flash_booking BOOLEAN      NOT NULL DEFAULT TRUE,
    staff_confirm BOOLEAN      NOT NULL DEFAULT FALSE,              -- staff mark the entry from the calendar
    qr_reader     BOOLEAN      NOT NULL DEFAULT TRUE,               -- off: the readers refuse every QR, but stay set up (broken, switched off)
    nfc           BOOLEAN      NOT NULL DEFAULT FALSE,              -- an NFC tag on the door, holding the entrance address with nfc_token
    nfc_token     UUID         NOT NULL DEFAULT gen_random_uuid(),  -- a new one voids the tags written
    staff_contacts JSONB       NOT NULL DEFAULT '[]'::JSONB,        -- who residents ask on site: [{name, phone, place}], not accounts
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- When each facility could take check-ins (a QR reader switched on, the QR on
-- the door, the entry code or staff): one row per stretch, the current one open
-- (ends_at NULL). Kept by triggers on checkin_device and facility_checkin
-- (03A_triggers.sql), never written by the API. A booking that began outside
-- them says nothing about no-shows: nobody could check in.
CREATE TABLE facility_checkin_period (
    id          UUID        PRIMARY KEY DEFAULT uuidv7(),
    facility_id UUID        NOT NULL REFERENCES facility(id) ON DELETE CASCADE,
    starts_at   TIMESTAMPTZ NOT NULL,
    ends_at     TIMESTAMPTZ NULL,
    CHECK (ends_at IS NULL OR ends_at >= starts_at)
);

-- Whether the facility can take check-ins now, in any way.
CREATE FUNCTION facility_can_check_in(target UUID) RETURNS BOOLEAN
LANGUAGE sql STABLE AS $$
    SELECT (EXISTS (SELECT 1 FROM checkin_device d WHERE d.facility_id = target)
            AND COALESCE((SELECT c.qr_reader FROM facility_checkin c WHERE c.facility_id = target), TRUE))
        OR COALESCE((SELECT c.printed_qr OR c.entry_code OR c.staff_confirm OR c.nfc FROM facility_checkin c WHERE c.facility_id = target), FALSE)
$$;

-- Whether the facility could take check-ins at `at` (a booking's start).
CREATE FUNCTION facility_took_checkin(target UUID, at TIMESTAMPTZ) RETURNS BOOLEAN
LANGUAGE sql STABLE AS $$
    SELECT EXISTS (
        SELECT 1 FROM facility_checkin_period p
        WHERE p.facility_id = target AND p.starts_at <= at AND (p.ends_at IS NULL OR p.ends_at > at)
    )
$$;

-- The personal QR on /perfil: a random token that lasts about a minute, so a
-- screenshot is soon useless. At a reader it checks the person in, or books the
-- turn about to start. The QR carries only the token, no personal data; a token
-- works once.
CREATE TABLE user_qr_token (
    token      UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID         NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ  NOT NULL,
    used_at    TIMESTAMPTZ  NULL,
    -- the flash booking a reader made with it, so the phone showing it can open it
    booking_id UUID         NULL
);


-- ============================================================
--  BOOKINGS (reservas)
-- ============================================================

CREATE TABLE booking (
    id           UUID           PRIMARY KEY DEFAULT uuidv7(),
    facility_id  UUID           NOT NULL REFERENCES facility(id) ON DELETE CASCADE,
    user_id      UUID           NULL REFERENCES app_user(id) ON DELETE SET NULL, -- NULL after account erasure (kept for indicators)
    starts_at    TIMESTAMPTZ    NOT NULL,
    ends_at      TIMESTAMPTZ    NOT NULL,
    status       booking_status NOT NULL DEFAULT 'confirmed',
    rules_accepted_at TIMESTAMPTZ NULL,                 -- when the resident ticked "I have read the rules" (or staff, that they told them)
    -- booked by council staff: for an account (user_id), or for someone without one
    -- (holder_*, forgotten 30 days after the booking ends)
    channel      booking_channel NOT NULL DEFAULT 'online',
    booked_by    UUID           NULL REFERENCES app_user(id) ON DELETE SET NULL,
    holder_name  VARCHAR(80)    NULL,
    holder_phone VARCHAR(30)    NULL,
    cancelled_at TIMESTAMPTZ    NULL,
    cancelled_by booking_canceller NULL,               -- who cancelled it, for the indicators
    created_at   TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    -- check-in: the booking QR carries only this random token, no personal data
    checkin_token        UUID        NOT NULL DEFAULT gen_random_uuid(),
    checked_in_at        TIMESTAMPTZ NULL,
    checked_in_device_id UUID        NULL REFERENCES checkin_device(id) ON DELETE SET NULL,
    checked_in_method    checkin_method NULL,               -- how the person got in, for the indicators
    checked_in_by        UUID        NULL REFERENCES app_user(id) ON DELETE SET NULL, -- the staff member who confirmed it
    -- invite link: others fill the facility's capacity; a new token revokes the old link
    invite_token         UUID        NULL,
    -- public and event: anyone signed in can find it in «For you» and join it while spots are left; private only by the invite link
    kind                 booking_kind NOT NULL DEFAULT 'private',
    public_description   TEXT        NOT NULL DEFAULT '' CHECK (char_length(public_description) <= 1000),
    -- event only: its name («Torneo de fútbol 5v5») and, if set, how many can join instead of the facility's capacity.
    -- An event belongs to the council: no owner, booked by staff, any start and end (not the turns).
    title                VARCHAR(120) NULL CHECK (char_length(btrim(title)) > 0),
    event_capacity       INT         NULL CHECK (event_capacity > 0),
    CHECK (ends_at > starts_at),
    CHECK ((status = 'cancelled') = (cancelled_by IS NOT NULL)),
    CHECK (holder_name IS NULL OR (user_id IS NULL AND channel = 'staff')),
    CHECK (holder_phone IS NULL OR holder_name IS NOT NULL),
    CHECK ((checked_in_at IS NULL) = (checked_in_method IS NULL)),
    CHECK (checked_in_by IS NULL OR checked_in_method = 'staff'),
    CHECK ((kind = 'event') = (title IS NOT NULL)),
    CHECK (event_capacity IS NULL OR kind = 'event'),
    CHECK (kind <> 'event' OR (user_id IS NULL AND holder_name IS NULL AND channel = 'staff')),

    -- no two confirmed bookings may overlap on the same facility
    EXCLUDE USING gist (
        facility_id WITH =,
        tstzrange(starts_at, ends_at) WITH &&
    ) WHERE (status = 'confirmed')
);

-- declared after booking exists: a personal QR keeps the flash booking it made
ALTER TABLE user_qr_token ADD CONSTRAINT fk_user_qr_token_booking FOREIGN KEY (booking_id) REFERENCES booking(id) ON DELETE SET NULL;


-- Who joins a booking through its invite link: a user with an account, or a
-- guest who gives only a name and an age. The owner is not a row here.
CREATE TABLE booking_participant (
    id         UUID         PRIMARY KEY DEFAULT uuidv7(),
    booking_id UUID         NOT NULL REFERENCES booking(id)  ON DELETE CASCADE,
    user_id    UUID         NULL REFERENCES app_user(id)     ON DELETE CASCADE,
    guest_name VARCHAR(80)  NULL,
    guest_age  SMALLINT     NULL CHECK (guest_age BETWEEN 1 AND 120),
    joined_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CHECK (
        (user_id IS NOT NULL AND guest_name IS NULL AND guest_age IS NULL)
        OR (user_id IS NULL AND guest_name IS NOT NULL AND guest_age IS NOT NULL)
    )
);


-- ============================================================
--  RATINGS (valoracións)
-- ============================================================

-- 1 to 5 stars, given after a confirmed booking ended by someone who was in it
-- (the owner, or a participant with an account), one per booking and person;
-- it can be changed while the booking can still be rated (30 days). Nobody sees
-- who gave which stars: only the average and how many. An erased account's
-- ratings of facilities stay, anonymous.
CREATE TABLE facility_rating (
    id          UUID         PRIMARY KEY DEFAULT uuidv7(),
    facility_id UUID         NOT NULL REFERENCES facility(id) ON DELETE CASCADE,
    booking_id  UUID         NOT NULL REFERENCES booking(id)  ON DELETE CASCADE,
    user_id     UUID         NULL REFERENCES app_user(id)     ON DELETE SET NULL,
    stars       SMALLINT     NOT NULL CHECK (stars BETWEEN 1 AND 5),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (booking_id, user_id)
);

-- People rate each other only after a public booking (strangers who met
-- through «For you»), never themselves; guests without an account are not rated.
-- Gone with either account.
CREATE TABLE user_rating (
    id          UUID         PRIMARY KEY DEFAULT uuidv7(),
    booking_id  UUID         NOT NULL REFERENCES booking(id)  ON DELETE CASCADE,
    rater_id    UUID         NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,
    rated_id    UUID         NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,
    stars       SMALLINT     NOT NULL CHECK (stars BETWEEN 1 AND 5),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (booking_id, rater_id, rated_id),
    CHECK (rater_id <> rated_id)
);


-- ============================================================
--  NOTIFICATIONS (avisos)
-- ============================================================

-- Problems residents report at a facility (broken net, no light…): the
-- council's maintenance list. The reporter goes on account erasure, the report
-- stays. share_contact: the reporter lets the council's staff see their email.
CREATE TABLE facility_issue (
    id            UUID           PRIMARY KEY DEFAULT uuidv7(),
    facility_id   UUID           NOT NULL REFERENCES facility(id) ON DELETE CASCADE,
    reporter_id   UUID           NULL REFERENCES app_user(id) ON DELETE SET NULL,
    booking_id    UUID           NULL REFERENCES booking(id) ON DELETE SET NULL,   -- the booking it was reported from
    category      issue_category NOT NULL,
    description   VARCHAR(1000)  NOT NULL,
    photo_ref     TEXT           NULL,       -- storage reference (issues/…), never a URL
    urgent        BOOLEAN        NOT NULL DEFAULT FALSE,
    share_contact BOOLEAN        NOT NULL DEFAULT FALSE,
    status        issue_status   NOT NULL DEFAULT 'new',
    staff_note    VARCHAR(1000)  NULL,       -- for the reporter: what was done, or why not
    resolved_by   UUID           NULL REFERENCES app_user(id) ON DELETE SET NULL,
    resolved_at   TIMESTAMPTZ    NULL,       -- solved or rejected
    created_at    TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE TABLE notification (
    id         UUID              PRIMARY KEY DEFAULT uuidv7(),
    user_id    UUID              NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,
    booking_id UUID              NULL REFERENCES booking(id) ON DELETE SET NULL,
    issue_id   UUID              NULL REFERENCES facility_issue(id) ON DELETE SET NULL, -- issue_solved / issue_rejected
    type       notification_type NOT NULL,
    payload    JSONB             NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ       NOT NULL DEFAULT NOW(),
    read_at    TIMESTAMPTZ       NULL,
    emailed_at TIMESTAMPTZ       NULL               -- NULL = pending in the email queue
);


-- ============================================================
--  AUDIT LOG (rexistro de actividade)
-- ============================================================

-- What administrators did, for accountability (GDPR Art. 5(2) and 32, ENS):
-- who, when, which council, what action on what. Append-only for the API
-- (05_row_security.sql revokes UPDATE and DELETE), purged after the retention
-- period by purge_audit_log(). No personal data beyond the actor's and target
-- person's account ids, which become NULL when those accounts are erased;
-- names are read from app_user when shown, never copied here.
CREATE TABLE audit_log (
    id             UUID         PRIMARY KEY DEFAULT uuidv7(),
    at             TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    actor_id       UUID         NULL REFERENCES app_user(id) ON DELETE SET NULL, -- NULL: erased account, or the system
    actor_role     VARCHAR(20)  NOT NULL,             -- superadmin, council_admin, resident, system
    tenant_id      UUID         NULL,                 -- the council it concerns; NULL: platform-wide. No FK: kept after the council is deleted
    tenant_label   VARCHAR(255) NULL,                 -- the council's name then
    action         VARCHAR(64)  NOT NULL,             -- a stable code, e.g. facility.updated
    target_type    VARCHAR(32)  NULL,                 -- facility, closure, booking, device, tenant, user, tag, …
    target_id      UUID         NULL,                 -- the target's id when it is not a person
    target_user_id UUID         NULL REFERENCES app_user(id) ON DELETE SET NULL, -- when the target is a person
    target_label   VARCHAR(255) NULL,                 -- a non-personal name then (a facility's, a tag's)
    summary        TEXT         NOT NULL DEFAULT ''   -- what changed, without personal data or free text
);
