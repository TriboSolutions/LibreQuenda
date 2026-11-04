import type { Page, Route } from '@playwright/test'
import type {
  BookingRating,
  Account,
  AdminFacility,
  AdminFacilitySummary,
  Booking,
  Branding,
  CouncilDetail,
  CouncilSmtp,
  CouncilSummary,
  Day,
  Facility,
  FacilityCard,
  FacilityDetails,
  FacilityPage,
  Indicators,
  Invite,
  Me,
  PeriodFigures,
  MyBookings,
  Notifications,
  PublicInvite,
  Reader,
  StaffBooking,
  StaffDay,
  Superadmin,
  TagCatalogue,
  Tenant,
  TenantCard,
  WeeklyWindow,
} from '../src/lib/api'

// A fake API and a fake signed-in session, so every screen renders without the docker stack.

const AUTHORITY = 'http://auth.test'
const CLIENT_ID = 'librequenda-web'
const TZ = 'Europe/Madrid'
const HOUR = 3_600_000

const now = new Date()
const iso = (d: Date) => d.toISOString()
const at = (hoursFromNow: number) => iso(new Date(Math.floor(now.getTime() / HOUR) * HOUR + hoursFromNow * HOUR))
const today = new Intl.DateTimeFormat('en-CA', { timeZone: TZ }).format(now)

const tenant: Tenant = { slug: 'lugo', name: 'Concello de Lugo', logo_url: '', contact_email: 'deportes@lugo.gal', contact_phone: '982 297 100' }
// without a contact: the statement sends residents to the council's online office
const tagCatalogue: TagCatalogue = {
  categories: [
    {
      id: 'c1',
      name: 'Deportes',
      position: 1,
      tags: [
        { id: 't1', slug: 'natacion', name: 'Natación', facility_count: 1 },
        { id: 't2', slug: 'padel', name: 'Pádel', facility_count: 2 },
      ],
    },
    {
      id: 'c2',
      name: 'Instalación',
      position: 2,
      tags: [
        { id: 't3', slug: 'cuberta', name: 'Cuberta', facility_count: 1 },
        { id: 't4', slug: 'piscina', name: 'Piscina', facility_count: 1 },
        { id: 't5', slug: 'pista', name: 'Pista', facility_count: 2 },
      ],
    },
  ],
}

const facilityCards: FacilityCard[] = [
  { id: 'f1', slug: 'padel-1', name: 'Pista de pádel 1', sport: 'Pádel', tags: ['padel', 'pista', 'cuberta'], image_url: '', address: 'Rúa da Raíña 1, Lugo', latitude: 43.0097, longitude: -7.5568, free_today: 4, council: tenant, joinable: ['event', 'public'] },
  { id: 'f3', slug: 'piscina', name: 'Piscina municipal', sport: 'Natación', tags: ['natacion', 'piscina'], image_url: '', address: '', latitude: null, longitude: null, free_today: 0, council: tenant, joinable: [] },
]

const tenants: TenantCard[] = [
  {
    ...tenant,
    stats: { facilities: 6, sports: ['Natación', 'Pádel', 'Tenis'], free_today: 42, bookings: 6179, residents: 20 },
  },
  { slug: 'ourense', name: 'Concello de Ourense', logo_url: '', contact_email: '', contact_phone: '', stats: { facilities: 5, sports: [], free_today: 0, bookings: 0, residents: 0 } },
]

function slots(): Facility['slots'] {
  // past, then free and booked turns, then one too far ahead and one closed
  const statuses = ['past', 'past', 'free', 'booked', 'free', 'free', 'booked', 'closed', 'free', 'too_far'] as const
  return statuses.map((status, i) => ({ starts_at: at(i - 2), ends_at: at(i - 1), status }))
}

const schedule: WeeklyWindow[] = [1, 2, 3, 4, 5].map((weekday) => ({ weekday, opens_at: '08:00', closes_at: '22:00' }))

function facility(id: string, name: string, slug: string): Facility {
  return {
    show_weather: false,
    id,
    slug,
    name,
    description: 'Pista cuberta con iluminación.',
    image_url: '',
    latitude: 43.0097,
    longitude: -7.5568,
    sport: 'Pádel',
    tags: ['padel', 'pista', 'cuberta'],
    address: 'Rúa da Raíña 1, Lugo',
    contact_phone: '', // the council's is shown instead
    contact_email: '',
    slot_minutes: 60,
    max_advance_hours: 7 * 24,
    min_advance_minutes: 30,
    min_cancel_hours: 2,
    max_active_bookings_per_user: 2,
    max_bookings_per_day: 1,
    max_turns_in_a_row: 0,
    max_bookings_per_week: 3,
    council_max_active_bookings: 4,
    weekly_opening: { weekday: 7, at: '20:00' },
    capacity: 4,
    min_age: 16,
    usage_rules: 'Calzado deportivo obrigatorio.\nRespecta o horario.',
    windows: [{ opens_at: '08:00', closes_at: '22:00' }],
    closures: [{ starts_at: at(5), ends_at: at(6), reason: 'Mantemento' }],
    slots: slots(),
  }
}

