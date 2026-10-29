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

CREATE TYPE checkin_method AS ENUM (
    'reader',       -- a QR reader at the door read the booking or the personal QR
    'printed_qr',   -- the resident's phone read the QR printed on the door
    'entry_code',   -- the resident typed the facility's entry code
    'staff'         -- council staff confirmed it from the calendar
);

CREATE TYPE notification_type AS ENUM (
    'booking_confirmed',
    'booking_changed',
    'booking_cancelled',            -- cancelled by the user
    'booking_cancelled_by_council', -- closure, maintenance…
    'booking_reminder',
    'council_admin_granted'         -- made administrator of a council (no booking)
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
    logo_ref      TEXT         NULL,              -- storage reference, never a URL
    is_active     BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);


-- A council's own SMTP relay for the emails LibreQuenda sends about it. Without
-- one, they go through the relay in the environment (SMTP_*); with neither, they
-- wait in the queue.
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
    image_ref                    TEXT         NULL,       -- storage reference, never a URL
    is_active                    BOOLEAN      NOT NULL DEFAULT TRUE,

    -- usage rules
    slot_minutes                 INT          NOT NULL DEFAULT 60 CHECK (slot_minutes > 0),
    max_advance_days             INT          NOT NULL DEFAULT 14 CHECK (max_advance_days > 0),
    min_cancel_hours             INT          NOT NULL DEFAULT 2  CHECK (min_cancel_hours >= 0),
    max_active_bookings_per_user INT          NOT NULL DEFAULT 2  CHECK (max_active_bookings_per_user > 0),
    min_age                      INT          NULL CHECK (min_age BETWEEN 1 AND 80),  -- age control: NULL = none
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
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- The personal QR on /perfil: a random token that lasts about a minute, so a
-- screenshot is soon useless. At a reader it checks the person in, or books the
-- turn about to start. The QR carries only the token, no personal data; a token
-- works once.
CREATE TABLE user_qr_token (
    token      UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID         NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ  NOT NULL,
    used_at    TIMESTAMPTZ  NULL
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
    CHECK (ends_at > starts_at),
    CHECK ((status = 'cancelled') = (cancelled_by IS NOT NULL)),
    CHECK (holder_name IS NULL OR (user_id IS NULL AND channel = 'staff')),
    CHECK (holder_phone IS NULL OR holder_name IS NOT NULL),
    CHECK ((checked_in_at IS NULL) = (checked_in_method IS NULL)),
    CHECK (checked_in_by IS NULL OR checked_in_method = 'staff'),

    -- no two confirmed bookings may overlap on the same facility
    EXCLUDE USING gist (
        facility_id WITH =,
        tstzrange(starts_at, ends_at) WITH &&
    ) WHERE (status = 'confirmed')
);


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
--  NOTIFICATIONS (avisos)
-- ============================================================

CREATE TABLE notification (
    id         UUID              PRIMARY KEY DEFAULT uuidv7(),
    user_id    UUID              NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,
    booking_id UUID              NULL REFERENCES booking(id) ON DELETE SET NULL,
    type       notification_type NOT NULL,
    payload    JSONB             NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ       NOT NULL DEFAULT NOW(),
    read_at    TIMESTAMPTZ       NULL,
    emailed_at TIMESTAMPTZ       NULL               -- NULL = pending in the email queue
);
