// Typed client for the LibreQuenda API (served by Caddy under /api)
import i18n from '@/i18n'

export type SlotStatus = 'free' | 'booked' | 'closed' | 'past' | 'too_far'

export type Slot = {
  starts_at: string
  ends_at: string
  status: SlotStatus
}

export type Facility = {
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
  max_advance_days: number
  min_cancel_hours: number
  max_active_bookings_per_user: number
  min_age: number | null // age control: null = none
  usage_rules: string
  windows: { opens_at: string; closes_at: string }[]
  closures: { starts_at: string; ends_at: string; reason: string }[]
  slots: Slot[]
}

// contact: where residents send accessibility feedback and complaints (RD 1112/2018); "" when not given
export type Tenant = { slug: string; name: string; logo_url: string; contact_email: string; contact_phone: string }

/** The home page's council card: how the council uses LibreQuenda. */
export type TenantStats = {
  facilities: number
  sports: string[]
  free_today: number // turns still free today across its facilities
  bookings: number // confirmed, past and upcoming
  residents: number // distinct accounts that have booked
}
export type GeoPoint = { latitude: number; longitude: number }
export type TenantCard = Tenant & { stats: TenantStats }

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
}

/** A facility tag from the platform's catalogue; facilities and addresses use its slug. */
export type Tag = { id: string; slug: string; name: string; facility_count: number }
export type TagCategory = { id: string; name: string; position: number; tags: Tag[] }
export type TagCatalogue = { categories: TagCategory[] }

/** An uploaded image: forms save the ref (storage reference); screens only show the url. */
export type StoredImage = { ref: string; url: string }
export type MediaFolder = 'branding' | 'councils' | 'facilities' | 'users'

/** How this installation presents itself, set by a superadministrator: name and logo. */
export type Branding = {
  name: string // what to show: the custom name, or LibreQuenda
  custom_name: string // "" = LibreQuenda
  logo_ref: string // "" = the LibreQuenda logo
  logo_url: string
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

export type FacilityPage = {
  tenant: Tenant
  timezone: string
  date: string
  now: string
  facility: Facility
  schedule: WeeklyWindow[]
}

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
  max_advance_days: number
  min_cancel_hours: number
  max_active_bookings_per_user: number
  min_age: number | null // age control: null = none, else 1 to 80
  usage_rules: string
}

export type AdminFacility = FacilityInput & {
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
export type CheckinMethod = 'reader' | 'printed_qr' | 'entry_code' | 'staff'
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
  'printed_qr' | 'entry_code' | 'radius_m' | 'flash_booking' | 'staff_confirm' | 'qr_reader' | 'staff_contacts'
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
  entry_code: boolean
  radius_m: number
  flash_booking: boolean
  booking: { starts_at: string; ends_at: string; checked_in_at: string | null } | null // theirs, that can go in now
}
/** The proof of being at the door, and where the phone is (checked, never stored). */
export type EntranceInput = {
  qr_token?: string
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
  checked_in_at: string | null
  checkin_ways?: CheckinMethod[] | null // owner only: the ways to check in the facility takes, for the check-in button
  staff_contacts?: StaffContact[] // owner only, with staff confirmation on: who to ask on site
  // past confirmed bookings only, from the QR readers: unknown = no reader when it began, so nobody could check in
  attendance: '' | 'attended' | 'no_show' | 'unknown'
  // invites: the owner shares a link and others fill the facility's capacity
  role: 'owner' | 'participant' // participant: joined someone else's booking through an invite
  owner_name: string // participant only
  capacity: number // people including the owner; 0 when the facility takes no invites
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
  usage_rules: string
}

// --- notifications (polled) ---------------------------------------------------

export type NotificationType =
  | 'booking_confirmed'
  | 'booking_changed'
  | 'booking_cancelled'
  | 'booking_cancelled_by_council'
  | 'booking_reminder'
  | 'council_admin_granted'

export type AppNotification = {
  id: string
  type: NotificationType
  facility_name: string
  starts_at: string | null
  ends_at: string | null
  reason: string // council cancellations
  closed_from: string | null // council cancellations caused by a closure
  closed_until: string | null
  by_staff: boolean // a council cancellation of this one booking, by staff
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
  kind: 'usual_today' | 'favourite' | 'similar'
  facility: FacilityCard
  starts_at: string // usual_today: that free turn; favourite and similar: the next free turn today
  times?: number // usual_today: bookings that weekday and time; favourite: bookings there
  weekday?: number // usual_today: ISO, 1 = Monday
  because?: string // similar: the facility of theirs it is like
  shared_tags?: string[] // similar: the tags (slugs) they share
}