const facilities = [facility('f1', 'Pista de pádel 1', 'padel-1'), facility('f2', 'Pavillón municipal', 'pavillon')]

const day: Day = { tenant, timezone: TZ, date: today, now: iso(now), facilities }

// every busy level, so axe checks the colours of all three
const facilityPage: FacilityPage = {
  tenant,
  timezone: TZ,
  date: today,
  now: iso(now),
  // the council's event over a booked turn: listed above the turns and marked on them
  facility: {
    ...facilities[0],
    public_bookings: [
      { id: 'ev1', kind: 'event', title: 'Torneo de pádel de outono', starts_at: at(4), ends_at: at(5), description: 'Por parellas, calquera nivel.', capacity: 16, people: ['Uxía', 'Brais'] },
    ],
  },
  schedule,
  rating: { average: 4.3, count: 12 },
}

const details: FacilityDetails = {
  image_url: '',
  sport: 'Pádel',
  address: 'Rúa da Raíña 1, Lugo',
  latitude: 43.0097,
  longitude: -7.5568,
  accessibility_info: 'Acceso sen chanzos e vestiario adaptado.',
  slot_minutes: 60,
  min_cancel_hours: 2,
  min_age: 16,
  usage_rules: 'Calzado deportivo obrigatorio.',
}

function booking(id: string, hoursFromNow: number, extra: Partial<Booking> = {}): Booking {
  return {
    id,
    starts_at: at(hoursFromNow),
    ends_at: at(hoursFromNow + 1),
    status: 'confirmed',
    facility_id: 'f1',
    facility_name: 'Pista de pádel 1',
    facility_slug: 'padel-1',
    tenant_slug: 'lugo',
    tenant_name: 'Concello de Lugo',
    can_cancel: true,
    can_change: true,
    cancel_until: at(hoursFromNow - 2),
    checkin_code: 'LQ-123456',
    checked_in_at: null,
    kind: 'private',
    title: '',
    attendance: '',
    can_rate: false,
    role: 'owner',
    owner_name: '',
    capacity: 4,
    participant_count: 1,
    can_leave: false,
    people: [
      { name: 'Ana Pérez', is_owner: true, is_guest: false, is_me: true },
      { name: 'Xoán', is_owner: false, is_guest: true, is_me: false },
    ],
    facility: details,
    ...extra,
  }
}

const myBookings: MyBookings = {
  upcoming: [
    // starts in 6 minutes from the real clock (not the hour at() rounds to): check-in open, with every way in
    booking('b0', 0, {
      starts_at: iso(new Date(now.getTime() + 6 * 60_000)),
      ends_at: iso(new Date(now.getTime() + 66 * 60_000)),
      can_cancel: false,
      can_change: false,
      checkin_ways: ['entry_code', 'printed_qr', 'reader', 'staff'],
      checkin_radius_m: 120,
      staff_contacts: [{ name: 'Marta Rey', phone: '982 000 111', place: 'Conserxería, á entrada' }],
    }),
    booking('b1', 3),
    booking('b2', 26, { role: 'participant', owner_name: 'Uxía', can_cancel: false, can_change: false, can_leave: true }),
    // the council's event the person joined: no owner, the whole capacity for who joins
    booking('ev1', 50, {
      role: 'participant', kind: 'event', title: 'Torneo de pádel de outono', public_description: 'Por parellas, calquera nivel.',
      checkin_code: '', can_cancel: false, can_change: false, can_leave: true, capacity: 16, participant_count: 2,
      people: [
        { name: 'Ana Pérez', is_owner: false, is_guest: false, is_me: true },
        { name: 'Brais', is_owner: false, is_guest: false, is_me: false },
      ],
    }),
  ],
  history: [
    booking('b3', -30, { checkin_code: '', checked_in_at: at(-30), attendance: 'attended', can_rate: true, can_cancel: false, can_change: false }),
    booking('b4', -60, { status: 'cancelled', checkin_code: '', can_cancel: false, can_change: false }),
    booking('b5', -80, { checkin_code: '', attendance: 'no_show', can_cancel: false, can_change: false }),
  ],
}

