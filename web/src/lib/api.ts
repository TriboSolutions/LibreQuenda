// Typed client for the LibreQuenda API (served by Caddy under /api)
import i18n from '@/i18n'

export type SlotStatus = 'free' | 'booked' | 'closed' | 'past' | 'too_far' | 'too_soon' // too_soon: closer to the start than the app books

export type Slot = {
  starts_at: string
  ends_at: string
  status: SlotStatus
}

/** private: only by the owner's invite link; public: a resident's, in «For you»; event: the council's, with a title and no owner. */
export type BookingKind = 'private' | 'public' | 'event'

/** A public booking or one of the council's events on the facility's day. */
export type PublicTurn = {
  id: string
  kind: 'public' | 'event'
  title: string // events only
  starts_at: string
  ends_at: string
  description: string
  capacity: number // with the owner for a public booking; only who joins for an event
  people: string[] // first names: the owner first (not for an event), then who joined; signed out, only the owner
  people_count: number // everyone in it, owner included: count with this, not people
  owner: PublicPerson | null // who booked it and answers for it; null for an event
  joined: PublicPerson[] // who joined, in order; empty signed out
}

/** Someone in a public booking: first name and the stars others gave them (null until rated, and for guests). */
export type PublicPerson = { name: string; rating: RatingSummary | null }

export type Facility = {
  public_bookings?: PublicTurn[]
  show_weather: boolean
  id: string
  slug: string
  name: string
  description: string
  image_url: string
  latitude: number | null
  longitude: number | null
  sport: string
  tags: string[] // tag slugs (see api.tags), in the catalogue's order
  address: string
  contact_phone: string // its own, "" when not given: show the council's instead (FacilityContact)
  contact_email: string
  slot_minutes: number
  // the booking window in force (the council's or the facility's own)
  max_advance_hours: number
  min_advance_minutes: number
  min_cancel_hours: number
  max_active_bookings_per_user: number
  max_bookings_per_day: number // per person and day; 0 = no limit
  max_turns_in_a_row: number // back-to-back turns per person; 0 = no limit
  max_bookings_per_week: number // per person and week; 0 = no limit
  council_max_active_bookings: number // per person across the council's facilities; 0 = no limit
  weekly_opening: { weekday: number; at: string } | null // instead of max_advance_hours: next week opens then (ISO weekday, "20:00")
  capacity: number // people per booking, owner included; 0 without a limit
  min_age: number | null // age control: null = none
  residents_only: boolean // only the council's verified residents book or join (Me.resident_of)
  usage_rules: string
  windows: { opens_at: string; closes_at: string }[]
  closures: { starts_at: string; ends_at: string; reason: string }[]
  slots: Slot[]
}

// contact: where residents send accessibility feedback and complaints (RD 1112/2018); "" when not given
export type Tenant = { slug: string; name: string; logo_url: string; contact_email: string; contact_phone: string }

/** The council's page: the council and its current announcements. */
export type CouncilPage = Tenant & { notices: PublicNotice[] }

// --- participation: announcements, the public maintenance list, polls -----------

/** One of the council's announcements, on its pages until it expires. */
export type PublicNotice = {
  id: string
  title: string
  body: string
  facility_name: string // "" for the whole council; only on the council's page
  facility_slug: string
  expires_at: string
  created_at: string
}

/** The public side of a facility's maintenance list: no one's words, no one's name. */
export type IssuesSummary = {
  open: { category: IssueCategory; reports: number; first_reported_at: string; in_progress: boolean; urgent: boolean }[]
  recently_solved: { category: IssueCategory; reports: number; solved_at: string }[] // the last 30 days
  average_days_to_solve: number | null // the council's average over 180 days; null before it solved any
  solved_count: number
}

export type AdminNotice = {
  id: string
  facility_id: string // "" for the whole council
  facility_name: string
  title: string
  body: string
  expires_at: string
  active: boolean
  notify: boolean
  recipients: number // told in the bell and by email
  created_at: string
}

export type NoticeInput = { facility_id: string; title: string; body: string; expires_on: string; notify: boolean }

export type PollOption = { id: string; label: string; votes: number | null } // votes: null for residents while open

export type AdminPoll = {
  id: string
  facility_id: string
  facility_name: string
  question: string
  residents_only: boolean
  closes_at: string
  closed_at: string | null
  open: boolean
  votes: number
  recipients: number // asked in the bell and by email when it opened
  options: PollOption[]
  created_at: string
}

export type PollInput = { facility_id: string; question: string; options: string[]; residents_only: boolean; closes_on: string }

/** A poll as a resident sees it in «For you». */
export type MyPoll = {
  id: string
  question: string
  tenant_name: string
  tenant_slug: string
  facility_name: string
  facility_slug: string
  residents_only: boolean
  open: boolean
  closes_at: string // or when it closed, if early
  my_option: string // "" before voting
  votes: number | null // null while open
  options: PollOption[]
}

export type GeoPoint = { latitude: number; longitude: number }

/** Where a page sits in a list the API pages (?page=&pageSize=, 100 by default). */
export type Pagination = { page: number; page_size: number; total: number; has_more: boolean }

/** One facility on the home page's list and map, from any council. */
export type FacilityCard = {
  id: string
  slug: string
  name: string
  sport: string
  tags: string[]
  image_url: string
  address: string
  latitude: number | null // null without a map pin
  longitude: number | null
  free_today: number // turns still free today
  council: Tenant
  residents_only?: boolean // only the council's verified residents book or join there (also its events)
  joinable?: JoinableKind[] // upcoming, with spots left: residents' public bookings, the council's events
  distance_km?: number | null // with a position: how far, by the API (which also sorts by it)
  rating?: RatingSummary | null // residents' stars after their bookings; null until someone rated
}

/** What can be joined at a facility: a resident's public booking, or the council's event. */
export type JoinableKind = 'public' | 'event'

/** A facility tag from the platform's catalogue; facilities and addresses use its slug. */
export type Tag = { id: string; slug: string; name: string; facility_count: number }
export type TagCategory = { id: string; name: string; position: number; tags: Tag[] }
export type TagCatalogue = { categories: TagCategory[] }

/** An uploaded image: forms save the ref (storage reference); screens only show the url. */
export type StoredImage = { ref: string; url: string }
export type MediaFolder = 'branding' | 'councils' | 'facilities' | 'issues' | 'users'
/** Whose file an upload is: a council's logo or facility image (its slug), a problem's photo (the facility). */
export type MediaOwner = { council: string } | { facility: string }

/** How this installation presents itself, set by a superadministrator: name and logo. */
export type Branding = {
  name: string // what to show: the custom name, or LibreQuenda
  custom_name: string // "" = LibreQuenda
  logo_ref: string // "" = the LibreQuenda logo
  logo_url: string
  support_email: string // the installation's support contact (SUPPORT_EMAIL, SUPPORT_PHONE); "" = not set
  support_phone: string
}
export type BrandingInput = { name: string; logo_ref: string } // "" = back to LibreQuenda's

export type Day = {
  tenant: Tenant
  timezone: string
  date: string // YYYY-MM-DD in the council's time zone
  now: string
  facilities: Facility[]
}

export type WeeklyWindow = { weekday: number; opens_at: string; closes_at: string } // ISO weekday: 1 = Monday

/** MeteoGalicia's forecast of one period of an outdoor facility's day. */
export type WeatherPeriod = {
  period: 'morning' | 'afternoon' | 'night'
  sky: string // clear, partly_cloudy, cloudy, overcast, fog, haze, drizzle, rain, showers, storm, snow, sleet, hail; '' unknown
  rain_chance: number | null // %
  wind: string // calm, variable, light, moderate, strong, very_strong; '' unknown
  wind_from: string // n, ne, e, se, s, sw, w, nw; '' none
  temperature: number | null // ºC
  warnings: WeatherWarning[]
}
export type WeatherWarning = 'rain' | 'storm' | 'snow' | 'wind' | 'heat' | 'cold' | 'alert'
export type DayForecast = {
  date: string
  sky: string
  temp_min: number | null
  temp_max: number | null
  periods: WeatherPeriod[]
  warnings: WeatherWarning[]
}
/** null when there is none: indoors, no municipality, outside the forecast's days, or the source unreachable. */
export type Weather = { forecast: DayForecast | null; source: string }

export type FacilityPage = {
  tenant: Tenant
  timezone: string
  date: string
  now: string
  facility: Facility
  schedule: WeeklyWindow[]
  rating: RatingSummary | null // residents' stars after their bookings; null until someone rated
  notices: PublicNotice[] // the council's current announcements for it and for the whole council
  issues: IssuesSummary
}

// --- ratings ------------------------------------------------------------------

