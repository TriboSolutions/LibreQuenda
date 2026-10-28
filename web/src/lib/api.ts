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
  address: string
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
export type Tenant = { slug: string; name: string; logo_url: string; contact_email: string; contact_phone: string; contact_address: string; contact_person: string }

/** An uploaded image: forms save the ref (storage reference); screens only show the url. */
export type StoredImage = { ref: string; url: string }
export type MediaFolder = 'councils' | 'facilities' | 'users'

export type Day = {
  tenant: Tenant
  timezone: string
  date: string // YYYY-MM-DD in the council's time zone
  now: string
  facilities: Facility[]
}

export type WeeklyWindow = { weekday: number; opens_at: string; closes_at: string } // ISO weekday: 1 = Monday

/** How busy a turn usually is at that facility, from the bookings of the last weeks. */
export type BusyLevel = 'quiet' | 'moderate' | 'busy'

export type SuggestedSlot = { starts_at: string; ends_at: string; level: BusyLevel }

/** Another facility of the same sport with free turns that day. */
export type Alternative = {
  slug: string
  name: string
  address: string
  distance_km: number | null // null when either facility has no map pin
  free_slots: number
  slots: SuggestedSlot[] // its quietest free turns
}

export type Suggestions = {
  quiet: SuggestedSlot[] // free turns of the day, quietest first; empty without booking history
  alternatives: Alternative[] // nearest first
}

export type FacilityPage = {
  tenant: Tenant
  timezone: string
  date: string
  now: string
  facility: Facility
  schedule: WeeklyWindow[]
  suggestions: Suggestions
}

// --- council administration -----------------------------------------------

export type AdminFacilitySummary = {
  id: string
  slug: string
  name: string
  sport: string
  image_url: string
  is_active: boolean
  opening_window_count: number
  upcoming_booking_count: number
}

export type FacilityInput = {
  slug: string
  name: string
  description: string
  sport: string
  address: string
  latitude: number | null
  longitude: number | null
  capacity: number | null
  accessibility_info: string
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
}

// --- superadministrator area -------------------------------------------------

export type CouncilSummary = {
  slug: string
  name: string
  timezone: string
  contact_email: string
  contact_phone: string
  contact_address: string
  contact_person: string
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
  contact_address: string
  contact_person: string
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

export type FacilityIndicators = IndicatorCounts & { id: string; name: string; is_active: boolean; has_reader: boolean }

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
  contact_address: string
  contact_person: string
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

/** GET …/admin/export: the council's full data as a ZIP, fetched with the token and saved. */
async function exportCouncilData(token: string, slug: string) {
  const res = await fetch(`/api/tenants/${encodeURIComponent(slug)}/admin/export`, {
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
  tenants: () => request<{ tenants: Tenant[] }>('/tenants'),
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
    createFacility: (token: string, slug: string, input: FacilityInput) =>
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
  },
}