// after a public booking: the facility already rated, the others not yet
const bookingRating: BookingRating = {
  booking_id: 'b3',
  facility_name: 'Pista de pádel 1',
  facility_slug: 'padel-1',
  tenant_slug: 'lugo',
  tenant_name: 'Concello de Lugo',
  starts_at: at(-30),
  ends_at: at(-29),
  is_public: true,
  can_rate: true,
  rated: false,
  rate_until: at(30 * 24 - 29),
  facility_stars: 0,
  people: [
    { user_id: 'u2', name: 'Uxía Lago', is_owner: true, stars: 0 },
    { user_id: 'u3', name: 'Brais Vidal', is_owner: false, stars: 0 },
  ],
}

const me: Me = {
  id: 'u1',
  email: 'ana@example.org',
  email_verified: true,
  full_name: 'Ana Pérez',
  is_superadmin: true,
  roles: [{ tenant_id: 't1', tenant_slug: 'lugo', tenant_name: 'Concello de Lugo', role: 'admin' }],
  birth_date: '1990-05-01',
  phone: '600000000',
  avatar_ref: '',
  avatar_url: '',
  preferred_language: 'gl',
  email_notifications: true,
  email_reminders: true,
  show_recommendations: true,
  rating: { average: 4.8, count: 5 },
  profile_complete: true,
  pending_invitations: [],
}

/** The same person before confirming their email: booking and joining with the account point to the profile. */
export const ME_UNVERIFIED: Me = { ...me, email_verified: false }

const account: Account = {
  given_name: 'Ana',
  family_name: 'Pérez',
  email: 'ana@example.org',
  email_verified: true,
  password_rules: { min_length: 8, uppercase: true, lowercase: true, number: true, symbol: false },
}

const notifications: Notifications = {
  unread_count: 3,
  notifications: [
    { name: '', id: 'n0', booking_id: 'b3', is_public: true, type: 'rating_requested', facility_name: 'Pista de pádel 1', starts_at: at(-30), ends_at: at(-29), reason: '', closed_from: null, closed_until: null, by_staff: false, tenant_name: 'Concello de Lugo', tenant_slug: 'lugo', facility_slug: 'padel-1', created_at: at(-29), read: false },
    { name: '', id: 'n1', type: 'booking_confirmed', facility_name: 'Pista de pádel 1', starts_at: at(3), ends_at: at(4), reason: '', closed_from: null, closed_until: null, by_staff: false, booking_id: '', is_public: false, tenant_name: 'Concello de Lugo', tenant_slug: 'lugo', facility_slug: '', created_at: at(-1), read: false },
    { name: '', id: 'n2', type: 'booking_cancelled_by_council', facility_name: 'Pavillón municipal', starts_at: at(30), ends_at: at(31), reason: 'Obras', closed_from: at(29), closed_until: at(40), by_staff: false, booking_id: '', is_public: false, tenant_name: 'Concello de Lugo', tenant_slug: 'lugo', facility_slug: '', created_at: at(-5), read: false },
    { name: '', id: 'n3', type: 'issue_solved', facility_name: 'Pista de pádel 1', starts_at: null, ends_at: null, reason: 'Rede nova colocada.', closed_from: null, closed_until: null, by_staff: false, booking_id: '', is_public: false, tenant_name: 'Concello de Lugo', tenant_slug: 'lugo', facility_slug: 'padel-1', created_at: at(-30), read: true },
  ],
}

const invite: Invite = {
  token: 'tok123',
  capacity: 4,
  spots_left: 2,
  state: 'open',
  participants: [{ id: 'p1', name: 'Xoán', is_guest: true, age: 20, joined_at: at(-2) }],
}

const publicInvite: PublicInvite = {
  facility_name: 'Pista de pádel 1',
  facility_slug: 'padel-1',
  tenant_slug: 'lugo',
  tenant_name: 'Concello de Lugo',
  timezone: TZ,
  owner_name: 'Uxía',
  starts_at: at(26),
  ends_at: at(27),
  capacity: 4,
  spots_left: 2,
  state: 'open',
  facility: details,
}

const adminFacilities: AdminFacilitySummary[] = [
  { id: 'f1', slug: 'padel-1', name: 'Pista de pádel 1', sport: 'Pádel', tags: ['padel', 'pista'], image_url: '', is_active: true, opening_window_count: 5, upcoming_booking_count: 3 },
  { id: 'f2', slug: 'pavillon', name: 'Pavillón municipal', sport: 'Baloncesto', tags: ['baloncesto', 'pavillon'], image_url: '', is_active: false, opening_window_count: 0, upcoming_booking_count: 0 },
]

