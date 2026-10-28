import type { Page, Route } from '@playwright/test'
import type {
  Account,
  AdminFacility,
  AdminFacilitySummary,
  Booking,
  CouncilDetail,
  CouncilSmtp,
  CouncilSummary,
  Day,
  Facility,
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
  Superadmin,
  Tenant,
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
const tenants: Tenant[] = [tenant, { slug: 'ourense', name: 'Concello de Ourense', logo_url: '', contact_email: '', contact_phone: '' }]

function slots(): Facility['slots'] {
  // past, then free and booked turns, then one too far ahead and one closed
  const statuses = ['past', 'past', 'free', 'booked', 'free', 'free', 'booked', 'closed', 'free', 'too_far'] as const
  return statuses.map((status, i) => ({ starts_at: at(i - 2), ends_at: at(i - 1), status }))
}

const schedule: WeeklyWindow[] = [1, 2, 3, 4, 5].map((weekday) => ({ weekday, opens_at: '08:00', closes_at: '22:00' }))

function facility(id: string, name: string, slug: string): Facility {
  return {
    id,
    slug,
    name,
    description: 'Pista cuberta con iluminación.',
    image_url: '',
    latitude: 43.0097,
    longitude: -7.5568,
    sport: 'Pádel',
    address: 'Rúa da Raíña 1, Lugo',
    slot_minutes: 60,
    max_advance_days: 7,
    min_cancel_hours: 2,
    max_active_bookings_per_user: 2,
    min_age: 16,
    usage_rules: 'Calzado deportivo obrigatorio.\nRespecta o horario.',
    windows: [{ opens_at: '08:00', closes_at: '22:00' }],
    closures: [{ starts_at: at(5), ends_at: at(6), reason: 'Mantemento' }],
    slots: slots(),
  }
}

const facilities = [facility('f1', 'Pista de pádel 1', 'padel-1'), facility('f2', 'Pavillón municipal', 'pavillon')]

const day: Day = { tenant, timezone: TZ, date: today, now: iso(now), facilities }

const facilityPage: FacilityPage = { tenant, timezone: TZ, date: today, now: iso(now), facility: facilities[0], schedule }

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
    booking('b1', 3),
    booking('b2', 26, { role: 'participant', owner_name: 'Uxía', can_cancel: false, can_change: false, can_leave: true }),
  ],
  history: [
    booking('b3', -30, { checkin_code: '', checked_in_at: at(-30), can_cancel: false, can_change: false }),
    booking('b4', -60, { status: 'cancelled', checkin_code: '', can_cancel: false, can_change: false }),
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
  email_reminders: true,
  profile_complete: true,
  pending_invitations: [],
}

const account: Account = {
  given_name: 'Ana',
  family_name: 'Pérez',
  email: 'ana@example.org',
  email_verified: true,
  password_rules: { min_length: 8, uppercase: true, lowercase: true, number: true, symbol: false },
}

const notifications: Notifications = {
  unread_count: 2,
  notifications: [
    { id: 'n1', type: 'booking_confirmed', facility_name: 'Pista de pádel 1', starts_at: at(3), ends_at: at(4), reason: '', closed_from: null, closed_until: null, tenant_name: 'Concello de Lugo', tenant_slug: 'lugo', created_at: at(-1), read: false },
    { id: 'n2', type: 'booking_cancelled_by_council', facility_name: 'Pavillón municipal', starts_at: at(30), ends_at: at(31), reason: 'Obras', closed_from: at(29), closed_until: at(40), tenant_name: 'Concello de Lugo', tenant_slug: 'lugo', created_at: at(-5), read: false },
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
  { id: 'f1', slug: 'padel-1', name: 'Pista de pádel 1', sport: 'Pádel', image_url: '', is_active: true, opening_window_count: 5, upcoming_booking_count: 3 },
  { id: 'f2', slug: 'pavillon', name: 'Pavillón municipal', sport: 'Baloncesto', image_url: '', is_active: false, opening_window_count: 0, upcoming_booking_count: 0 },
]

const adminFacility: AdminFacility = {
  id: 'f1',
  slug: 'padel-1',
  name: 'Pista de pádel 1',
  description: 'Pista cuberta con iluminación.',
  sport: 'Pádel',
  address: 'Rúa da Raíña 1, Lugo',
  latitude: 43.0097,
  longitude: -7.5568,
  capacity: 4,
  accessibility_info: 'Acceso sen chanzos.',
  image_ref: '',
  image_url: '',
  is_active: true,
  slot_minutes: 60,
  max_advance_days: 7,
  min_cancel_hours: 2,
  max_active_bookings_per_user: 2,
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
      { ...counts, id: 'f1', name: 'Pista de pádel 1', is_active: true, has_reader: true },
      { ...counts, id: 'f2', name: 'Pavillón municipal', is_active: false, has_reader: false, occupancy: null, no_show_rate: null },
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

const reader: Reader = {
  name: 'Entrada principal',
  facility_name: 'Pista de pádel 1',
  facility_slug: 'padel-1',
  tenant_name: 'Concello de Lugo',
  tenant_slug: 'lugo',
  timezone: TZ,
  now: iso(now),
}

/** GET handlers by path (without /api), first match wins. */
const routes: [RegExp, () => unknown][] = [
  [/^\/tenants$/, () => ({ tenants })],
  [/^\/tenants\/[^/]+\/day/, () => day],
  [/^\/tenants\/[^/]+\/facilities\/[^/?]+/, () => facilityPage],
  [/^\/tenants\/[^/]+\/admin\/facilities$/, () => ({ facilities: adminFacilities })],
  [/^\/tenants\/[^/]+\/admin\/facilities\/[^/]+\/devices$/, () => ({ devices: [{ id: 'd1', name: 'Entrada principal', key_hint: 'ab12', last_seen_at: at(-1), created_at: at(-100) }] })],
  [/^\/tenants\/[^/]+\/admin\/facilities\/[^/]+$/, () => adminFacility],
  [/^\/tenants\/[^/]+\/admin\/indicators/, indicators],
  [/^\/tenants\/[^/]+\/admin\/smtp$/, () => smtp],
  [/^\/me$/, () => me],
  [/^\/me\/account$/, () => account],
  [/^\/me\/bookings$/, () => myBookings],
  [/^\/me\/bookings\/[^/]+\/invite$/, () => invite],
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

export type MockOptions = { signedIn?: boolean; theme?: 'light' | 'dark'; lang?: string }

/** Mocks /config.json, the OIDC provider and /api, and seeds the session in localStorage. */
export async function mockApp(page: Page, { signedIn = true, theme = 'light', lang = 'gl' }: MockOptions = {}) {
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
    if (route.request().method() !== 'GET') return route.fulfill({ status: 204 })
    const handler = routes.find(([pattern]) => pattern.test(path))
    return handler ? fulfil(route, handler[1]()) : fulfil(route, { error: 'not found' }, 404)
  })
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