/** 1 to 5 stars: the average (one decimal) and how many gave them, never who. */
/** The analytics assistant's language model, as a superadministrator sees it (never the key). */
export type AISettings = { enabled: boolean; provider: 'openai' | 'anthropic'; base_url: string; model: string; has_key: boolean; updated_at: string | null }
/** What the form saves; an empty api_key keeps the saved one. */
export type AISettingsInput = { enabled: boolean; provider: 'openai' | 'anthropic'; base_url: string; model: string; api_key: string }
/** One query the assistant ran, and what it returned (at most 500 rows). */
export type AnalyticsStep = { sql: string; result?: { columns: string[]; rows: unknown[][]; truncated: boolean }; error?: string }
/** A chart the data assistant drew: its data is what its SQL returned, never numbers the model wrote. */
export type AnalyticsChart = {
  kind: 'bar' | 'stacked_bar' | 'horizontal_bar' | 'line' | 'pie' | 'heatmap'
  title: string
  x: 'x'
  y?: 'y' // heatmap: the rows of the grid
  series: { key: string; label: string }[]
  rows: Record<string, string | number | null>[]
  sql: string
}

export type AnalyticsAnswer = { answer: string; steps: AnalyticsStep[]; charts?: AnalyticsChart[] }

export type RatingSummary = { average: number; count: number }

/** Someone else in a public booking, with the stars the signed-in person gave them (0 = not yet). */
export type PersonToRate = { user_id: string; name: string; is_owner: boolean; stars: number }

/** The rating of one booking: the facility's stars and, after a public booking, the others'. */
export type BookingRating = {
  booking_id: string
  facility_name: string
  facility_slug: string
  tenant_slug: string
  tenant_name: string
  starts_at: string
  ends_at: string
  is_public: boolean
  can_rate: boolean // ended, not a no-show, within 30 days, not rated yet
  rated: boolean // rated already: the stars are final
  rate_until: string
  facility_stars: number // 0 = not yet
  people: PersonToRate[] // public bookings only
}

/** Saved once, for good: the facility's stars (required) and the others' (0: left unrated). */
export type RatingInput = { facility_stars: number; people: { user_id: string; stars: number }[] }

// --- council administration -----------------------------------------------

export type AdminFacilitySummary = {
  id: string
  slug: string
  name: string
  sport: string
  tags: string[]
  image_url: string
  is_active: boolean
  opening_window_count: number
  upcoming_booking_count: number
}

export type FacilityInput = {
  show_weather: boolean
  slug: string
  name: string
  description: string
  tags: string[] // tag slugs from the platform's catalogue (api.tags); its sport line comes from them
  address: string
  latitude: number | null
  longitude: number | null
  capacity: number | null
  accessibility_info: string
  contact_phone: string // optional: without them, residents see the council's
  contact_email: string
  image_ref: string // a storage reference, never a URL
  is_active: boolean
  slot_minutes: number
  min_age: number | null // age control: null = none, else 1 to 80
  residents_only: boolean // only the council's verified residents (its Veciños tab) book or join
  usage_rules: string
}

/** Booking rules across councils: a person holds upcoming bookings in one council at a time. */
export type BookingPolicy = { one_council_per_account: boolean }

/** The booking window: the council's, which its facilities inherit, or a facility's own. */
export type BookingRules = {
  max_advance_hours: number // how far ahead, from now
  min_advance_minutes: number // how close to the start the app still books
  min_cancel_hours: number // cancel or change until this long before
  max_active_bookings_per_user: number
  max_bookings_per_day: number // per person and day; 0 = no limit
  max_turns_in_a_row: number // back-to-back turns per person; 0 = no limit
  max_bookings_per_week: number // per person and ISO week; 0 = no limit
  max_active_bookings_council?: number // council only: a person's own upcoming bookings across its facilities; 0 = no limit
  // instead of max_advance_hours: next week's turns open all at once on opens_weekday (ISO, 7 = Sunday) at opens_minute
  weekly_opening: boolean
  opens_weekday: number
  opens_minute: number // minutes after local midnight
  // after no_show_limit no-shows in no_show_window_days, no booking for no_show_block_days
  no_show_penalty: boolean
  no_show_limit: number
  no_show_window_days: number
  no_show_block_days: number
}

export type AdminFacility = FacilityInput & {
  // the booking window, saved from its Reserva tab: the council's (custom_rules false) or its own
  custom_rules: boolean
  rules: BookingRules // the window in force
  council_rules: BookingRules // the council's, to show what it inherits
  id: string
  image_url: string
  booking_count: number
  bookings_to_cancel: number // upcoming bookings that deactivating could cancel
  schedule: WeeklyWindow[]
  closures: Closure[] // current and upcoming
  timezone: string // the council's: closure dates are typed in it
}

// --- staff booking (the "Reservas" tab of the council administration) ---------

/** A booking as council staff see it, with how to reach the person. */
export type StaffBooking = {
  id: string
  starts_at: string
  ends_at: string
  booked_by_staff: boolean
  is_event: boolean // the council's event: name is its title, nobody checks in
  has_account: boolean
  name: string // "" once a deleted account or a forgotten name
  email: string
  phone: string
  checked_in_at: string | null
  checked_in_method: CheckinMethod | ''
  checked_in_by: string // staff entry: who marked it
  checkin_state: StaffCheckinState // staff entry on and not in yet; '' otherwise
  undo_until: string | null // staff entry: until when it can be taken back
  participant_count: number
  can_cancel: boolean // until it starts
  checkin_code: string // booked by staff only: the entry QR to print for the person
}

/** How the person got in. */
export type CheckinMethod = 'reader' | 'printed_qr' | 'nfc' | 'entry_code' | 'staff'
/** Whether staff can mark the entry now: from 15 minutes before the start until the turn ends. */
export type StaffCheckinState = 'open' | 'too_early' | 'ended' | ''

export type StaffSlot = { starts_at: string; ends_at: string; status: SlotStatus; booking_id: string }

/** One facility's turns on one day, and who holds each booking. */
export type StaffDay = {
  date: string
  timezone: string
  facility_id: string
  min_age: number | null
  slot_minutes: number
  slots: StaffSlot[]
  bookings: StaffBooking[]
  staff_checkin: boolean // staff can mark the entry (the facility's Check-in tab)
  checkin_ways: CheckinMethod[] // how the person gets in; the booking's QR is only any use with 'reader'
  residents_only: boolean // an account must be a verified resident; a name, staff check
}

/** The council's own activity, joined like a public booking: any start and end, not the turns. */
export type CouncilEvent = {
  id: string
  facility_id: string
  facility_name: string
  starts_at: string
  ends_at: string
  title: string
  description: string
  own_capacity: number | null // null: the facility's
  capacity: number // who can join
  participant_count: number
  can_change: boolean // until it starts
  timezone: string // the council's: the form shows and takes the times in it
}

/** Someone in one of the council's events, as its staff see them. */
export type EventPerson = {
  id: string
  has_account: boolean
  name: string // the account's name and surnames, or the guest's name
  email: string // '' for a guest
  age: number | null // a guest's
  rating: RatingSummary | null
  is_resident: boolean // this council verified them as its resident
  joined_at: string
}

export type EventInput = {
  facility_id: string
  starts_at: string // local date-time in the council's time zone: "2026-11-15T10:00"
  ends_at: string
  title: string
  description: string
  capacity: number | null // null: the facility's
}

/** What stops an event at that time: confirmed bookings (cancel them from manual booking first) and closures. */
export type EventInTheWay = {
  bookings: { id: string; starts_at: string; ends_at: string; holder: string; is_event: boolean }[]
  closures: { starts_at: string; ends_at: string; reason: string }[]
}

/** For an account (email) or for someone without one (name, phone): exactly one of the two. */
export type StaffBookingInput = {
  facility_id: string
  starts_at: string
  email: string
  name: string
  phone: string
  accept_rules: boolean // staff told the person the usage rules
}

/** A closure (works, maintenance, an event): its slots show as closed. */
export type Closure = { id: string; starts_at: string; ends_at: string; reason: string }

/** Dates as typed in the council's time zone: "2026-11-15T00:00". */
export type ClosureInput = { starts_at: string; ends_at: string; reason: string }

export type GeocodeResult = { label: string; latitude: number; longitude: number }

/** A QR reader at a facility entrance. Its key is only returned on creation. */
export type CheckinDevice = { id: string; name: string; key_hint: string; last_seen_at: string | null; created_at: string }
export type CreatedCheckinDevice = CheckinDevice & { key: string }

/** The personal QR on /perfil: a new code every `refresh_seconds`; each one works once, until `expires_at`. */
export type PersonalQr = { code: string; expires_at: string; refresh_seconds: number }

/** The .ics feed of the user's bookings: url is empty while it is off. */
export type CalendarFeed = { url: string }

// --- entrance reader (signs in with its device key) -------------------------

export type Reader = {
  name: string
  facility_name: string
  facility_slug: string
  tenant_name: string
  tenant_slug: string
  timezone: string
  now: string
}

export type CheckinResultCode =
  | 'checked_in'
  | 'already_checked_in'
  | 'not_a_booking'
  | 'unknown_booking'
  | 'other_facility'
  | 'cancelled'
  | 'too_early'
  | 'ended'
  | 'reader_off' // the council turned this facility's readers off
  // the personal QR from /perfil
  | 'booked' // the turn about to start was booked, and the entry recorded
  | 'not_booked' // no booking here to let in, and none could be made: see `reason`
  | 'code_expired' // expired or already used