// the council's booking window, which the facility inherits
const councilRules = {
  max_advance_hours: 7 * 24,
  min_advance_minutes: 30,
  min_cancel_hours: 2,
  max_active_bookings_per_user: 2,
  max_bookings_per_day: 1,
  max_turns_in_a_row: 2,
  max_bookings_per_week: 0,
  max_active_bookings_council: 4,
  weekly_opening: false,
  opens_weekday: 7,
  opens_minute: 20 * 60,
  no_show_penalty: true,
  no_show_limit: 3,
  no_show_window_days: 30,
  no_show_block_days: 7,
}

const adminFacility: AdminFacility = {
  show_weather: false,
  id: 'f1',
  slug: 'padel-1',
  name: 'Pista de pádel 1',
  description: 'Pista cuberta con iluminación.',
  tags: ['padel', 'pista', 'cuberta'],
  address: 'Rúa da Raíña 1, Lugo',
  latitude: 43.0097,
  longitude: -7.5568,
  capacity: 4,
  accessibility_info: 'Acceso sen chanzos.',
  contact_phone: '982 000 111',
  contact_email: '',
  image_ref: '',
  image_url: '',
  is_active: true,
  slot_minutes: 60,
  custom_rules: false,
  rules: councilRules,
  council_rules: councilRules,
  min_age: 16,
  usage_rules: 'Calzado deportivo obrigatorio.',
  booking_count: 12,
  bookings_to_cancel: 3,
  schedule,
  closures: [{ id: 'c1', starts_at: at(48), ends_at: at(72), reason: 'Mantemento' }],
  timezone: TZ,
}

const counts = {
  bookings: 120,
  confirmed: 100,
  cancelled_by_user: 15,
  cancelled_by_council: 5,
  cancellation_rate: 0.1667,
  booked_hours: 100,
  open_hours: 300,
  occupancy: 0.333,
  finished: 80,
  attended: 70,
  no_show_rate: 0.125,
}

function figures(daysBack: number, scale: number): PeriodFigures {
  const series = Array.from({ length: 14 }, (_, i) => {
    const d = new Date(now.getTime() - (daysBack + 13 - i) * 24 * HOUR)
    return { period: d.toISOString().slice(0, 10), confirmed: Math.round((5 + ((i * 7) % 9)) * scale), cancelled: i % 3 }
  })
  const weekdays = [1, 2, 3, 4, 5, 6, 7].map((weekday) => ({ weekday, booked_hours: 10 + weekday, open_hours: 42, occupancy: (10 + weekday) / 42 }))
  return {
    from: series[0].period,
    to: series[series.length - 1].period,
    totals: { ...counts, bookings: Math.round(counts.bookings * scale), active_users: 42, avg_party_size: 2.4, median_lead_hours: 30 },
    facilities: [
      { ...counts, id: 'f1', name: 'Pista de pádel 1', is_active: true, takes_checkin: true },
      { ...counts, id: 'f2', name: 'Pavillón municipal', is_active: false, takes_checkin: false, occupancy: null, no_show_rate: null },
    ],
    series,
    weekdays,
  }
}

function indicators(): Indicators {
  const busy_hours = [1, 2, 3, 4, 5, 6, 7].flatMap((weekday) =>
    Array.from({ length: 14 }, (_, h) => ({ weekday, hour: h + 8, bookings: (weekday * 3 + h * 5) % 11 })),
  )
  return {
    ...figures(0, 1),
    bucket: 'day',
    timezone: TZ,
    generated_at: iso(now),
    busy_hours,
    compare: 'previous',
    previous: figures(14, 0.8),
  }
}

const councils: CouncilSummary[] = [
  {
    slug: 'lugo',
    name: 'Concello de Lugo',
    timezone: TZ,
    contact_email: 'deportes@lugo.gal',
    contact_phone: '982000000',
    logo_url: '',
    is_active: true,
    created_at: at(-1000),
    facility_count: 2,
    admin_count: 1,
    pending_invitation_count: 1,
    booking_count: 120,
  },
]

const council: CouncilDetail = {
  slug: 'lugo',
  name: 'Concello de Lugo',
  timezone: TZ,
  contact_email: 'deportes@lugo.gal',
  contact_phone: '982000000',
  municipality_code: '27028',
  logo_ref: '',
  logo_url: '',
  is_active: true,
  booking_count: 120,
  admins: [{ user_id: 'u1', email: 'ana@example.org', full_name: 'Ana Pérez', role: 'admin', since: at(-500) }],
  invitations: [{ id: 'i1', email: 'novo@lugo.gal', role: 'admin', created_at: at(-10) }],
}

const superadmins: Superadmin[] = [
  { user_id: 'u1', email: 'ana@example.org', full_name: 'Ana Pérez', from_env: true, created_at: at(-1000) },
  { user_id: 'u2', email: 'brais@example.org', full_name: 'Brais Díaz', from_env: false, created_at: at(-100) },
]