export type MyBookings = { upcoming: Booking[]; history: Booking[] }

export type Me = {
  id: string
  email: string
  email_verified: boolean
  full_name: string
  is_superadmin: boolean
  roles: { tenant_id: string; tenant_slug: string; tenant_name: string; role: 'admin' }[]
  // profile kept by LibreQuenda; name and email belong to Zitadel
  birth_date: string // YYYY-MM-DD, "" until filled in
  phone: string
  avatar_ref: string
  avatar_url: string
  preferred_language: string
  email_reminders: boolean
  show_recommendations: boolean // "For you" on the home page
  profile_complete: boolean // the date of birth is required before booking
  // council roles waiting for this person: granted once the email is verified
  pending_invitations?: { tenant_name: string; tenant_slug: string }[]
}

/** The sign-in account (kept by Zitadel), edited from /perfil. */
export type Account = {
  given_name: string
  family_name: string
  email: string
  email_verified: boolean
  password_rules: { min_length: number; uppercase: boolean; lowercase: boolean; number: boolean; symbol: boolean }
}

export type ProfileInput = {
  birth_date: string
  phone: string
  avatar_ref: string // a storage reference, never a URL
  preferred_language: string
  email_reminders: boolean
  show_recommendations?: boolean // left as it is when not sent
}

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
  series: { period: string; confirmed: number; cancelled: number }[]
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

