import { ArrowLeft, ExternalLink, Info, Search } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { Link, useNavigate, useParams, useSearchParams } from 'react-router'
import { FacilityMap } from '@/components/FacilityMap'
import { ImagePicker } from '@/components/ImagePicker'
import { FacilityBookingRules } from '@/components/admin/FacilityBookingRules'
import { CheckinMethods } from '@/components/admin/CheckinMethods'
import { ClosuresEditor } from '@/components/admin/ClosuresEditor'
import { ScheduleEditor } from '@/components/admin/ScheduleEditor'
import { TagPicker } from '@/components/admin/TagPicker'
import { TopBar } from '@/components/TopBar'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { Checkbox } from '@/components/ui/checkbox'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { useAsync } from '@/hooks/useAsync'
import { useCouncilAdmin } from '@/hooks/useCouncilAdmin'
import { api, type AdminFacility, type FacilityInput, type GeocodeResult, type StoredImage, type WeeklyWindow, type Tenant } from '@/lib/api'
import { slugify } from '@/lib/slug'
import { usePageTitle } from '@/hooks/usePageTitle'
import { SiteFooter } from '@/components/SiteFooter'

const EMPTY: FacilityInput = {
  slug: '',
  name: '',
  description: '',
  tags: [],
  address: '',
  latitude: null,
  longitude: null,
  capacity: null,
  accessibility_info: '',
  contact_phone: '',
  contact_email: '',
  image_ref: '',
  is_active: true,
  slot_minutes: 60,
  min_age: null,
  usage_rules: '',
}

function toInput(f: AdminFacility): FacilityInput {
  const {
    id: _id,
    image_url: _url,
    booking_count: _count,
    bookings_to_cancel: _toCancel,
    schedule: _schedule,
    closures: _closures,
    timezone: _tz,
    council_rules: _council,
    custom_rules: _custom,
    rules: _rules,
    ...input
  } = f
  return input
}

/** /:slug/admin/facilities/:id ("new" to create): everything about one facility. */
export function FacilityEditorPage() {
  const { slug = '', id = 'new' } = useParams()
  const creating = id === 'new'
  const allowed = useCouncilAdmin(slug)
  const auth = useAuth()
  const token = auth.user?.access_token ?? ''
  const tenants = useAsync(() => api.tenants(), 'tenants')
  const existing = useAsync(!creating && allowed ? () => api.council.facility(token, slug, id) : null, `facility-admin:${slug}:${id}:${allowed}`)
  const { t } = useTranslation(['admin', 'common'])
  usePageTitle(creating ? t('admin:editor.newTitle') : existing.data?.name, tenants.data?.tenants.find((c) => c.slug === slug)?.name)

  if (!creating && !existing.data) {
    return (
      <div className="min-h-screen">
        <TopBar tenants={tenants.data?.tenants ?? []} tenantSlug={slug} />
        <main id="main" tabIndex={-1} className="mx-auto max-w-[1152px] px-4 pt-8 sm:px-6">
          {existing.error ? <p className="text-destructive">{existing.error.message}</p> : <Skeleton className="h-40 w-full" />}
          {!allowed && !auth.isLoading && (
            <Button asChild variant="outline" className="mt-4">
              <Link to={`/${slug}/admin`}>{t('admin:facilities.back')}</Link>
            </Button>
          )}
        </main>
        <SiteFooter />
      </div>
    )
  }
  return <Editor key={id} slug={slug} facility={creating ? null : existing.data} tenants={tenants.data?.tenants ?? []} token={token} />
}