export type CheckinResult = {
  granted: boolean
  result: CheckinResultCode
  reason?: string // not_booked only, e.g. 'no_turn_soon', 'slot_booked', 'booking_limit_reached'
  booking: {
    facility_name: string
    first_name: string // only when granted
    starts_at: string
    ends_at: string
    checked_in_at: string | null
  } | null
}

/** How residents check in from their own phone at the facility (Entrada tab): both ways need its pin. */
export type CheckinSettings = {
  printed_qr: boolean
  qr_token: string // in the printed QR's address; a new one voids the prints
  nfc: boolean // an NFC tag on the door, holding the entrance address with nfc_token
  nfc_token: string // a new one voids the tags written
  entry_code: boolean
  code: string // the entry code to print; a new one voids the prints
  radius_m: number // how far from the pin the phone may be
  flash_booking: boolean // a free turn can be booked on the spot, also with the personal QR at the readers
  staff_confirm: boolean // staff mark the entry from the bookings calendar
  qr_reader: boolean // off: the readers refuse every QR, but stay set up
  staff_contacts: StaffContact[] // who residents ask on site (staff confirmation); not accounts
  has_location: boolean
}
export type CheckinSettingsInput = Pick<
  CheckinSettings,
  'printed_qr' | 'nfc' | 'entry_code' | 'radius_m' | 'flash_booking' | 'staff_confirm' | 'qr_reader' | 'staff_contacts'
>
/** Someone residents can ask on site to check in or book a free turn: only what the council shows. */
/** Without any, the facility's own contact (else the council's) stands in, with an email. */
export type StaffContact = { name: string; phone: string; place: string; email?: string }

/** /<council>/<facility>/entrada: checking in from the resident's own phone. */
export type Entrance = {
  facility_name: string
  facility_slug: string
  tenant_name: string
  tenant_slug: string
  timezone: string
  printed_qr: boolean
  nfc: boolean
  entry_code: boolean
  radius_m: number
  flash_booking: boolean
  booking: { starts_at: string; ends_at: string; checked_in_at: string | null } | null // theirs, that can go in now
}
/** The proof of being at the door, and where the phone is (checked, never stored). */
export type EntranceInput = {
  qr_token?: string
  nfc_token?: string // read from the NFC tag on the door (the tag opens /entrada?n=)
  code?: string
  latitude: number
  longitude: number
  accuracy: number
  book?: boolean // without a booking: book a free turn and check in
}
type Position = Pick<EntranceInput, 'latitude' | 'longitude' | 'accuracy'>
/** Flash booking: the facilities the phone is at (by entry code or with staff to ask), nearest first; result is a location problem when it cannot tell. */
export type Nearby = {
  result: '' | 'location_required' | 'location_inaccurate'
  facilities: {
    facility_name: string
    facility_slug: string
    tenant_name: string
    tenant_slug: string
    entry_code: boolean // books with its entry code
    printed_qr: boolean // has a QR on the door
    radius_m: number // how close the phone must be
    staff_contacts: StaffContact[] // staff to ask on site
  }[]
}
export type EntranceResult = Omit<CheckinResult, 'result'> & {
  result:
    | CheckinResultCode
    | 'method_off'
    | 'invalid_qr' // an old print
    | 'wrong_code'
    | 'no_facility_location'
    | 'location_required'
    | 'location_inaccurate'
    | 'too_far'
    | 'no_booking' // see can_book
  can_book: boolean
}

export type Booking = {
  id: string
  starts_at: string
  ends_at: string
  status: 'confirmed' | 'cancelled'
  facility_id: string
  facility_name: string
  facility_slug: string
  tenant_slug: string
  tenant_name: string
  can_cancel: boolean
  can_change: boolean // owner only: another time of the same facility, on the same terms as cancelling
  cancel_until: string
  checkin_code: string // QR content; "" unless the booking is upcoming and confirmed
  public_description?: string
  is_public?: boolean // owner only: anyone signed in finds it in «For you» and can join it
  kind: BookingKind // event: the council's, joined by the person
  title: string // events only
  checked_in_at: string | null
  checkin_ways?: CheckinMethod[] | null // owner only: the ways to check in the facility takes, for the check-in button
  staff_contacts?: StaffContact[] // owner only, with staff confirmation on: who to ask on site
  checkin_radius_m?: number // owner only: how close the phone must be for the code or the door QR
  // past confirmed bookings only, from the QR readers: unknown = no reader when it began, so nobody could check in
  attendance: '' | 'attended' | 'no_show' | 'unknown'
  can_rate: boolean // ended, not a no-show, within 30 days, not rated yet: the rating dialog
  // invites: the owner shares a link and others fill the facility's capacity
  role: 'owner' | 'participant' // participant: joined someone else's booking through an invite
  owner_name: string // participant only
  capacity: number // people including the owner (an event has none: who can join); 0 when the facility takes no invites
  participant_count: number // people who joined, not counting the owner
  can_leave: boolean
  people: Person[] // who is coming: the owner first; empty while nobody has joined
  facility: FacilityDetails // empty in the book and change responses
}

/** Where to go and what to know before going: in each booking and on the invite page. */
export type FacilityDetails = {
  image_url: string
  sport: string
  address: string
  latitude: number | null
  longitude: number | null
  accessibility_info: string
  slot_minutes: number
  min_cancel_hours: number
  min_age: number | null // age control: null = none
  residents_only: boolean // only the council's verified residents join: no guests
  usage_rules: string
}

// --- notifications (polled) ---------------------------------------------------

export type NotificationType =
  | 'booking_confirmed'
  | 'booking_changed'
  | 'booking_joined'
  | 'booking_cancelled'
  | 'booking_cancelled_by_council'
  | 'booking_reminder'
  | 'council_admin_granted'
  | 'issue_reported'
  | 'issue_solved' // a problem the person reported was fixed (reason: the council's note)
  | 'issue_rejected'
  | 'rating_requested' // a booking ended: rate the facility and, if it was public, the others
  | 'resident_requested' // to the council's administrators: someone asks to be verified
  | 'resident_verified' // the council verified the person as its resident
  | 'resident_rejected' // …or could not, from its census
  | 'participant_removed' // the owner of a public booking took the person out (they cannot join it again)
  | 'resident_revoked' // the council took the verification back (and cancelled the bookings at its residents-only facilities)
  | 'council_notice' // the council's announcement (title), about a facility (facility_slug) or the whole council
  | 'poll_opened' // the council asks (title: the question): vote in poll_id
  | 'poll_closed' // …and it closed: its results are out

export type AppNotification = {
  id: string
  booking_id: string // "" for notices about no booking
  poll_id: string // poll_opened, poll_closed
  is_public: boolean // rating_requested: the others can be rated too
  type: NotificationType
  facility_name: string
  facility_slug: string // issue_solved, issue_rejected
  starts_at: string | null
  ends_at: string | null
  name: string // first name of the person who joined
  reason: string // council cancellations
  closed_from: string | null // council cancellations caused by a closure
  closed_until: string | null
  by_staff: boolean // a council cancellation of this one booking, by staff
  cause: '' | 'owner' | 'account_deleted' | 'resident_revoked' | 'resident_revoked_left' // why it was cancelled, beyond the type
  title: string // the event's name, when the booking was one
  tenant_name: string // council_admin_granted
  tenant_slug: string
  created_at: string
  read: boolean
}

export type Notifications = { unread_count: number; notifications: AppNotification[] } // most recent first

/** What players see of each other: a name, never an email or an age. */
export type Person = { name: string; is_owner: boolean; is_guest: boolean; is_me: boolean }

// --- invites ------------------------------------------------------------------

export type InviteState = 'open' | 'full' | 'closed'

export type Participant = { id: string; name: string; is_guest: boolean; age: number | null; joined_at: string }

/** The owner's view of a booking's invite. */
export type Invite = { token: string; capacity: number; spots_left: number; state: InviteState; participants: Participant[] }

/** What anyone with the link sees, signed in or not. */
export type PublicInvite = {
  facility_name: string
  facility_slug: string
  tenant_slug: string
  tenant_name: string
  timezone: string
  owner_name: string
  starts_at: string
  ends_at: string
  capacity: number
  spots_left: number
  state: InviteState
  facility: FacilityDetails // image, address, rules and minimum age, before joining
}

/** The home page's "For you": from the signed-in person's own bookings and today's free turns. */
export type Recommendation = {
  kind: 'usual_today' | 'favourite' | 'similar' | 'public_booking'
  facility: FacilityCard
  starts_at: string // usual_today: that free turn; favourite and similar: the next free turn today
  times?: number // usual_today: bookings that weekday and time; favourite: bookings there
  weekday?: number // usual_today: ISO, 1 = Monday
  because?: string // similar: the facility of theirs it is like
  shared_tags?: string[] // similar: the tags (slugs) they share
  // public_booking: someone's public booking to join; at most the owner's first name
  booking?: {
    id: string
    kind?: 'public' | 'event' // event: the council's, with a title and no owner
    title?: string
    starts_at: string
    ends_at: string
    spots_left: number
    owner_first_name: string
    owner_rating?: RatingSummary | null // what others gave the owner after public bookings
    description?: string
  }
}