const smtp: CouncilSmtp = {
  configured: true,
  unreadable: false,
  host: 'smtp.lugo.gal',
  port: 587,
  username: 'reservas',
  has_password: true,
  from: 'reservas@lugo.gal',
  updated_at: at(-50),
  env_relay: true,
}

// the Reservas tab: a turn under way whose entry staff can mark, a free turn, one walk-in
// booked by staff, and one online booking that came in with the reader
const staffBooking: StaffBooking = {
  id: '',
  starts_at: '',
  ends_at: '',
  booked_by_staff: false,
  is_event: false,
  has_account: false,
  name: '',
  email: '',
  phone: '',
  checked_in_at: null,
  checked_in_method: '',
  checked_in_by: '',
  checkin_state: '',
  undo_until: null,
  participant_count: 0,
  can_cancel: true,
  checkin_code: '',
}

const staffDay: StaffDay = {
  date: iso(now).slice(0, 10),
  timezone: TZ,
  facility_id: 'f1',
  min_age: 16,
  slot_minutes: 60,
  slots: [
    { starts_at: at(0), ends_at: at(1), status: 'booked', booking_id: 'sb3' },
    { starts_at: at(1), ends_at: at(2), status: 'free', booking_id: '' },
    { starts_at: at(2), ends_at: at(3), status: 'booked', booking_id: 'sb1' },
    { starts_at: at(3), ends_at: at(4), status: 'booked', booking_id: 'sb2' },
    { starts_at: at(4), ends_at: at(5), status: 'closed', booking_id: '' },
  ],
  bookings: [
    { ...staffBooking, id: 'sb3', starts_at: at(0), ends_at: at(1), has_account: true, name: 'Lucía Gómez', email: 'lucia@example.org', can_cancel: false, checkin_state: 'open' },
    { ...staffBooking, id: 'sb1', starts_at: at(2), ends_at: at(3), booked_by_staff: true, name: 'Carmen Souto', phone: '600123123', checkin_state: 'too_early', checkin_code: 'lq1:5c1d3b8e-3f6a-4b0e-9d6e-2a7c1f0b9e41' },
    { ...staffBooking, id: 'sb2', starts_at: at(3), ends_at: at(4), has_account: true, name: 'Ana Pérez', email: 'ana@example.org', checked_in_at: at(3), checked_in_method: 'reader', participant_count: 2 },
  ],
  staff_checkin: true,
}

/** Staff mark Lucía as in (POST), or take it back (DELETE). */
export const staffCheckinResponse = (undo: boolean): StaffBooking => {
  const lucia = staffDay.bookings[0]
  if (undo) return lucia
  const marked = new Date()
  return { ...lucia, checked_in_at: iso(marked), checked_in_method: 'staff', checked_in_by: 'Xoán Conserxe', checkin_state: '', undo_until: iso(new Date(marked.getTime() + 5 * 60_000)) }
}

const reader: Reader = {
  name: 'Entrada principal',
  facility_name: 'Pista de pádel 1',
  facility_slug: 'padel-1',
  tenant_name: 'Concello de Lugo',
  tenant_slug: 'lugo',
  timezone: TZ,
  now: iso(now),
}

// check-in from the resident's phone: both ways on, so the tab shows the QR and the code
const checkinSettings = { printed_qr: true, qr_token: '9a1f6c2e-3b4d-4e5f-8a7b-1c2d3e4f5a6b', nfc: false, nfc_token: '3c4d5e6f-7a8b-4c9d-8e1f-2a3b4c5d6e7f', entry_code: true, code: '482915', radius_m: 100, flash_booking: true, staff_confirm: true, qr_reader: true, staff_contacts: [{ name: 'Marta Rey', phone: '982 000 111', place: 'Conserxería, á entrada' }], has_location: true }
const entrance = {
  facility_name: 'Pista de pádel 1',
  facility_slug: 'padel-1',
  tenant_name: 'Concello de Lugo',
  tenant_slug: 'lugo',
  timezone: TZ,
  printed_qr: true,
  entry_code: true,
  radius_m: 100,
  flash_booking: true,
  booking: null,
}