function Editor({ slug, facility, tenants, token }: { slug: string; facility: AdminFacility | null; tenants: Tenant[]; token: string }) {
  const { t } = useTranslation(['admin', 'common'])
  const navigate = useNavigate()
  const creating = facility === null
  const council = tenants.find((c) => c.slug === slug) // its contact is what an empty facility contact falls back to
  const [form, setForm] = useState<FacilityInput>(facility ? toInput(facility) : EMPTY)
  const [image, setImage] = useState<StoredImage | null>(facility?.image_ref ? { ref: facility.image_ref, url: facility.image_url } : null)
  const [schedule, setSchedule] = useState<WeeklyWindow[]>(facility?.schedule ?? [])
  const [saving, setSaving] = useState(false)
  const [savingSchedule, setSavingSchedule] = useState(false)
  const [notice, setNotice] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [results, setResults] = useState<GeocodeResult[] | null>(null)
  const [searching, setSearching] = useState(false)
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [activeInfo, setActiveInfo] = useState(false)
  // the facility as last saved: switching active on or off saves only that, at once
  const [saved, setSaved] = useState<AdminFacility | null>(facility)
  const [confirmActive, setConfirmActive] = useState<boolean | null>(null) // the value waiting for confirmation
  const [savingActive, setSavingActive] = useState(false)
  // deactivating keeps the bookings; cancelling them is an explicit choice with a reason
  const [cancelBookings, setCancelBookings] = useState(false)
  const [cancelReason, setCancelReason] = useState('')
  // the tab lives in the address (?vista=entrada, ?vista=reserva), like the council admin's
  const [searchParams, setSearchParams] = useSearchParams()
  const vista = searchParams.get('vista')
  const view = vista === 'entrada' ? 'checkin' : vista === 'reserva' ? 'booking' : 'details'
  const changeView = (value: string) =>
    setSearchParams(value === 'checkin' ? { vista: 'entrada' } : value === 'booking' ? { vista: 'reserva' } : {}, { replace: true })

  const set = <K extends keyof FacilityInput>(field: K, value: FacilityInput[K]) => setForm((f) => ({ ...f, [field]: value }))
  const number = (value: string) => (value === '' ? 0 : Number(value))

  const run = async (action: () => Promise<void>) => {
    setError(null)
    setNotice(null)
    try {
      await action()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
      window.scrollTo({ top: 0, behavior: 'smooth' })
    }
  }

  const save = (event: React.FormEvent) => {
    event.preventDefault()
    setSaving(true)
    void run(async () => {
      const input = { ...form, image_ref: image?.ref ?? '' } // save the reference, never the URL
      const saved = creating
        ? await api.council.createFacility(token, slug, { ...input, schedule: { windows: schedule } })
        : await api.council.updateFacility(token, slug, facility.id, input)
      if (creating) {
        navigate(`/${slug}/admin/facilities/${saved.id}`, { replace: true })
      } else {
        setSaved(saved)
        setForm(toInput(saved))
        setNotice(t('admin:editor.saved'))
      }
    }).finally(() => setSaving(false))
  }

  // other unsaved edits in the form stay as they are, waiting for "Save"
  const saveActive = (active: boolean) => {
    if (!saved) return
    const cancel = !active && cancelBookings
    setSavingActive(true)
    void run(async () => {
      const { facility: updated, cancelled } = await api.council.setActive(token, slug, saved.id, active, cancel, cancelReason.trim())
      setSaved(updated)
      set('is_active', updated.is_active)
      setNotice(
        active
          ? t('admin:editor.activeConfirm.activated')
          : cancel
            ? t('admin:editor.activeConfirm.deactivatedCancelled', { count: cancelled })
            : t('admin:editor.activeConfirm.deactivatedKept', { count: saved.bookings_to_cancel }),
      )
    }).finally(() => {
      setSavingActive(false)
      setConfirmActive(null)
      setCancelBookings(false)
      setCancelReason('')
    })
  }

  const saveSchedule = (windows: WeeklyWindow[]) => {
    if (!facility) return
    setSavingSchedule(true)
    void run(async () => {
      const saved = await api.council.replaceSchedule(token, slug, facility.id, windows)
      setSchedule(saved.schedule)
      setNotice(t('admin:editor.saved'))
    }).finally(() => setSavingSchedule(false))
  }

  const search = () => {
    setSearching(true)
    void run(async () => {
      const found = (await api.council.geocode(token, form.address)).results
      setResults(found)
      if (found.length === 1) pick(found[0])
    }).finally(() => setSearching(false))
  }
  const pick = (place: GeocodeResult) => {
    set('latitude', Number(place.latitude.toFixed(6)))
    set('longitude', Number(place.longitude.toFixed(6)))
    setResults(null)
  }

  const remove = () =>
    void run(async () => {
      if (!facility) return
      await api.council.deleteFacility(token, slug, facility.id)
      navigate(`/${slug}/admin`, { replace: true })
    }).finally(() => setConfirmDelete(false))

  // the facility itself on the left; the photo and the map beside it
  const layout = (
    <div className="grid gap-6 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)] lg:items-start">
      <div className="grid gap-6">
        <form id="facility-form" onSubmit={save} className="grid gap-6">
          <section aria-labelledby="sec-details" className="grid gap-4 rounded-lg border bg-card p-5">
            <h2 id="sec-details" className="text-lg font-semibold">
              {t('admin:editor.details')}
            </h2>
            <div className="grid gap-4 md:grid-cols-2 md:items-start">
              <div className="grid gap-2">
                <Label htmlFor="f-name">{t('admin:editor.name')}</Label>
                <Input
                  id="f-name"
                  required
                  maxLength={255}
                  placeholder={t('admin:editor.namePlaceholder')}
                  value={form.name}
                  aria-describedby="f-url"
                  onChange={(e) => {
                    set('name', e.target.value)
                    // the address comes from the name when the facility is created, and then stays: links to it are out there
                    if (creating) set('slug', slugify(e.target.value))
                  }}
                />
                <p id="f-url" className="text-xs text-muted-foreground">
                  {t('admin:editor.slugHelp', { url: `${window.location.host}/${slug}/${form.slug || '…'}` })}
                </p>
              </div>
              <div className="grid gap-2">
                <Label htmlFor="f-capacity">
                  {t('admin:editor.capacity')} <span className="font-normal text-muted-foreground">({t('admin:form.optional')})</span>
                </Label>
                <Input
                  id="f-capacity"
                  type="number"
                  min={1}
                  value={form.capacity ?? ''}
                  onChange={(e) => set('capacity', e.target.value === '' ? null : Number(e.target.value))}
                  className="num"
                />
              </div>
              <div className="md:col-span-2">
                <TagPicker value={form.tags} onChange={(tags) => set('tags', tags)} />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="f-description">{t('admin:editor.description')}</Label>
                <Textarea id="f-description" rows={4} maxLength={5000} value={form.description} onChange={(e) => set('description', e.target.value)} />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="f-accessibility">
                  {t('admin:editor.accessibility')} <span className="font-normal text-muted-foreground">({t('admin:form.optional')})</span>
                </Label>
                <Textarea id="f-accessibility" rows={4} maxLength={2000} value={form.accessibility_info} onChange={(e) => set('accessibility_info', e.target.value)} />
              </div>
            </div>
            {/* its own contact, optional: each one left empty shows the council's on the facility's page */}
            <fieldset className="grid gap-4 border-t pt-4 md:grid-cols-2 md:items-start">
              <legend className="sr-only">{t('admin:editor.contact')}</legend>
              <p className="font-medium md:col-span-2">{t('admin:editor.contact')}</p>
              <div className="grid gap-2">
                <Label htmlFor="f-contact-phone">
                  {t('admin:editor.contactPhone')} <span className="font-normal text-muted-foreground">({t('admin:form.optional')})</span>
                </Label>
                <Input
                  id="f-contact-phone"
                  type="tel"
                  maxLength={50}
                  value={form.contact_phone}
                  onChange={(e) => set('contact_phone', e.target.value)}
                  className="num"
                  aria-describedby="f-contact-phone-help"
                />
                <p id="f-contact-phone-help" className="text-xs text-muted-foreground">
                  {council?.contact_phone ? t('admin:editor.contactInherits', { value: council.contact_phone }) : t('admin:editor.contactNone')}
                </p>
              </div>
              <div className="grid gap-2">
                <Label htmlFor="f-contact-email">
                  {t('admin:editor.contactEmail')} <span className="font-normal text-muted-foreground">({t('admin:form.optional')})</span>
                </Label>
                <Input
                  id="f-contact-email"
                  type="email"
                  maxLength={255}
                  value={form.contact_email}
                  onChange={(e) => set('contact_email', e.target.value)}
                  aria-describedby="f-contact-email-help"
                />
                <p id="f-contact-email-help" className="text-xs text-muted-foreground">
                  {council?.contact_email ? t('admin:editor.contactInherits', { value: council.contact_email }) : t('admin:editor.contactNone')}
                </p>
              </div>
            </fieldset>
          </section>

          <section aria-labelledby="sec-booking" className="grid gap-4 rounded-lg border bg-card p-5">
            <h2 id="sec-booking" className="text-lg font-semibold">
              {t('admin:editor.booking')}
            </h2>
            <div className="grid gap-4 sm:grid-cols-2">
              {(
                [['slot_minutes', 'admin:editor.slotMinutes', 15, 480, 5]] as const
              ).map(([field, label, min, max, step]) => (
                <div key={field} className="grid gap-2">
                  <Label htmlFor={`f-${field}`}>{t(label)}</Label>
                  <Input
                    id={`f-${field}`}
                    type="number"
                    required
                    min={min}
                    max={max}
                    step={step}
                    value={form[field]}
                    onChange={(e) => set(field, number(e.target.value))}
                    className="num"
                  />
                </div>
              ))}
            </div>
            <div className="grid gap-2">
              <Label htmlFor="f-rules">{t('admin:editor.usageRules')}</Label>
              <Textarea id="f-rules" rows={4} maxLength={5000} value={form.usage_rules} aria-describedby="f-rules-help" onChange={(e) => set('usage_rules', e.target.value)} />
              <p id="f-rules-help" className="text-xs text-muted-foreground">
                {t('admin:editor.usageRulesHelp')}
              </p>
            </div>
            <div className="grid gap-2 sm:max-w-xs">
              <Label htmlFor="f-min-age">
                {t('admin:editor.minAge')} <span className="font-normal text-muted-foreground">({t('admin:form.optional')})</span>
              </Label>
              <Input
                id="f-min-age"
                type="number"
                min={1}
                max={80}
                value={form.min_age ?? ''}
                aria-describedby="f-min-age-help"
                onChange={(e) => set('min_age', e.target.value === '' ? null : Number(e.target.value))}
                className="num"
              />
              <p id="f-min-age-help" className="text-xs text-muted-foreground">
                {t('admin:editor.minAgeHelp')}
              </p>
            </div>
            {/* creating: one button at the very end, after the weekly hours that are saved with it */}
            {!creating && (
              <div className="flex justify-end border-t pt-4">
                <Button type="submit" disabled={saving}>
                  {saving ? t('admin:form.saving') : t('admin:editor.save')}
                </Button>
              </div>
            )}
          </section>
        </form>

        <section aria-labelledby="sec-schedule" className="rounded-lg border bg-card p-5">
          <h2 id="sec-schedule" className="text-lg font-semibold">
            {t('admin:editor.schedule')}
          </h2>
          <p className="mt-1 text-muted-foreground">{creating ? t('admin:editor.scheduleWithCreate') : t('admin:editor.scheduleHelp')}</p>
          <div className="mt-3">
            {creating ? (
              <ScheduleEditor value={schedule} onChange={setSchedule} />
            ) : (
              <ScheduleEditor key={JSON.stringify(schedule)} value={schedule} saving={savingSchedule} onSave={saveSchedule} />
            )}
          </div>
        </section>

        {creating && (
          <div className="flex justify-end">
            <Button type="submit" form="facility-form" disabled={saving}>
              {saving ? t('admin:form.saving') : t('admin:editor.create')}
            </Button>
          </div>
        )}

        {!creating && saved && (
          <ClosuresEditor slug={slug} facilityId={saved.id} token={token} timeZone={saved.timezone} initial={saved.closures} />
        )}
      </div>

      <aside className="grid gap-6">
        <section aria-labelledby="sec-photo" className="grid gap-4 rounded-lg border bg-card p-5">
          <h2 id="sec-photo" className="text-lg font-semibold">
            {t('admin:editor.image')}
          </h2>
          <ImagePicker label={t('admin:editor.image')} hideLabel folder="facilities" shape="large" value={image} onChange={setImage} />
        </section>

        <section aria-labelledby="sec-location" className="grid gap-4 rounded-lg border bg-card p-5">
          <h2 id="sec-location" className="text-lg font-semibold">
            {t('admin:editor.location')}
          </h2>
          <div className="grid gap-2">
            <Label htmlFor="f-address">{t('admin:editor.address')}</Label>
            <div className="flex gap-2">
              <Input id="f-address" maxLength={500} value={form.address} onChange={(e) => set('address', e.target.value)} />
              <Button type="button" variant="outline" disabled={searching || form.address.trim().length < 3} onClick={search}>
                <Search />
                {searching ? t('admin:editor.searching') : t('admin:editor.search')}
              </Button>
            </div>
          </div>
          {results && (
            <ul className="grid rounded-md border">
              {results.length === 0 ? (
                <li className="px-3 py-2 text-muted-foreground">{t('admin:editor.noResults')}</li>
              ) : (
                results.map((place) => (
                  <li key={`${place.latitude},${place.longitude}`} className="border-t first:border-t-0">
                    <button type="button" className="min-h-11 w-full px-3 py-2 text-left hover:bg-accent" onClick={() => pick(place)}>
                      {place.label}
                    </button>
                  </li>
                ))
              )}
            </ul>
          )}
          <FacilityMap
            editable
            latitude={form.latitude}
            longitude={form.longitude}
            label={t('facility:mapLabel')}
            onChange={(lat, lng) => {
              set('latitude', Number(lat.toFixed(6)))
              set('longitude', Number(lng.toFixed(6)))
            }}
          />
          <div className="flex flex-wrap items-center justify-between gap-2">
            <p className="text-xs text-muted-foreground">{t('admin:editor.mapHelp')}</p>
            {form.latitude !== null && (
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={() => {
                  set('latitude', null)
                  set('longitude', null)
                }}
              >
                {t('admin:editor.removePin')}
              </Button>
            )}
          </div>
        </section>

        {!creating && (
          <section aria-labelledby="sec-delete" className="grid gap-3 rounded-lg border bg-card p-5">
            <h2 id="sec-delete" className="text-lg font-semibold">
              {t('admin:editor.deleteTitle')}
            </h2>
            {facility.booking_count > 0 ? (
              <p className="text-muted-foreground">{t('admin:manage.deleteBlocked', { count: facility.booking_count })}</p>
            ) : (
              <>
                <p className="text-muted-foreground">{t('admin:editor.deleteHelp')}</p>
                <div>
                  <Button variant="outline" className="text-destructive hover:text-destructive" onClick={() => setConfirmDelete(true)}>
                    {t('admin:manage.delete')}
                  </Button>
                </div>
              </>
            )}
          </section>
        )}
      </aside>
    </div>
  )

  return (
    <div className="min-h-screen">
      <TopBar tenants={tenants} tenantSlug={slug} />
      <main id="main" tabIndex={-1} className="mx-auto max-w-[1152px] px-4 pt-8 pb-16 sm:px-6">
        <Link to={`/${slug}/admin`} className="inline-flex items-center gap-1.5 text-muted-foreground hover:text-foreground">
          <ArrowLeft className="size-4" aria-hidden="true" />
          {t('admin:facilities.back')}
        </Link>
        <div className="mt-3 flex flex-wrap items-end justify-between gap-3">
          <h1 className="text-[28px] leading-9 font-bold tracking-[-0.02em] sm:text-[32px] sm:leading-10">
            {creating ? t('admin:editor.newTitle') : form.name || facility.name}
          </h1>
          <div className="flex flex-wrap items-center gap-x-5 gap-y-2">
            <div className="flex items-center gap-2">
              <Switch
                id="f-active"
                checked={form.is_active}
                disabled={savingActive}
                onCheckedChange={(checked) => (creating ? set('is_active', checked) : setConfirmActive(checked))}
              />
              <Label htmlFor="f-active">{form.is_active ? t('admin:councils.active') : t('admin:councils.inactive')}</Label>
              <Button type="button" variant="ghost" size="icon" onClick={() => setActiveInfo(true)} aria-label={t('admin:editor.activeInfo')}>
                <Info />
              </Button>
            </div>
            {!creating && saved?.is_active && (
              <Link target="_blank" rel="noopener noreferrer" to={`/${slug}/${facility.slug}`} className="inline-flex items-center gap-1.5 text-primary hover:underline">
                {t('admin:facilities.viewPage')}
                <ExternalLink className="size-3.5" aria-hidden="true" />
              </Link>
            )}
          </div>
        </div>

        <div aria-live="polite" className="mt-4 empty:hidden">
          {notice && (
            <Alert className="border-0 bg-pill-positive-bg text-pill-positive-ink">
              <AlertDescription className="text-pill-positive-ink">{notice}</AlertDescription>
            </Alert>
          )}
          {error && (
            <Alert variant="destructive">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}
        </div>

        {creating ? (
          <div className="mt-6">{layout}</div>
        ) : (
          <Tabs value={view} onValueChange={changeView} className="mt-6 gap-6">
            <TabsList>
              <TabsTrigger value="details">{t('admin:editor.tabDetails')}</TabsTrigger>
              <TabsTrigger value="booking">{t('admin:editor.tabBooking')}</TabsTrigger>
              <TabsTrigger value="checkin">{t('admin:editor.tabCheckin')}</TabsTrigger>
            </TabsList>
            <TabsContent value="details">{layout}</TabsContent>
            <TabsContent value="booking">
              {saved && <FacilityBookingRules slug={slug} facility={saved} token={token} onSaved={setSaved} />}
            </TabsContent>
            <TabsContent value="checkin">
              <CheckinMethods slug={slug} facilityId={facility.id} facilitySlug={facility.slug} token={token} />
            </TabsContent>
          </Tabs>
        )}

        {activeInfo && (
          <Dialog open onOpenChange={setActiveInfo}>
            <DialogContent className="sm:max-w-md">
              <DialogHeader>
                <DialogTitle>{t('admin:editor.activeInfoTitle')}</DialogTitle>
              </DialogHeader>
              <div className="grid gap-3">
                <p>{t('admin:editor.activeInfoActive')}</p>
                <p>{t('admin:editor.activeInfoInactive')}</p>
                <p className="text-muted-foreground">{t('admin:editor.activeInfoWhen')}</p>
              </div>
              <DialogFooter>
                <Button onClick={() => setActiveInfo(false)}>{t('admin:editor.understood')}</Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        )}

        {confirmActive !== null && saved && (
          <Dialog open onOpenChange={(open) => !open && !savingActive && setConfirmActive(null)}>
            <DialogContent className="sm:max-w-md">
              <DialogHeader>
                <DialogTitle>
                  {confirmActive
                    ? t('admin:editor.activeConfirm.activateTitle', { name: saved.name })
                    : t('admin:editor.activeConfirm.deactivateTitle', { name: saved.name })}
                </DialogTitle>
                <DialogDescription>
                  {confirmActive ? t('admin:editor.activeConfirm.activateBody') : t('admin:editor.activeConfirm.deactivateBody')}
                </DialogDescription>
              </DialogHeader>
              {!confirmActive &&
                (saved.bookings_to_cancel > 0 ? (
                  <div className="grid gap-3">
                    <p className="num">{t('admin:editor.activeConfirm.keepNote', { count: saved.bookings_to_cancel })}</p>
                    <div className="flex items-start gap-2">
                      <Checkbox id="cancel-bookings" checked={cancelBookings} onCheckedChange={(checked) => setCancelBookings(checked === true)} className="mt-0.5" />
                      <Label htmlFor="cancel-bookings" className="num leading-snug font-normal">
                        {t('admin:editor.activeConfirm.cancelOption', { count: saved.bookings_to_cancel })}
                      </Label>
                    </div>
                    {cancelBookings && (
                      <div className="grid gap-1.5">
                        <Label htmlFor="cancel-reason">{t('admin:editor.activeConfirm.reason')}</Label>
                        <Input
                          id="cancel-reason"
                          required
                          maxLength={255}
                          placeholder={t('admin:editor.activeConfirm.reasonPlaceholder')}
                          value={cancelReason}
                          onChange={(e) => setCancelReason(e.target.value)}
                        />
                        <p className="text-sm text-muted-foreground">{t('admin:editor.activeConfirm.reasonHint')}</p>
                      </div>
                    )}
                  </div>
                ) : (
                  <p className="text-muted-foreground">{t('admin:editor.activeConfirm.noBookings')}</p>
                ))}
              <DialogFooter>
                <Button variant="outline" onClick={() => setConfirmActive(null)} disabled={savingActive}>
                  {t('admin:form.cancel')}
                </Button>
                <Button
                  variant={confirmActive ? 'default' : 'destructive'}
                  onClick={() => saveActive(confirmActive)}
                  disabled={savingActive || (!confirmActive && cancelBookings && cancelReason.trim() === '')}
                >
                  {savingActive
                    ? t('admin:editor.activeConfirm.saving')
                    : confirmActive
                      ? t('admin:editor.activeConfirm.activate')
                      : t('admin:editor.activeConfirm.deactivate')}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        )}

        {confirmDelete && facility && (
          <Dialog open onOpenChange={(open) => !open && setConfirmDelete(false)}>
            <DialogContent className="sm:max-w-md">
              <DialogHeader>
                <DialogTitle>{t('admin:editor.deleteConfirmTitle', { name: facility.name })}</DialogTitle>
                <DialogDescription>{t('admin:editor.deleteConfirmBody')}</DialogDescription>
              </DialogHeader>
              <DialogFooter>
                <Button variant="outline" onClick={() => setConfirmDelete(false)}>
                  {t('admin:form.cancel')}
                </Button>
                <Button variant="destructive" onClick={remove}>
                  {t('admin:manage.deleteConfirm')}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        )}
      </main>
      <SiteFooter />
    </div>
  )
}