/** A problem residents report at a facility, for the council's maintenance list. */
export type IssueCategory = 'equipment' | 'lighting' | 'cleaning' | 'changing_rooms' | 'accessibility' | 'safety' | 'other'
export type IssueStatus = 'new' | 'in_progress' | 'solved' | 'rejected'
export type IssueReport = {
  facility_id: string
  booking_id: string // the active booking it is reported from
  category: IssueCategory
  description: string
  photo_ref?: string // a storage reference under issues/
  urgent: boolean
  share_contact: boolean // the council's staff may see the reporter's email
  // where the phone is: the reporter must be at the facility (checked, never stored)
  latitude?: number
  longitude?: number
  accuracy?: number
}
/** A report in the council's maintenance list. */
export type CouncilIssue = {
  facility_image_url: string
  id: string
  facility_id: string
  facility_name: string
  category: IssueCategory
  description: string
  photo_url: string
  urgent: boolean
  status: IssueStatus
  staff_note: string
  created_at: string
  resolved_at: string | null
  booking_starts_at: string | null
  reporter_name: string // first name only; "" after the account was deleted
  reporter_email: string // only when the reporter shared it
  reporter_phone: string // optional profile phone, only when contact sharing was allowed
}

export type MyBookings = { upcoming: Booking[]; history: Booking[] }

export type Me = {
  id: string
  email: string
  email_verified: boolean
  full_name: string
  is_superadmin: boolean
  roles: { tenant_id: string; tenant_slug: string; tenant_name: string; role: 'admin' }[]
  analytics_enabled?: boolean // the data assistant is on and they can use it
  // profile kept by LibreQuenda; name and email belong to Zitadel
  birth_date: string // YYYY-MM-DD, "" until filled in
  phone: string
  avatar_ref: string
  avatar_url: string
  preferred_language: string
  email_notifications: boolean
  email_reminders: boolean
  show_recommendations: boolean // "For you" on the home page
  profile_complete: boolean // the date of birth is required before booking
  rating?: RatingSummary | null // what others gave them after public bookings; null until someone did
  // council roles waiting for this person: granted once the email is verified
  pending_invitations?: { tenant_name: string; tenant_slug: string }[]
  // councils that verified this person as their resident: they book at their residents-only facilities
  // where they book as residents; by_role: as its administrator or a superadministrator, not verified by it
  resident_of?: { tenant_name: string; tenant_slug: string; verified_at: string; by_role: boolean }[]
  // councils the person asked to verify them, still undecided
  resident_requests?: { tenant_name: string; tenant_slug: string; requested_at: string }[]
}

/** Someone a council verified as its resident (its Veciños tab). */
export type Resident = { user_id: string; email: string; full_name: string; verified_at: string; verified_by: string }

export type ResidentsPage = { residents: Resident[]; total: number; page: number; page_size: number }

export type ResidentSort = 'name' | 'recent'

/** What the council needs to look the person up in its census; deleted once it decides, or after 15 days. */
export type ResidentRequestInput = { dni: string; full_name: string; postal_code: string; street: string }

/** A pending request to be verified, for the council's administrators (Veciños tab), the oldest first. */
export type ResidentRequest = ResidentRequestInput & {
  id: string
  user_id: string
  email: string // the account's
  account_name: string // the account's, "" when not set
  created_at: string
  expires_at: string // deleted then if nobody decided
}

export type ResidentRequestsPage = { requests: ResidentRequest[]; total: number; page: number; page_size: number }

/** The sign-in account (kept by Zitadel), edited from /perfil. */
export type Account = {
  given_name: string
  family_name: string
  email: string
  email_verified: boolean
  password_rules: { min_length: number; uppercase: boolean; lowercase: boolean; number: boolean; symbol: boolean }
}

/** The personal data; the preferences are saved apart. */
export type ProfileInput = {
  birth_date: string
  phone: string
  avatar_ref: string // a storage reference, never a URL
}

/** How the app speaks and what it shows or sends them; saved without the personal data. */
export type PreferencesInput = Pick<Me, 'preferred_language' | 'email_notifications' | 'email_reminders' | 'show_recommendations'>

// --- superadministrator area -------------------------------------------------

export type CouncilSummary = {
  slug: string
  name: string
  timezone: string
  contact_email: string
  contact_phone: string
  logo_url: string
  is_active: boolean
  created_at: string
  facility_count: number
  admin_count: number
  pending_invitation_count: number
  booking_count: number
}

export type CouncilDetail = {
  slug: string
  name: string
  timezone: string
  contact_email: string
  contact_phone: string
  municipality_code: string // INE code, for MeteoGalicia's forecast; "" = none
  logo_ref: string
  logo_url: string
  is_active: boolean
  booking_count: number
  admins: { user_id: string; email: string; full_name: string; role: string; since: string }[]
  invitations: { id: string; email: string; role: string; created_at: string }[]
}

// --- council indicators -------------------------------------------------------

export type IndicatorCounts = {
  bookings: number // confirmed + cancelled
  confirmed: number
  cancelled_by_user: number
  cancelled_by_council: number
  cancellation_rate: number | null // 0-1; null without bookings
  booked_hours: number
  open_hours: number // bookable time, closures taken out
  occupancy: number | null // 0-1; null when never open
  finished: number // past confirmed bookings at facilities with a QR reader
  attended: number
  no_show_rate: number | null
  // the residents' stars for the facilities after the period's bookings
  ratings: number
  rating_average: number | null // 1-5; null without ratings
  rating_stars: [number, number, number, number, number] // how many gave 1, 2, 3, 4 and 5 stars
}

export type FacilityIndicators = IndicatorCounts & { id: string; name: string; is_active: boolean; takes_checkin: boolean } // takes_checkin: a reader, the code, the door QR or staff

export type IndicatorTotals = IndicatorCounts & {
  active_users: number
  avg_party_size: number | null // people per confirmed booking, owner + invited
  median_lead_hours: number | null // how long before playing people book
}

export type WeekdayOccupancy = { weekday: number; booked_hours: number; open_hours: number; occupancy: number | null } // ISO weekday: 1 = Monday

/** The figures of one period: the one asked for, or the one it is compared with. */
export type PeriodFigures = {
  from: string // YYYY-MM-DD, both days included
  to: string
  totals: IndicatorTotals
  facilities: FacilityIndicators[]
  series: { period: string; confirmed: number; cancelled: number; ratings: number; rating_average: number | null }[] // rating_average: null when nobody rated
  weekdays: WeekdayOccupancy[]
}

export type IndicatorsCompare = 'previous' | 'year' | 'none'

export type Indicators = PeriodFigures & {
  bucket: 'day' | 'week' | 'month'
  timezone: string
  generated_at: string
  busy_hours: { weekday: number; hour: number; bookings: number }[] // ISO weekday: 1 = Monday
  compare: IndicatorsCompare
  previous: PeriodFigures | null // null with compare "none"
}

// days: the last n days up to today in the council's time zone (the API works out the dates),
// or from/to for a range of your own. compare: previous = the same number of days just before.
export type IndicatorsQuery = { days?: number; from?: string; to?: string; facility?: string; compare?: IndicatorsCompare }
// every export is anonymous: totals only, counts of 1 or 2 shown as "<3"
export type IndicatorsExport = { kind: 'summary' | 'daily' | 'hours' | 'usage'; excel: boolean }

export type CouncilInput = {
  slug: string
  name: string
  timezone: string
  contact_email: string
  contact_phone: string
  municipality_code: string // INE code (5 digits), for MeteoGalicia's forecast; "" = none
  logo_ref: string // a storage reference, never a URL
  is_active?: boolean // update only
  admin_email?: string // create only
}

/** The council's own SMTP relay; without one its emails use the platform's (env_relay). */
export type CouncilSmtp = {
  configured: boolean
  unreadable: boolean // sealed with another key: to be typed again
  host: string
  port: number
  username: string
  has_password: boolean // the password itself never comes back
  from: string
  updated_at: string | null
  env_relay: boolean
}
/** An empty password keeps the saved one. */
export type CouncilSmtpInput = { host: string; port: number; username: string; password: string; from: string }

export type Superadmin = { user_id: string; email: string; full_name: string; from_env: boolean; created_at: string }

/** Fired when the administration refuses a sign-in without a second factor (REQUIRE_ADMIN_MFA). */
export const MFA_REQUIRED_EVENT = 'librequenda:mfa-required'

/** Fired once when the API runs a newer build than this page (its X-App-Version differs): the page has to reload. */
export const NEW_VERSION_EVENT = 'librequenda:new-version'