// "For you": their usual Friday 19:00 free today, a facility they book often, one alike
const recommendations = {
  recommendations: [
    { kind: 'usual_today', facility: facilityCards[0], starts_at: at(3), times: 5, weekday: 5 },
    { kind: 'favourite', facility: facilityCards[1], starts_at: at(4), times: 3 },
    {
      kind: 'similar',
      facility: { ...facilityCards[0], id: 'f4', slug: 'padel-2', name: 'Pista de pádel 2' },
      starts_at: at(2),
      because: facilityCards[0].name,
      shared_tags: ['cuberta'],
    },
    {
      kind: 'similar',
      facility: { ...facilityCards[0], id: 'f5', slug: 'padel-3', name: 'Pista de pádel 3' },
      starts_at: at(5),
      because: facilityCards[0].name,
      shared_tags: ['padel'],
    },
    // someone's public booking with spots left, to join from «For you»
    {
      kind: 'public_booking',
      facility: facilityCards[0],
      starts_at: at(5),
      booking: { id: 'pb1', kind: 'public', starts_at: at(5), ends_at: at(6), spots_left: 2, owner_first_name: 'Uxía', owner_rating: { average: 4.5, count: 8 } },
    },
    // the council's event, joined the same way
    {
      kind: 'public_booking',
      facility: facilityCards[0],
      starts_at: at(50),
      booking: { id: 'ev1', kind: 'event', title: 'Torneo de pádel de outono', starts_at: at(50), ends_at: at(54), spots_left: 14, owner_first_name: '', owner_rating: null, description: 'Por parellas, calquera nivel.' },
    },
  ],
}

// the activity log: a council admin's change, an erased account's and the system's
const auditPage = {
  entries: [
    { id: 'a1', at: at(-2), actor_name: 'Ana Pérez', actor_role: 'council_admin', actor_deleted: false, tenant_slug: 'lugo', tenant_name: 'Concello de Lugo', action: 'facility.updated', target_type: 'facility', target_id: 'f1', target_name: 'Pista de pádel 1', target_exists: true, summary: 'slot_minutes: 90 → 60' },
    { id: 'a2', at: at(-5), actor_name: '', actor_role: 'council_admin', actor_deleted: true, tenant_slug: 'lugo', tenant_name: 'Concello de Lugo', action: 'closure.created', target_type: 'facility', target_id: 'f9', target_name: 'Piscina vella', target_exists: false, summary: 'bookings cancelled: 2' },
    { id: 'a3', at: at(-30), actor_name: '', actor_role: 'system', actor_deleted: false, tenant_slug: '', tenant_name: '', action: 'tag.created', target_type: 'tag', target_id: 't1', target_name: 'Pádel', target_exists: false, summary: '"Pádel"' },
  ],
  next: at(-31),
}

// the council's maintenance list: one new and urgent with the reporter's email, one being dealt with
const councilIssues = {
  issues: [
    {
      facility_image_url: '/test-facility.svg',
      id: 'i1',
      facility_id: 'f1',
      facility_name: 'Pista de pádel 1',
      category: 'equipment',
      description: 'A rede está rota polo medio.',
      photo_url: '',
      urgent: true,
      status: 'new',
      staff_note: '',
      created_at: at(-3),
      resolved_at: null,
      booking_starts_at: at(-5),
      reporter_name: 'Lucía',
      reporter_email: 'lucia@example.org',
      reporter_phone: '+34 600 123 456',
    },
    {
      id: 'i2',
      facility_id: 'f2',
      facility_name: 'Pavillón municipal',
      category: 'lighting',
      description: 'Dous focos non acenden.',
      photo_url: '',
      urgent: false,
      status: 'in_progress',
      staff_note: 'Avisado o electricista.',
      created_at: at(-50),
      resolved_at: null,
      booking_starts_at: null,
      reporter_name: 'Brais',
      reporter_email: '',
    },
  ],
}

// MeteoGalicia's forecast for an outdoor facility: a fine morning, showers in the afternoon
const weather = {
  source: 'MeteoGalicia',
  forecast: {
    date: today,
    sky: 'showers',
    temp_min: 9,
    temp_max: 17,
    periods: [
      { period: 'morning', sky: 'partly_cloudy', rain_chance: 10, wind: 'light', wind_from: 'sw', temperature: 15, warnings: [] },
      { period: 'afternoon', sky: 'showers', rain_chance: 80, wind: 'strong', wind_from: 'w', temperature: 17, warnings: ['rain', 'wind'] },
      { period: 'night', sky: 'cloudy', rain_chance: 40, wind: 'moderate', wind_from: 'w', temperature: 10, warnings: [] },
    ],
    warnings: ['rain', 'wind'],
  },
}

// the Eventos tab: one with people in it, one with the facility's capacity
const councilEvents = {
  timezone: TZ,
  events: [
    { id: 'ev1', facility_id: 'f1', facility_name: 'Pista de pádel 1', starts_at: at(50), ends_at: at(54), title: 'Torneo de pádel de outono', description: 'Por parellas, calquera nivel.', own_capacity: 16, capacity: 16, participant_count: 2, can_change: true, timezone: TZ },
    { id: 'ev2', facility_id: 'f1', facility_name: 'Pista de pádel 1', starts_at: at(100), ends_at: at(124), title: 'Xornada de portas abertas', description: '', own_capacity: null, capacity: 4, participant_count: 0, can_change: true, timezone: TZ },
  ],
}

