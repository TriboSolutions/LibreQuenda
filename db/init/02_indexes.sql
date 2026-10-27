-- ============================================================
--  INDEXES
-- ============================================================

CREATE INDEX idx_tenant_membership_tenant ON tenant_membership (tenant_id);

CREATE INDEX idx_facility_tenant          ON facility (tenant_id) WHERE is_active = TRUE;

CREATE INDEX idx_opening_hours_facility   ON opening_hours (facility_id, weekday);

CREATE INDEX idx_closure_facility         ON closure (facility_id, starts_at);

CREATE INDEX idx_booking_user             ON booking (user_id, starts_at DESC);
CREATE INDEX idx_booking_facility_start   ON booking (facility_id, starts_at);
CREATE UNIQUE INDEX idx_booking_checkin_token ON booking (checkin_token);
CREATE UNIQUE INDEX idx_booking_invite_token  ON booking (invite_token) WHERE invite_token IS NOT NULL;
CREATE INDEX idx_booking_participant_booking  ON booking_participant (booking_id);
CREATE UNIQUE INDEX uq_booking_participant_user ON booking_participant (booking_id, user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_booking_participant_user     ON booking_participant (user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_checkin_device_facility  ON checkin_device (facility_id);

CREATE INDEX idx_notification_user        ON notification (user_id, created_at DESC);
CREATE INDEX idx_notification_booking     ON notification (booking_id, type);
CREATE INDEX idx_notification_email_queue ON notification (created_at) WHERE emailed_at IS NULL;
CREATE INDEX idx_notification_unread      ON notification (user_id) WHERE read_at IS NULL; -- the bell polls the unread count

-- one pending invitation per council, email and role
CREATE UNIQUE INDEX uq_tenant_invitation_pending ON tenant_invitation (tenant_id, lower(email), role) WHERE accepted_at IS NULL;
CREATE INDEX idx_tenant_invitation_email          ON tenant_invitation (lower(email)) WHERE accepted_at IS NULL;
CREATE INDEX idx_app_user_email                   ON app_user (lower(email));