// The build's version (VITE_APP_VERSION, the API's VERSION in the same deployment);
// empty in development, where nothing is checked.
const APP_VERSION: string = import.meta.env.VITE_APP_VERSION ?? ''
let newerVersion = ''

/** The API's version when it is not this page's, or "" while they match. */
export function newerApiVersion() {
  return newerVersion
}

/**
 * Every call to the API: the language, and the page's version, which the API
 * refuses with 409 once it runs another one. Each answer carries the API's
 * version, so a deployment is noticed on the next call, whichever it is.
 */
async function apiFetch(url: string, init: RequestInit = {}) {
  const headers = new Headers(init.headers)
  headers.set('Accept-Language', i18n.resolvedLanguage ?? 'gl')
  if (APP_VERSION) headers.set('X-App-Version', APP_VERSION)
  const res = await fetch(url, { ...init, headers })
  const apiVersion = res.headers.get('X-App-Version')
  if (APP_VERSION && apiVersion && apiVersion !== APP_VERSION && !newerVersion) {
    newerVersion = apiVersion
    window.dispatchEvent(new Event(NEW_VERSION_EVENT))
  }
  return res
}

/** Asks the API only for its version, for screens that may go long without a call. */
export async function checkApiVersion() {
  if (!APP_VERSION) return
  await apiFetch('/api/health/api', { method: 'HEAD' }).catch(() => undefined) // offline: the next check
}

export class ApiError extends Error {
  status: number

  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

type RequestOptions = { token?: string; method?: string; body?: unknown }

// Access tokens are short (Zitadel, 10 minutes by default) and renewed in the
// background; one that expired just before a request is renewed here once, and
// the request sent again with the new one (AuthWithRouter sets the renewal).
let renewToken: (() => Promise<string | null>) | null = null
export function onTokenExpired(renew: (() => Promise<string | null>) | null) {
  renewToken = renew
}

async function request<T>(path: string, { token, method = 'GET', body }: RequestOptions = {}, renewed = false): Promise<T> {
  const headers: Record<string, string> = {}
  if (token) headers.Authorization = `Bearer ${token}`
  if (body !== undefined) headers['Content-Type'] = 'application/json'

  const res = await apiFetch(`/api${path}`, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
  })
  // only user tokens (Zitadel's are JWTs); a reader's device key is never renewed
  if (res.status === 401 && token?.startsWith('ey') && !renewed && renewToken) {
    const fresh = await renewToken()
    if (fresh && fresh !== token) return request<T>(path, { token: fresh, method, body }, true)
  }
  if (res.status === 204) return undefined as T
  if (!res.ok) {
    let message = `HTTP ${res.status}`
    // the rate limiter answers in English only
    if (res.status === 429) throw new ApiError(429, i18n.t('common:rateLimited'))
    // an administrator signed in without a second factor: MfaRequiredNotice explains how to set one up
    if (res.status === 403 && res.headers.get('X-Mfa-Required')) window.dispatchEvent(new Event(MFA_REQUIRED_EVENT))
    try {
      const data = await res.json()
      message = data.error ?? message
    } catch {
      // keep the status line
    }
    throw new ApiError(res.status, message)
  }
  return res.json() as Promise<T>
}

function indicatorsParams({ days, from, to, facility, compare }: IndicatorsQuery) {
  const params = new URLSearchParams()
  if (from && to) {
    params.set('from', from)
    params.set('to', to)
  } else if (days) params.set('days', String(days))
  if (facility) params.set('facility', facility)
  if (compare) params.set('compare', compare)
  return params
}

/** One line of the audit log: what an administrator (or the system) did. Names come from the accounts as they are now. */
export type AuditEntry = {
  id: string
  at: string
  actor_name: string
  actor_role: 'superadmin' | 'council_admin' | 'resident' | 'system'
  actor_deleted: boolean // the account was erased
  tenant_slug: string
  tenant_name: string
  action: string // a stable code, e.g. facility.updated
  target_type: string
  target_id: string
  target_name: string
  target_exists: boolean // a facility that still exists, to link to
  summary: string
}
export type AuditPage = { entries: AuditEntry[]; next: string | null }
/** What the log is narrowed to: an action code or prefix, days (YYYY-MM-DD), a council (superadmins), and a page. */
export type AuditFilters = { action?: string; from?: string; to?: string; council?: string; before?: string }

function auditParams(filters: AuditFilters) {
  const params = new URLSearchParams()
  for (const [key, value] of Object.entries(filters)) if (value) params.set(key, value)
  return params
}

/** The filtered audit log as a CSV, fetched with the token and saved. */
async function downloadAudit(token: string, path: string, filters: AuditFilters) {
  const params = auditParams(filters)
  params.set('format', 'csv')
  const res = await apiFetch(`/api${path}?${params}`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok) throw new ApiError(res.status, `HTTP ${res.status}`)
  const url = URL.createObjectURL(await res.blob())
  const link = document.createElement('a')
  link.href = url
  link.download = `audit-log-${new Date().toISOString().slice(0, 10)}.csv`
  link.click()
  URL.revokeObjectURL(url)
}

/** GET …/indicators/export: the CSV comes with the token, so it is fetched and saved as a file. */
async function exportIndicators(token: string, slug: string, query: IndicatorsQuery, { kind, excel }: IndicatorsExport) {
  const params = indicatorsParams(query)
  params.set('kind', kind)
  if (excel) params.set('excel', '1')
  const res = await apiFetch(`/api/tenants/${encodeURIComponent(slug)}/admin/indicators/export?${params}`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok) {
    const data = await res.json().catch(() => ({}))
    throw new ApiError(res.status, data.error ?? `HTTP ${res.status}`)
  }
  const url = URL.createObjectURL(await res.blob())
  const link = document.createElement('a')
  link.href = url
  link.download = `${slug}-${kind}-${query.from && query.to ? `${query.from}_${query.to}` : `${query.days}d-${new Date().toISOString().slice(0, 10)}`}.csv`
  link.click()
  URL.revokeObjectURL(url)
}

/** GET …/admin/export: the council's full data as a ZIP, fetched with the token and saved; `personal` adds the residents' personal data. */
async function exportCouncilData(token: string, slug: string, personal: boolean) {
  const res = await apiFetch(`/api/tenants/${encodeURIComponent(slug)}/admin/export${personal ? '?personal=1' : ''}`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok) {
    if (res.status === 429) throw new ApiError(429, i18n.t('common:rateLimited'))
    const data = await res.json().catch(() => ({}))
    throw new ApiError(res.status, data.error ?? `HTTP ${res.status}`)
  }
  const url = URL.createObjectURL(await res.blob())
  const link = document.createElement('a')
  link.href = url
  link.download = `librequenda-${slug}-${new Date().toISOString().slice(0, 10)}.zip`
  link.click()
  URL.revokeObjectURL(url)
}

/** GET /me/export: the user's data as a file (RXPD / GDPR), fetched with the token and saved. */
async function exportMyData(token: string, format: 'json' | 'csv') {
  const res = await apiFetch(`/api/me/export?format=${format}`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok) {
    if (res.status === 429) throw new ApiError(429, i18n.t('common:rateLimited'))
    const data = await res.json().catch(() => ({}))
    throw new ApiError(res.status, data.error ?? `HTTP ${res.status}`)
  }
  const url = URL.createObjectURL(await res.blob())
  const link = document.createElement('a')
  link.href = url
  link.download = `librequenda-${new Date().toISOString().slice(0, 10)}.${format}`
  link.click()
  URL.revokeObjectURL(url)
}

/** POST /media: multipart upload into the owner's private storage, answered with { ref, url } (a presigned link). */
async function uploadImage(token: string, folder: MediaFolder, file: File, owner?: MediaOwner): Promise<StoredImage> {
  const body = new FormData()
  body.append('folder', folder)
  if (owner && 'council' in owner) body.append('council', owner.council)
  if (owner && 'facility' in owner) body.append('facility', owner.facility)
  body.append('file', file)
  const res = await apiFetch('/api/media', {
    method: 'POST',
    headers: { Authorization: `Bearer ${token}` },
    body,
  })
  if (!res.ok) {
    const data = await res.json().catch(() => ({}))
    throw new ApiError(res.status, data.error ?? `HTTP ${res.status}`)
  }
  return res.json()
}

