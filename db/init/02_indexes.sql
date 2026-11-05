-- ============================================================
--  INDEXES
-- ============================================================

CREATE INDEX idx_tenant_membership_tenant ON tenant_membership (tenant_id);
CREATE INDEX idx_tenant_resident_user ON tenant_resident (user_id);
CREATE INDEX idx_resident_request_user ON resident_request (user_id);
CREATE INDEX idx_resident_request_created ON resident_request (created_at);

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
CREATE INDEX idx_user_qr_token_user       ON user_qr_token (user_id);

CREATE INDEX idx_notification_user        ON notification (user_id, created_at DESC);
CREATE INDEX idx_notification_booking     ON notification (booking_id, type);
CREATE INDEX idx_notification_email_queue ON notification (created_at) WHERE emailed_at IS NULL;
CREATE INDEX idx_notification_unread      ON notification (user_id) WHERE read_at IS NULL; -- the bell polls the unread count

-- one pending invitation per council, email and role
CREATE UNIQUE INDEX uq_tenant_invitation_pending ON tenant_invitation (tenant_id, lower(email), role) WHERE accepted_at IS NULL;
CREATE INDEX idx_tenant_invitation_email          ON tenant_invitation (lower(email)) WHERE accepted_at IS NULL;
CREATE INDEX idx_app_user_email                   ON app_user (lower(email));

-- a tag or category name once, whatever its case
CREATE UNIQUE INDEX idx_tag_category_name ON tag_category (lower(name));
CREATE UNIQUE INDEX idx_tag_name          ON tag (lower(name));
CREATE INDEX idx_tag_category             ON tag (category_id);
CREATE INDEX idx_facility_tag_tag         ON facility_tag (tag_id);

-- check-in history: one open stretch per facility, and the lookups by start
CREATE UNIQUE INDEX idx_checkin_period_open ON facility_checkin_period (facility_id) WHERE ends_at IS NULL;
CREATE INDEX idx_checkin_period_facility ON facility_checkin_period (facility_id, starts_at);

-- the activity log: by council and time, newest first, and by action
CREATE INDEX idx_audit_log_tenant_at ON audit_log (tenant_id, at DESC);
CREATE INDEX idx_audit_log_at        ON audit_log (at DESC);
CREATE INDEX idx_audit_log_action    ON audit_log (action);

-- «For you»: the upcoming public bookings and events
CREATE INDEX idx_booking_public ON booking (starts_at) WHERE kind <> 'private' AND status = 'confirmed';

-- reported problems: the council's list by facility and status, and the open ones per person
CREATE INDEX idx_facility_issue_facility  ON facility_issue (facility_id, status, created_at DESC);
CREATE INDEX idx_facility_issue_reporter  ON facility_issue (reporter_id, status);

-- ratings: the averages on the facility page and next to people
CREATE INDEX idx_facility_rating_facility ON facility_rating (facility_id);
CREATE INDEX idx_user_rating_rated        ON user_rating (rated_id);
CREATE INDEX idx_user_rating_rater        ON user_rating (rater_id);