// what is in the way of a new event: one resident's booking and a closure
export const eventInTheWay = {
  bookings: [{ id: 'b9', starts_at: at(26), ends_at: at(27), holder: 'Lucía Varela', is_event: false }],
  closures: [{ starts_at: at(28), ends_at: at(30), reason: 'Mantemento' }],
}

/** GET handlers by path (without /api), first match wins. */
const routes: [RegExp, () => unknown][] = [
  [/^\/tenants$/, () => ({ tenants })],
  [/^\/tags$/, () => tagCatalogue],
  [/^\/facilities$/, () => ({ facilities: facilityCards })],
  [/^\/tenants\/[^/]+\/day/, () => day],
  [/^\/tenants\/[^/]+\/facilities\/[^/]+\/entrance$/, () => entrance],
  [/^\/tenants\/[^/]+\/facilities\/[^/]+\/weather$/, () => weather],
  [/^\/tenants\/[^/]+\/facilities\/[^/?]+/, () => facilityPage],
  [/^\/tenants\/[^/]+\/admin\/facilities$/, () => ({ facilities: adminFacilities })],
  [/^\/tenants\/[^/]+\/admin\/facilities\/[^/]+\/bookings/, () => staffDay],
  [/^\/tenants\/[^/]+\/admin\/facilities\/[^/]+\/devices$/, () => ({ devices: [{ id: 'd1', name: 'Entrada principal', key_hint: 'ab12', last_seen_at: at(-1), created_at: at(-100) }] })],
  [/^\/tenants\/[^/]+\/admin\/facilities\/[^/]+\/checkin$/, () => checkinSettings],
  [/^\/tenants\/[^/]+\/admin\/booking-rules$/, () => councilRules],
  [/^\/tenants\/[^/]+\/admin\/events$/, () => councilEvents],
  [/^\/tenants\/[^/]+\/admin\/issues\/count$/, () => ({ new: 1 })],
  [/^\/tenants\/[^/]+\/admin\/issues/, () => councilIssues],
  [/^\/admin\/booking-policy$/, () => ({ one_council_per_account: true })],
  [/^\/tenants\/[^/]+\/admin\/facilities\/[^/]+$/, () => adminFacility],
  [/^\/tenants\/[^/]+\/admin\/indicators/, indicators],
  [/^\/tenants\/[^/]+\/admin\/smtp$/, () => smtp],
  [/^\/me$/, () => me],
  [/^\/me\/account$/, () => account],
  [/^\/me\/qr$/, () => ({ code: 'lq2:0b5e1a8e-6f1d-4c1e-9a55-3f6c2d7b9e10', expires_at: at(1), refresh_seconds: 60 })],
  [/^\/me\/calendar$/, () => ({ url: 'https://reservas.example.org/api/calendar/7c9e6679-7425-40de-944b-e07fc1f90ae7.ics' })],
  [/^\/me\/bookings$/, () => myBookings],
  [/^\/admin\/audit/, () => auditPage],
  [/^\/tenants\/[^/]+\/admin\/audit/, () => auditPage],
  [/^\/me\/recommendations$/, () => recommendations],
  [/^\/me\/bookings\/[^/]+\/invite$/, () => invite],
  [/^\/me\/bookings\/[^/]+\/rating$/, () => bookingRating],
  [/^\/me\/notifications$/, () => notifications],
  [/^\/invites\/[^/]+$/, () => publicInvite],
  [/^\/admin\/tenants$/, () => ({ tenants: councils })],
  [/^\/admin\/tenants\/[^/]+$/, () => council],
  [/^\/admin\/superadmins$/, () => ({ superadmins })],
  [/^\/reader$/, () => reader],
  [/^\/geocode/, () => ({ results: [] })],
]

function fulfil(route: Route, body: unknown, status = 200) {
  return route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) })
}

export const defaultBranding: Branding = { name: 'LibreQuenda', custom_name: '', logo_ref: '', logo_url: '', support_email: 'soporte@example.gal', support_phone: '+34 981 000 000' }

export type MockOptions = { signedIn?: boolean; theme?: 'light' | 'dark'; lang?: string; branding?: Branding; outdoor?: boolean; outdoorTagOnly?: boolean }