export const api = {
  uploadImage,
  tenants: () => request<{ tenants: Tenant[] }>('/tenants'),
  tenant: (slug: string) => request<CouncilPage>(`/tenants/${encodeURIComponent(slug)}`),
  // the councils' polls for the signed-in person; one vote each, changed while open
  myPolls: (token: string) => request<{ polls: MyPoll[] }>('/me/polls', { token }),
  vote: (token: string, pollId: string, optionId: string) => request<void>(`/me/polls/${pollId}/vote`, { token, method: 'PUT', body: { option_id: optionId } }),
  tags: () => request<TagCatalogue>('/tags'),
  branding: () => request<Branding>('/branding'),
  // near: nearest first and each one's distance_km (the API sorts); radiusKm keeps those within it.
  // The map shows them all, so every page is fetched, in the API's order.
  // nearest first within the radius, and only the ones rated at least minRating: the API does it
  facilities: async (near?: { latitude: number; longitude: number } | null, radiusKm = 0, minRating = 0) => {
    const query = (near ? `&lat=${near.latitude}&lon=${near.longitude}${radiusKm > 0 ? `&radius_km=${radiusKm}` : ''}` : '') + (minRating > 0 ? `&min_rating=${minRating}` : '')
    const facilities: FacilityCard[] = []
    for (let page = 1; ; page++) {
      const answer = await request<{ facilities: FacilityCard[]; pagination: Pagination }>(`/facilities?page=${page}${query}`)
      facilities.push(...answer.facilities)
      if (!answer.pagination?.has_more) return { facilities }
    }
  },
  day: (slug: string, date?: string) =>
    request<Day>(`/tenants/${encodeURIComponent(slug)}/day${date ? `?date=${date}` : ''}`),
  me: (token: string) => request<Me>('/me', { token }),
  updateProfile: (token: string, input: ProfileInput) => request<Me>('/me/profile', { token, method: 'PUT', body: input }),
  updatePreferences: (token: string, input: PreferencesInput) => request<Me>('/me/preferences', { token, method: 'PUT', body: input }),
  personalQr: (token: string) => request<PersonalQr>('/me/qr', { token }),
  calendarFeed: {
    get: (token: string) => request<CalendarFeed>('/me/calendar', { token }),
    // turns the feed on, or gives it a new address: the old one stops working
    reset: (token: string) => request<CalendarFeed>('/me/calendar', { token, method: 'POST' }),
    turnOff: (token: string) => request<void>('/me/calendar', { token, method: 'DELETE' }),
  },
  exportMyData,
  // erases the account for good: upcoming bookings are cancelled and the sign-in account is deleted
  deleteAccount: (token: string) => request<void>('/me', { token, method: 'DELETE' }),
  account: {
    get: (token: string) => request<Account>('/me/account', { token }),
    setName: (token: string, givenName: string, familyName: string) =>
      request<void>('/me/account/name', { token, method: 'PUT', body: { given_name: givenName, family_name: familyName } }),
    setPassword: (token: string, currentPassword: string, newPassword: string) =>
      request<void>('/me/account/password', { token, method: 'PUT', body: { current_password: currentPassword, new_password: newPassword } }),
    // the new address gets a code; the link in that email comes back to /perfil?verificar_correo=<code>
    setEmail: (token: string, email: string) => request<void>('/me/account/email', { token, method: 'PUT', body: { email } }),
    resendEmailCode: (token: string) => request<void>('/me/account/email/resend', { token, method: 'POST' }),
    verifyEmail: (token: string, code: string) => request<void>('/me/account/email/verify', { token, method: 'POST', body: { code } }),
  },
  myBookings: (token: string) => request<MyBookings>('/me/bookings', { token }),
  // asking the council to verify the person as its resident (residents-only facilities), or withdrawing it
  requestResidency: (token: string, slug: string, input: ResidentRequestInput) =>
    request<{ requested_at: string }>(`/tenants/${encodeURIComponent(slug)}/resident-request`, { token, method: 'POST', body: input }),
  withdrawResidency: (token: string, slug: string) =>
    request<void>(`/tenants/${encodeURIComponent(slug)}/resident-request`, { token, method: 'DELETE' }),
  reportIssue: (token: string, input: IssueReport) => request<{ id: string }>('/me/issues', { token, method: 'POST', body: input }),
  // near: where the person is (the home page's «near me»), in the body; checked, never stored
  recommendations: (token: string, near?: { latitude: number; longitude: number } | null) =>
    request<{ recommendations: Recommendation[] }>('/me/recommendations', { token, method: 'POST', body: near ?? {} }),
  // "For you" on or off, also from the section itself
  setShowRecommendations: (token: string, show: boolean) => request<Me>('/me/recommendations', { token, method: 'PUT', body: { show } }),
  // a facility's forecast when enabled, from MeteoGalicia through the API
  weather: (slug: string, facility: string, date: string) =>
    request<Weather>(`/tenants/${encodeURIComponent(slug)}/facilities/${encodeURIComponent(facility)}/weather?date=${date}`),
  // signed in (token), each public booking also says who joined; signed out, only who answers for it
  facilityPage: (slug: string, facility: string, date?: string, token?: string) =>
    request<FacilityPage>(`/tenants/${encodeURIComponent(slug)}/facilities/${encodeURIComponent(facility)}${date ? `?date=${date}` : ''}`, { token }),
  // acceptRules: the resident ticked "I have read the rules"; the API refuses the booking otherwise
  // isPublic: anyone signed in may find it in «For you» and join it (facilities with room for more than one)
  book: (token: string, facilityId: string, startsAt: string, acceptRules: boolean, isPublic = false, description = '') =>
    request<Booking>('/me/bookings', {
      token,
      method: 'POST',
      body: { facility_id: facilityId, starts_at: startsAt, accept_rules: acceptRules, public: isPublic, public_description: description },
    }),
  setPublic: (token: string, bookingId: string, isPublic: boolean, description?: string) =>
    request<void>(`/me/bookings/${bookingId}/public`, { token, method: 'PUT', body: { public: isPublic, public_description: description } }),
  joinPublic: (token: string, bookingId: string) => request<void>(`/me/bookings/${bookingId}/join`, { token, method: 'POST' }),
  // another free turn of the same facility, any day it can be booked
  changeBooking: (token: string, bookingId: string, startsAt: string) =>
    request<Booking>(`/me/bookings/${bookingId}`, { token, method: 'PATCH', body: { starts_at: startsAt } }),
  cancel: (token: string, bookingId: string) =>
    request<void>(`/me/bookings/${bookingId}`, { token, method: 'DELETE' }),

  ratings: {
    get: (token: string, bookingId: string) => request<BookingRating>(`/me/bookings/${bookingId}/rating`, { token }),
    save: (token: string, bookingId: string, input: RatingInput) =>
      request<BookingRating>(`/me/bookings/${bookingId}/rating`, { token, method: 'PUT', body: input }),
  },
  invites: {
    get: (token: string, bookingId: string) => request<Invite>(`/me/bookings/${bookingId}/invite`, { token }),
    // creates the link, or replaces it so the old one stops working
    share: (token: string, bookingId: string) => request<Invite>(`/me/bookings/${bookingId}/invite`, { token, method: 'POST' }),
    removeParticipant: (token: string, bookingId: string, participantId: string) =>
      request<void>(`/me/bookings/${bookingId}/participants/${participantId}`, { token, method: 'DELETE' }),
    leave: (token: string, bookingId: string) => request<void>(`/me/bookings/${bookingId}/participation`, { token, method: 'DELETE' }),
    open: (inviteToken: string) => request<PublicInvite>(`/invites/${encodeURIComponent(inviteToken)}`),
    join: (token: string, inviteToken: string) =>
      request<void>(`/me/invites/${encodeURIComponent(inviteToken)}/join`, { token, method: 'POST' }),
    joinAsGuest: (inviteToken: string, name: string, age: number) =>
      request<void>(`/invites/${encodeURIComponent(inviteToken)}/join`, { method: 'POST', body: { name, age } }),
  },

  notifications: {
    list: (token: string) => request<Notifications>('/me/notifications', { token }),
    markAllRead: (token: string) => request<void>('/me/notifications/read', { token, method: 'POST' }),
  },

  council: {
    // the analytics assistant, about this council's data only
    askAnalytics: (token: string, slug: string, question: string) =>
      request<AnalyticsAnswer>(`/tenants/${encodeURIComponent(slug)}/admin/analytics/ask`, { token, method: 'POST', body: { question } }),
    audit: (token: string, slug: string, filters: AuditFilters) =>
      request<AuditPage>(`/tenants/${encodeURIComponent(slug)}/admin/audit?${auditParams(filters)}`, { token }),
    downloadAudit: (token: string, slug: string, filters: AuditFilters) =>
      downloadAudit(token, `/tenants/${encodeURIComponent(slug)}/admin/audit`, filters),
    facilities: (token: string, slug: string) =>
      request<{ facilities: AdminFacilitySummary[] }>(`/tenants/${encodeURIComponent(slug)}/admin/facilities`, { token }),
    facility: (token: string, slug: string, id: string) =>
      request<AdminFacility>(`/tenants/${encodeURIComponent(slug)}/admin/facilities/${id}`, { token }),
    // the schedule goes with it, so a new facility is set up in one go
    createFacility: (token: string, slug: string, input: FacilityInput & { schedule?: { windows: WeeklyWindow[] } }) =>
      request<AdminFacility>(`/tenants/${encodeURIComponent(slug)}/admin/facilities`, { token, method: 'POST', body: input }),
    updateFacility: (token: string, slug: string, id: string, input: FacilityInput) =>
      request<AdminFacility>(`/tenants/${encodeURIComponent(slug)}/admin/facilities/${id}`, { token, method: 'PUT', body: input }),
    // a copy with every setting, switched off, under its own name and address
    duplicateFacility: (token: string, slug: string, id: string, input: { name: string; slug: string }) =>
      request<AdminFacility>(`/tenants/${encodeURIComponent(slug)}/admin/facilities/${id}/duplicate`, { token, method: 'POST', body: input }),
    deleteFacility: (token: string, slug: string, id: string) =>
      request<void>(`/tenants/${encodeURIComponent(slug)}/admin/facilities/${id}`, { token, method: 'DELETE' }),
    // deactivating keeps the bookings unless cancelBookings (then a reason for residents is required)
    setActive: (token: string, slug: string, id: string, active: boolean, cancelBookings = false, reason = '') =>
      request<{ facility: AdminFacility; cancelled: number }>(`/tenants/${encodeURIComponent(slug)}/admin/facilities/${id}/active`, {
        token,
        method: 'PUT',
        body: { active, cancel_bookings: cancelBookings, reason },
      }),
    previewClosure: (token: string, slug: string, id: string, input: ClosureInput) =>
      request<{ bookings_to_cancel: number }>(`/tenants/${encodeURIComponent(slug)}/admin/facilities/${id}/closures/preview`, {
        token,
        method: 'POST',
        body: input,
      }),
    createClosure: (token: string, slug: string, id: string, input: ClosureInput) =>
      request<{ closure: Closure; cancelled: number; closures: Closure[] }>(`/tenants/${encodeURIComponent(slug)}/admin/facilities/${id}/closures`, {
        token,
        method: 'POST',
        body: input,
      }),
    deleteClosure: (token: string, slug: string, id: string, closureId: string) =>
      request<void>(`/tenants/${encodeURIComponent(slug)}/admin/facilities/${id}/closures/${closureId}`, { token, method: 'DELETE' }),
    replaceSchedule: (token: string, slug: string, id: string, windows: WeeklyWindow[]) =>
      request<AdminFacility>(`/tenants/${encodeURIComponent(slug)}/admin/facilities/${id}/schedule`, {
        token,
        method: 'PUT',
        body: { windows },
      }),
    devices: (token: string, slug: string, facilityId: string) =>
      request<{ devices: CheckinDevice[] }>(`/tenants/${encodeURIComponent(slug)}/admin/facilities/${facilityId}/devices`, { token }),
    createDevice: (token: string, slug: string, facilityId: string, name: string) =>
      request<CreatedCheckinDevice>(`/tenants/${encodeURIComponent(slug)}/admin/facilities/${facilityId}/devices`, {
        token,
        method: 'POST',
        body: { name },
      }),
    deleteDevice: (token: string, slug: string, facilityId: string, deviceId: string) =>
      request<void>(`/tenants/${encodeURIComponent(slug)}/admin/facilities/${facilityId}/devices/${deviceId}`, { token, method: 'DELETE' }),
    checkinSettings: (token: string, slug: string, facilityId: string) =>
      request<CheckinSettings>(`/tenants/${encodeURIComponent(slug)}/admin/facilities/${facilityId}/checkin`, { token }),
    saveCheckinSettings: (token: string, slug: string, facilityId: string, input: CheckinSettingsInput) =>
      request<CheckinSettings>(`/tenants/${encodeURIComponent(slug)}/admin/facilities/${facilityId}/checkin`, { token, method: 'PUT', body: input }),
    rotateCheckinQr: (token: string, slug: string, facilityId: string) =>
      request<CheckinSettings>(`/tenants/${encodeURIComponent(slug)}/admin/facilities/${facilityId}/checkin/qr`, { token, method: 'POST' }),
    rotateCheckinNfc: (token: string, slug: string, facilityId: string) =>
      request<CheckinSettings>(`/tenants/${encodeURIComponent(slug)}/admin/facilities/${facilityId}/checkin/nfc`, { token, method: 'POST' }),
    rotateCheckinCode: (token: string, slug: string, facilityId: string) =>
      request<CheckinSettings>(`/tenants/${encodeURIComponent(slug)}/admin/facilities/${facilityId}/checkin/code`, { token, method: 'POST' }),
    staffDay: (token: string, slug: string, facilityId: string, date: string) =>
      request<StaffDay>(`/tenants/${encodeURIComponent(slug)}/admin/facilities/${facilityId}/bookings?date=${date}`, { token }),
    staffBook: (token: string, slug: string, input: StaffBookingInput) =>
      request<StaffBooking>(`/tenants/${encodeURIComponent(slug)}/admin/bookings`, { token, method: 'POST', body: input }),
    // the resident is told if they have an account, with the reason when there is one
    // staff sign someone up for one of the council's events: an account (email) or a name and an age
    addEventParticipant: (token: string, slug: string, bookingId: string, input: { email: string } | { name: string; age: number }) =>
      request<void>(`/tenants/${encodeURIComponent(slug)}/admin/bookings/${bookingId}/participants`, { token, method: 'POST', body: input }),
    staffCancel: (token: string, slug: string, bookingId: string, reason: string) =>
      request<void>(`/tenants/${encodeURIComponent(slug)}/admin/bookings/${bookingId}/cancel`, { token, method: 'POST', body: { reason } }),
    eventPeople: (token: string, slug: string, eventId: string) =>
      request<{ people: EventPerson[] }>(`/tenants/${encodeURIComponent(slug)}/admin/events/${eventId}/people`, { token }),
    removeEventParticipant: (token: string, slug: string, eventId: string, participantId: string) =>
      request<void>(`/tenants/${encodeURIComponent(slug)}/admin/events/${eventId}/people/${participantId}`, { token, method: 'DELETE' }),
    events: (token: string, slug: string) => request<{ timezone: string; events: CouncilEvent[] }>(`/tenants/${encodeURIComponent(slug)}/admin/events`, { token }),
    // except: the event being moved, which is not in its own way
    previewEvent: (token: string, slug: string, input: EventInput, except?: string) =>
      request<EventInTheWay>(`/tenants/${encodeURIComponent(slug)}/admin/events/preview${except ? `?except=${except}` : ''}`, { token, method: 'POST', body: input }),
    createEvent: (token: string, slug: string, input: EventInput) =>
      request<CouncilEvent>(`/tenants/${encodeURIComponent(slug)}/admin/events`, { token, method: 'POST', body: input }),
    // moved, who joined is told the new time
    updateEvent: (token: string, slug: string, id: string, input: EventInput) =>
      request<CouncilEvent>(`/tenants/${encodeURIComponent(slug)}/admin/events/${id}`, { token, method: 'PUT', body: input }),
    // staff mark that the person came in; undo only within a few minutes of a staff entry
    staffCheckIn: (token: string, slug: string, bookingId: string) =>
      request<StaffBooking>(`/tenants/${encodeURIComponent(slug)}/admin/bookings/${bookingId}/checkin`, { token, method: 'POST' }),
    undoStaffCheckIn: (token: string, slug: string, bookingId: string) =>
      request<StaffBooking>(`/tenants/${encodeURIComponent(slug)}/admin/bookings/${bookingId}/checkin`, { token, method: 'DELETE' }),
    indicators: (token: string, slug: string, query: IndicatorsQuery) =>
      request<Indicators>(`/tenants/${encodeURIComponent(slug)}/admin/indicators?${indicatorsParams(query)}`, { token }),
    exportIndicators,
    exportData: exportCouncilData,
    issues: (token: string, slug: string, filter: { facility?: string; status?: IssueStatus | '' } = {}) => {
      const query = new URLSearchParams()
      if (filter.facility) query.set('facility', filter.facility)
      if (filter.status) query.set('status', filter.status)
      return request<{ issues: CouncilIssue[] }>(`/tenants/${encodeURIComponent(slug)}/admin/issues${query.size ? `?${query}` : ''}`, { token })
    },
    residents: (token: string, slug: string, filter: { q?: string; sort?: ResidentSort; page?: number } = {}) => {
      const query = new URLSearchParams()
      if (filter.q) query.set('q', filter.q)
      if (filter.sort) query.set('sort', filter.sort)
      if (filter.page && filter.page > 1) query.set('page', String(filter.page))
      return request<ResidentsPage>(`/tenants/${encodeURIComponent(slug)}/admin/residents${query.size ? `?${query}` : ''}`, { token })
    },
    addResident: (token: string, slug: string, email: string) =>
      request<void>(`/tenants/${encodeURIComponent(slug)}/admin/residents`, { token, method: 'POST', body: { email } }),
    removeResident: (token: string, slug: string, userId: string) =>
      request<void>(`/tenants/${encodeURIComponent(slug)}/admin/residents/${userId}`, { token, method: 'DELETE' }),
    residentRequests: (token: string, slug: string, page = 1) =>
      request<ResidentRequestsPage>(`/tenants/${encodeURIComponent(slug)}/admin/resident-requests${page > 1 ? `?page=${page}` : ''}`, { token }),
    residentRequestCount: (token: string, slug: string) =>
      request<{ pending: number }>(`/tenants/${encodeURIComponent(slug)}/admin/resident-requests/count`, { token }),
    // either way the request, with its DNI and address, is deleted and the person told
    verifyResidentRequest: (token: string, slug: string, id: string) =>
      request<void>(`/tenants/${encodeURIComponent(slug)}/admin/resident-requests/${id}/verify`, { token, method: 'POST' }),
    discardResidentRequest: (token: string, slug: string, id: string) =>
      request<void>(`/tenants/${encodeURIComponent(slug)}/admin/resident-requests/${id}`, { token, method: 'DELETE' }),
    // the current ones, or the history (expired)
    notices: (token: string, slug: string, past = false) =>
      request<{ timezone: string; notices: AdminNotice[] }>(`/tenants/${encodeURIComponent(slug)}/admin/notices${past ? '?state=past' : ''}`, { token }),
    createNotice: (token: string, slug: string, input: NoticeInput) =>
      request<AdminNotice>(`/tenants/${encodeURIComponent(slug)}/admin/notices`, { token, method: 'POST', body: input }),
    deleteNotice: (token: string, slug: string, id: string) => request<void>(`/tenants/${encodeURIComponent(slug)}/admin/notices/${id}`, { token, method: 'DELETE' }),
    // the open ones, or the history (closed)
    polls: (token: string, slug: string, past = false) =>
      request<{ timezone: string; polls: AdminPoll[] }>(`/tenants/${encodeURIComponent(slug)}/admin/polls${past ? '?state=past' : ''}`, { token }),
    createPoll: (token: string, slug: string, input: PollInput) =>
      request<AdminPoll>(`/tenants/${encodeURIComponent(slug)}/admin/polls`, { token, method: 'POST', body: input }),
    closePoll: (token: string, slug: string, id: string) => request<void>(`/tenants/${encodeURIComponent(slug)}/admin/polls/${id}/close`, { token, method: 'POST' }),
    deletePoll: (token: string, slug: string, id: string) => request<void>(`/tenants/${encodeURIComponent(slug)}/admin/polls/${id}`, { token, method: 'DELETE' }),
    newIssueCount: (token: string, slug: string) => request<{ new: number }>(`/tenants/${encodeURIComponent(slug)}/admin/issues/count`, { token }),
    changeIssue: (token: string, slug: string, id: string, input: { status: IssueStatus; staff_note: string }) =>
      request<CouncilIssue>(`/tenants/${encodeURIComponent(slug)}/admin/issues/${id}`, { token, method: 'PUT', body: input }),
    bookingRules: (token: string, slug: string) => request<BookingRules>(`/tenants/${encodeURIComponent(slug)}/admin/booking-rules`, { token }),
    saveFacilityRules: (token: string, slug: string, facilityId: string, input: { custom_rules: boolean; rules: BookingRules }) =>
      request<AdminFacility>(`/tenants/${encodeURIComponent(slug)}/admin/facilities/${facilityId}/booking-rules`, { token, method: 'PUT', body: input }),
    saveBookingRules: (token: string, slug: string, rules: BookingRules) =>
      request<BookingRules>(`/tenants/${encodeURIComponent(slug)}/admin/booking-rules`, { token, method: 'PUT', body: rules }),
    smtp: (token: string, slug: string) => request<CouncilSmtp>(`/tenants/${encodeURIComponent(slug)}/admin/smtp`, { token }),
    saveSmtp: (token: string, slug: string, input: CouncilSmtpInput) =>
      request<CouncilSmtp>(`/tenants/${encodeURIComponent(slug)}/admin/smtp`, { token, method: 'PUT', body: input }),
    deleteSmtp: (token: string, slug: string) => request<void>(`/tenants/${encodeURIComponent(slug)}/admin/smtp`, { token, method: 'DELETE' }),
    // emails the administrator through the relay the council's emails use now
    testSmtp: (token: string, slug: string) =>
      request<{ to: string; via: 'council' | 'environment' }>(`/tenants/${encodeURIComponent(slug)}/admin/smtp/test`, { token, method: 'POST' }),
    geocode: (token: string, query: string) =>
      request<{ results: GeocodeResult[] }>(`/geocode?q=${encodeURIComponent(query)}`, { token }),
  },

  // the resident's own phone at the door
  entrance: {
    get: (token: string, slug: string, facility: string) =>
      request<Entrance>(`/tenants/${encodeURIComponent(slug)}/facilities/${encodeURIComponent(facility)}/entrance`, { token }),
    checkIn: (token: string, slug: string, facility: string, input: EntranceInput) =>
      request<EntranceResult>(`/tenants/${encodeURIComponent(slug)}/facilities/${encodeURIComponent(facility)}/entrance`, {
        token,
        method: 'POST',
        body: input,
      }),
    // flash booking with the entry code: the facilities the phone is at (the location goes in the body, never stored)
    nearby: (token: string, position: Position) => request<Nearby>('/entrance/nearby', { token, method: 'POST', body: position }),
  },

  // the device key goes where a user token would
  reader: {
    me: (deviceKey: string) => request<Reader>('/reader', { token: deviceKey }),
    checkIn: (deviceKey: string, code: string) =>
      request<CheckinResult>('/reader/checkin', { token: deviceKey, method: 'POST', body: { code } }),
  },

  admin: {
    audit: (token: string, filters: AuditFilters) => request<AuditPage>(`/admin/audit?${auditParams(filters)}`, { token }),
    downloadAudit: (token: string, filters: AuditFilters) => downloadAudit(token, '/admin/audit', filters),
    // platform-wide booking rules (superadministrators)
    // the analytics assistant: its language model (the key is never sent back), and questions about every council
    aiSettings: (token: string) => request<AISettings>('/admin/ai-settings', { token }),
    saveAISettings: (token: string, input: AISettingsInput) => request<AISettings>('/admin/ai-settings', { token, method: 'PUT', body: input }),
    bookingPolicy: (token: string) => request<BookingPolicy>('/admin/booking-policy', { token }),
    saveBookingPolicy: (token: string, policy: BookingPolicy) => request<BookingPolicy>('/admin/booking-policy', { token, method: 'PUT', body: policy }),
    councils: (token: string) => request<{ tenants: CouncilSummary[] }>('/admin/tenants', { token }),
    council: (token: string, slug: string) => request<CouncilDetail>(`/admin/tenants/${encodeURIComponent(slug)}`, { token }),
    createCouncil: (token: string, input: CouncilInput) =>
      request<CouncilDetail>('/admin/tenants', { token, method: 'POST', body: input }),
    updateCouncil: (token: string, slug: string, input: CouncilInput) =>
      request<CouncilDetail>(`/admin/tenants/${encodeURIComponent(slug)}`, { token, method: 'PUT', body: input }),
    deleteCouncil: (token: string, slug: string) =>
      request<void>(`/admin/tenants/${encodeURIComponent(slug)}`, { token, method: 'DELETE' }),
    invite: (token: string, slug: string, email: string) =>
      request<CouncilDetail>(`/admin/tenants/${encodeURIComponent(slug)}/invitations`, { token, method: 'POST', body: { email } }),
    withdrawInvitation: (token: string, slug: string, id: string) =>
      request<void>(`/admin/tenants/${encodeURIComponent(slug)}/invitations/${id}`, { token, method: 'DELETE' }),
    removeAdmin: (token: string, slug: string, userId: string) =>
      request<void>(`/admin/tenants/${encodeURIComponent(slug)}/admins/${userId}`, { token, method: 'DELETE' }),
    superadmins: (token: string) => request<{ superadmins: Superadmin[] }>('/admin/superadmins', { token }),
    addSuperadmin: (token: string, email: string) =>
      request<void>('/admin/superadmins', { token, method: 'POST', body: { email } }),
    removeSuperadmin: (token: string, userId: string) =>
      request<void>(`/admin/superadmins/${userId}`, { token, method: 'DELETE' }),
    // the tag catalogue: each change answers with the whole catalogue as it is now
    createTagCategory: (token: string, name: string) =>
      request<TagCatalogue>('/admin/tag-categories', { token, method: 'POST', body: { name } }),
    updateTagCategory: (token: string, id: string, input: { name: string; position?: number }) =>
      request<TagCatalogue>(`/admin/tag-categories/${id}`, { token, method: 'PUT', body: input }),
    deleteTagCategory: (token: string, id: string) => request<TagCatalogue>(`/admin/tag-categories/${id}`, { token, method: 'DELETE' }),
    createTag: (token: string, input: { category_id: string; name: string; slug: string }) =>
      request<TagCatalogue>('/admin/tags', { token, method: 'POST', body: input }),
    updateTag: (token: string, id: string, input: { category_id: string; name: string }) =>
      request<TagCatalogue>(`/admin/tags/${id}`, { token, method: 'PUT', body: input }),
    deleteTag: (token: string, id: string) => request<TagCatalogue>(`/admin/tags/${id}`, { token, method: 'DELETE' }),
    updateBranding: (token: string, input: BrandingInput) =>
      request<Branding>('/admin/branding', { token, method: 'PUT', body: input }),
  },
}