export class ApiError extends Error {
  status: number

  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

type RequestOptions = { token?: string; method?: string; body?: unknown }

async function request<T>(path: string, { token, method = 'GET', body }: RequestOptions = {}): Promise<T> {
  const headers: Record<string, string> = { 'Accept-Language': i18n.resolvedLanguage ?? 'gl' }
  if (token) headers.Authorization = `Bearer ${token}`
  if (body !== undefined) headers['Content-Type'] = 'application/json'

  const res = await fetch(`/api${path}`, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
  })
  if (res.status === 204) return undefined as T
  if (!res.ok) {
    let message = `HTTP ${res.status}`
    // the rate limiter answers in English only
    if (res.status === 429) throw new ApiError(429, i18n.t('common:rateLimited'))
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

/** GET …/indicators/export: the CSV comes with the token, so it is fetched and saved as a file. */
async function exportIndicators(token: string, slug: string, query: IndicatorsQuery, { kind, excel }: IndicatorsExport) {
  const params = indicatorsParams(query)
  params.set('kind', kind)
  if (excel) params.set('excel', '1')
  const res = await fetch(`/api/tenants/${encodeURIComponent(slug)}/admin/indicators/export?${params}`, {
    headers: { Authorization: `Bearer ${token}`, 'Accept-Language': i18n.resolvedLanguage ?? 'gl' },
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
  const res = await fetch(`/api/tenants/${encodeURIComponent(slug)}/admin/export${personal ? '?personal=1' : ''}`, {
    headers: { Authorization: `Bearer ${token}`, 'Accept-Language': i18n.resolvedLanguage ?? 'gl' },
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
  const res = await fetch(`/api/me/export?format=${format}`, {
    headers: { Authorization: `Bearer ${token}`, 'Accept-Language': i18n.resolvedLanguage ?? 'gl' },
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

/** POST /media: multipart upload, answered with { ref, url }. */
async function uploadImage(token: string, folder: MediaFolder, file: File): Promise<StoredImage> {
  const body = new FormData()
  body.append('folder', folder)
  body.append('file', file)
  const res = await fetch('/api/media', {
    method: 'POST',
    headers: { Authorization: `Bearer ${token}`, 'Accept-Language': i18n.resolvedLanguage ?? 'gl' },
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
  tenants: () => request<{ tenants: TenantCard[] }>('/tenants'),
  tags: () => request<TagCatalogue>('/tags'),
  branding: () => request<Branding>('/branding'),
  facilities: () => request<{ facilities: FacilityCard[] }>('/facilities'),
  day: (slug: string, date?: string) =>
    request<Day>(`/tenants/${encodeURIComponent(slug)}/day${date ? `?date=${date}` : ''}`),
  me: (token: string) => request<Me>('/me', { token }),
  updateProfile: (token: string, input: ProfileInput) => request<Me>('/me/profile', { token, method: 'PUT', body: input }),
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
  recommendations: (token: string) => request<{ recommendations: Recommendation[] }>('/me/recommendations', { token }),
  // "For you" on or off, also from the section itself
  setShowRecommendations: (token: string, show: boolean) => request<Me>('/me/recommendations', { token, method: 'PUT', body: { show } }),
  facilityPage: (slug: string, facility: string, date?: string) =>
    request<FacilityPage>(`/tenants/${encodeURIComponent(slug)}/facilities/${encodeURIComponent(facility)}${date ? `?date=${date}` : ''}`),
  // acceptRules: the resident ticked "I have read the rules"; the API refuses the booking otherwise
  book: (token: string, facilityId: string, startsAt: string, acceptRules: boolean) =>
    request<Booking>('/me/bookings', {
      token,
      method: 'POST',
      body: { facility_id: facilityId, starts_at: startsAt, accept_rules: acceptRules },
    }),
  // another free turn of the same facility, any day it can be booked
  changeBooking: (token: string, bookingId: string, startsAt: string) =>
    request<Booking>(`/me/bookings/${bookingId}`, { token, method: 'PATCH', body: { starts_at: startsAt } }),
  cancel: (token: string, bookingId: string) =>
    request<void>(`/me/bookings/${bookingId}`, { token, method: 'DELETE' }),

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
    facilities: (token: string, slug: string) =>
      request<{ facilities: AdminFacilitySummary[] }>(`/tenants/${encodeURIComponent(slug)}/admin/facilities`, { token }),
    facility: (token: string, slug: string, id: string) =>
      request<AdminFacility>(`/tenants/${encodeURIComponent(slug)}/admin/facilities/${id}`, { token }),
    // the schedule goes with it, so a new facility is set up in one go
    createFacility: (token: string, slug: string, input: FacilityInput & { schedule?: { windows: WeeklyWindow[] } }) =>
      request<AdminFacility>(`/tenants/${encodeURIComponent(slug)}/admin/facilities`, { token, method: 'POST', body: input }),
    updateFacility: (token: string, slug: string, id: string, input: FacilityInput) =>
      request<AdminFacility>(`/tenants/${encodeURIComponent(slug)}/admin/facilities/${id}`, { token, method: 'PUT', body: input }),
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
      request<{ closure: Closure; cancelled: number }>(`/tenants/${encodeURIComponent(slug)}/admin/facilities/${id}/closures`, {
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
    rotateCheckinCode: (token: string, slug: string, facilityId: string) =>
      request<CheckinSettings>(`/tenants/${encodeURIComponent(slug)}/admin/facilities/${facilityId}/checkin/code`, { token, method: 'POST' }),
    staffDay: (token: string, slug: string, facilityId: string, date: string) =>
      request<StaffDay>(`/tenants/${encodeURIComponent(slug)}/admin/facilities/${facilityId}/bookings?date=${date}`, { token }),
    staffBook: (token: string, slug: string, input: StaffBookingInput) =>
      request<StaffBooking>(`/tenants/${encodeURIComponent(slug)}/admin/bookings`, { token, method: 'POST', body: input }),
    // the resident is told if they have an account, with the reason when there is one
    staffCancel: (token: string, slug: string, bookingId: string, reason: string) =>
      request<void>(`/tenants/${encodeURIComponent(slug)}/admin/bookings/${bookingId}/cancel`, { token, method: 'POST', body: { reason } }),
    // staff mark that the person came in; undo only within a few minutes of a staff entry
    staffCheckIn: (token: string, slug: string, bookingId: string) =>
      request<StaffBooking>(`/tenants/${encodeURIComponent(slug)}/admin/bookings/${bookingId}/checkin`, { token, method: 'POST' }),
    undoStaffCheckIn: (token: string, slug: string, bookingId: string) =>
      request<StaffBooking>(`/tenants/${encodeURIComponent(slug)}/admin/bookings/${bookingId}/checkin`, { token, method: 'DELETE' }),
    indicators: (token: string, slug: string, query: IndicatorsQuery) =>
      request<Indicators>(`/tenants/${encodeURIComponent(slug)}/admin/indicators?${indicatorsParams(query)}`, { token }),
    exportIndicators,
    exportData: exportCouncilData,
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