/** Mocks /config.json, the OIDC provider and /api, and seeds the session in localStorage. */
export async function mockApp(page: Page, { signedIn = true, theme = 'light', lang = 'gl', branding = defaultBranding, outdoor = false, outdoorTagOnly = false }: MockOptions = {}) {
  await page.route('**/config.json', (route) =>
    fulfil(route, { authority: AUTHORITY, clientId: CLIENT_ID, projectId: 'p1', scope: 'openid profile email' }),
  )
  await page.route(`${AUTHORITY}/**`, (route) =>
    fulfil(route, {
      issuer: AUTHORITY,
      authorization_endpoint: `${AUTHORITY}/oauth/v2/authorize`,
      token_endpoint: `${AUTHORITY}/oauth/v2/token`,
      userinfo_endpoint: `${AUTHORITY}/oidc/v1/userinfo`,
      end_session_endpoint: `${AUTHORITY}/oidc/v1/end_session`,
      jwks_uri: `${AUTHORITY}/oauth/v2/keys`,
    }),
  )
  await page.route('**/api/**', (route) => {
    const url = new URL(route.request().url())
    const path = url.pathname.replace(/^\/api/, '')
    if (route.request().method() === 'GET' && /^\/tenants\/[^/]+\/admin\/issues$/.test(path)) {
      const status = url.searchParams.get('status')
      const facility = url.searchParams.get('facility')
      return fulfil(route, { issues: councilIssues.issues.filter((issue) => (!status || issue.status === status) && (!facility || issue.facility_id === facility)) })
    }
    // flash booking with the entry code: the phone is at Lugo's padel court
    // «For you» is asked for by POST, with the home page's position when «near me» is on
    if (route.request().method() === 'POST' && path === '/me/recommendations') {
      return fulfil(route, recommendations)
    }
    if (route.request().method() === 'POST' && path === '/entrance/nearby') {
      return fulfil(route, {
        result: '',
        facilities: [
          {
            facility_name: 'Pista de pádel 1',
            facility_slug: 'padel-1',
            tenant_name: 'Concello de Lugo',
            tenant_slug: 'lugo',
            entry_code: true,
            printed_qr: true,
            radius_m: 150,
            staff_contacts: [{ name: 'Marta Rey', phone: '982 000 111', place: 'Conserxería, á entrada' }],
          },
        ],
      })
    }
    // the phone at the door without a booking: it is offered to book the free turn
    if (route.request().method() === 'POST' && /\/entrance$/.test(path)) {
      return fulfil(route, { granted: false, result: 'no_booking', can_book: true, booking: null })
    }
    // a new event: something in the way, so the form says what
    if (route.request().method() === 'POST' && /^\/tenants\/[^/]+\/admin\/events\/preview$/.test(path)) {
      return fulfil(route, eventInTheWay)
    }
    // staff mark the entry of a booking from the calendar, or take it back
    if (/^\/tenants\/[^/]+\/admin\/bookings\/[^/]+\/checkin$/.test(path)) {
      return fulfil(route, staffCheckinResponse(route.request().method() === 'DELETE'))
    }
    if (route.request().method() !== 'GET') return route.fulfill({ status: 204 })
    if (path === '/branding') return fulfil(route, branding)
    const handler = routes.find(([pattern]) => pattern.test(path))
    return handler ? fulfil(route, handler[1]()) : fulfil(route, { error: 'not found' }, 404)
  })
  // Enable the explicit preference, without changing any tags.
  if (outdoor || outdoorTagOnly) {
    await page.route(/\/api\/tenants\/[^/]+\/facilities\/[^/?]+(\?.*)?$/, (route) =>
      fulfil(route, { ...facilityPage, facility: { ...facilityPage.facility, show_weather: outdoor, tags: outdoorTagOnly ? ['aire-libre'] : facilityPage.facility.tags } }),
    )
  }
  // tiles for the map: an empty image keeps the test offline
  await page.route('https://*.tile.openstreetmap.org/**', (route) => route.fulfill({ status: 204 }))

  const user = signedIn
    ? JSON.stringify({
        id_token: 'id',
        access_token: 'access',
        token_type: 'Bearer',
        scope: 'openid profile email',
        expires_at: Math.floor(Date.now() / 1000) + 3600,
        profile: { sub: 'u1', iss: AUTHORITY, aud: CLIENT_ID, exp: 0, iat: 0, name: 'Ana Pérez', email: 'ana@example.org' },
      })
    : ''
  await page.addInitScript(
    ([key, user, theme, lang]) => {
      if (user) localStorage.setItem(key, user)
      localStorage.setItem('theme', theme)
      localStorage.setItem('lang', lang)
      localStorage.setItem('lq.readerKey', 'reader-key')
    },
    [`oidc.user:${AUTHORITY}:${CLIENT_ID}`, user, theme, lang],
  )
}
