import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { Search } from 'lucide-react'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { useAsync } from '@/hooks/useAsync'
import { api, type Resident, type ResidentSort } from '@/lib/api'
import { formatDateWithYear } from '@/lib/time'

/**
 * The council's verified residents (?vista=vecinos): who books and joins at its
 * residents-only facilities. Staff verify someone by the email of their account,
 * after checking at the desk (or against the census) that they live here.
 */
export function ResidentsPanel({ slug, onRequestsChanged }: { slug: string; onRequestsChanged?: () => void }) {
  const { t } = useTranslation('admin')
  const auth = useAuth()
  const token = auth.user?.access_token ?? ''
  const [email, setEmail] = useState('')
  const [search, setSearch] = useState('')
  const [query, setQuery] = useState('')
  const [sort, setSort] = useState<ResidentSort>('name')
  const [page, setPage] = useState(1)
  const [notice, setNotice] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [removing, setRemoving] = useState<Resident | null>(null)
  const requests = useAsync(() => api.council.residentRequests(token, slug), `resident-requests:${slug}`)
  const pendingRequests = requests.data?.requests ?? []
  const residents = useAsync(() => api.council.residents(token, slug, { q: query, sort, page }), `residents:${slug}:${query}:${sort}:${page}`)
  const list = residents.data?.residents ?? []
  const total = residents.data?.total ?? 0
  const pages = Math.max(1, Math.ceil(total / (residents.data?.page_size ?? 50)))

  const run = async (action: () => Promise<unknown>, message: string) => {
    setBusy(true)
    setError(null)
    setNotice(null)
    try {
      await action()
      setNotice(message)
      residents.reload()
      requests.reload()
      onRequestsChanged?.()
      return true
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
      return false
    } finally {
      setBusy(false)
    }
  }

  const add = async (event: FormEvent) => {
    event.preventDefault()
    const value = email.trim()
    if (await run(() => api.council.addResident(token, slug, value), t('residents.added', { email: value }))) setEmail('')
  }

  const find = (event: FormEvent) => {
    event.preventDefault()
    setPage(1)
    setQuery(search.trim())
  }

  return (
    <section aria-labelledby="residents-title" className="grid gap-4 rounded-lg border bg-card p-5">
      <div>
        <h2 id="residents-title" className="text-lg font-semibold">
          {t('residents.title')}
        </h2>
        <p className="mt-1 max-w-[70ch] text-muted-foreground">{t('residents.help')}</p>
      </div>

      {/* people who asked from the web or the app: staff check them against the census, then decide */}
      <section aria-labelledby="resident-requests-title" className="grid gap-3">
        <div>
          <h3 id="resident-requests-title" className="font-semibold">
            {t('residents.requests.title')}
            {pendingRequests.length > 0 && <span className="num ml-2 text-muted-foreground">({requests.data?.total ?? pendingRequests.length})</span>}
          </h3>
          <p className="mt-1 max-w-[70ch] text-sm text-muted-foreground">{t('residents.requests.help')}</p>
        </div>
        {requests.error ? (
          <p className="text-destructive">{requests.error.message}</p>
        ) : !requests.data ? (
          <Skeleton className="h-16 w-full" />
        ) : pendingRequests.length === 0 ? (
          <p className="rounded-md border border-dashed px-4 py-3 text-muted-foreground">{t('residents.requests.empty')}</p>
        ) : (
          <ul className="grid gap-2">
            {pendingRequests.map((q) => (
              <li key={q.id} className="grid gap-3 rounded-md border p-3 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center">
                <dl className="grid min-w-0 grid-cols-[auto_minmax(0,1fr)] gap-x-3 gap-y-0.5 text-sm">
                  <dt className="text-muted-foreground">{t('residents.requests.fullName')}</dt>
                  <dd className="font-medium break-words">{q.full_name}</dd>
                  <dt className="text-muted-foreground">{t('residents.requests.dni')}</dt>
                  <dd className="num font-medium">{q.dni}</dd>
                  <dt className="text-muted-foreground">{t('residents.requests.address')}</dt>
                  <dd className="break-words">
                    {q.street} · <span className="num">{q.postal_code}</span>
                  </dd>
                  <dt className="text-muted-foreground">{t('residents.requests.account')}</dt>
                  <dd className="truncate">{q.account_name ? `${q.account_name} · ${q.email}` : q.email}</dd>
                  <dt className="text-muted-foreground">{t('residents.requests.sent')}</dt>
                  <dd>
                    {t('residents.requests.sentOn', { date: formatDateWithYear(q.created_at.slice(0, 10)), expires: formatDateWithYear(q.expires_at.slice(0, 10)) })}
                  </dd>
                </dl>
                <div className="flex flex-wrap gap-2">
                  <Button
                    size="sm"
                    disabled={busy}
                    aria-label={`${t('residents.requests.verify')}: ${q.full_name}`}
                    onClick={() => void run(() => api.council.verifyResidentRequest(token, slug, q.id), t('residents.requests.verified', { name: q.full_name }))}
                  >
                    {t('residents.requests.verify')}
                  </Button>
                  <Button
                    size="sm"
                    variant="outline"
                    disabled={busy}
                    aria-label={`${t('residents.requests.discard')}: ${q.full_name}`}
                    onClick={() => void run(() => api.council.discardResidentRequest(token, slug, q.id), t('residents.requests.discarded', { name: q.full_name }))}
                  >
                    {t('residents.requests.discard')}
                  </Button>
                </div>
              </li>
            ))}
          </ul>
        )}
      </section>

      <form onSubmit={(e) => void add(e)} className="grid max-w-[560px] gap-2 border-t pt-4">
        <Label htmlFor="resident-email">{t('residents.addLabel')}</Label>
        <div className="flex gap-2">
          <Input id="resident-email" type="email" required autoComplete="off" value={email} onChange={(e) => setEmail(e.target.value)} aria-describedby="resident-email-help" />
          <Button type="submit" disabled={busy}>
            {t('residents.add')}
          </Button>
        </div>
        <p id="resident-email-help" className="text-sm text-muted-foreground">
          {t('residents.addHelp')}
        </p>
      </form>

      <div aria-live="polite">
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

      <div className="flex flex-wrap items-end gap-3 border-t pt-4">
        <form onSubmit={find} role="search" className="grid min-w-0 max-w-full gap-1.5">
          <Label htmlFor="residents-search">{t('residents.search')}</Label>
          <div className="flex gap-2">
            <Input id="residents-search" type="search" className="w-64 max-w-full" value={search} onChange={(e) => setSearch(e.target.value)} />
            <Button type="submit" variant="outline" size="icon" aria-label={t('residents.searchAction')} title={t('residents.searchAction')}>
              <Search />
            </Button>
          </div>
        </form>
        <div className="grid min-w-0 max-w-full gap-1.5">
          <Label htmlFor="residents-sort">{t('residents.sort')}</Label>
          <Select
            value={sort}
            onValueChange={(value) => {
              setPage(1)
              setSort(value as ResidentSort)
            }}
          >
            <SelectTrigger id="residents-sort" className="w-56 max-w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="name">{t('residents.sortName')}</SelectItem>
              <SelectItem value="recent">{t('residents.sortRecent')}</SelectItem>
            </SelectContent>
          </Select>
        </div>
        {residents.data && <p className="num pb-2 text-muted-foreground">{t('residents.count', { count: total })}</p>}
      </div>

      {residents.error ? (
        <p className="text-destructive">{residents.error.message}</p>
      ) : !residents.data ? (
        <Skeleton className="h-24 w-full" />
      ) : list.length === 0 ? (
        <p className="rounded-md border border-dashed px-4 py-3 text-muted-foreground">{query ? t('residents.noMatch') : t('residents.empty')}</p>
      ) : (
        <ul>
          {list.map((r) => (
            <li key={r.user_id} className="flex flex-wrap items-center justify-between gap-3 border-t py-2.5 first:border-t-0">
              <div className="min-w-0">
                <p className="truncate font-medium">{r.full_name || r.email}</p>
                <p className="truncate text-muted-foreground">
                  {r.full_name ? `${r.email} · ` : ''}
                  {r.verified_by
                    ? t('residents.verifiedBy', { date: formatDateWithYear(r.verified_at.slice(0, 10)), name: r.verified_by })
                    : t('residents.verifiedOn', { date: formatDateWithYear(r.verified_at.slice(0, 10)) })}
                </p>
              </div>
              <Button
                variant="ghost"
                size="sm"
                className="text-destructive hover:text-destructive"
                disabled={busy}
                aria-label={`${t('residents.remove')}: ${r.full_name || r.email}`}
                onClick={() => setRemoving(r)}
              >
                {t('residents.remove')}
              </Button>
            </li>
          ))}
        </ul>
      )}

      <Dialog open={removing !== null} onOpenChange={(open) => !open && setRemoving(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('residents.removeTitle', { name: removing?.full_name || removing?.email })}</DialogTitle>
            <DialogDescription>{t('residents.removeBody')}</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setRemoving(null)}>
              {t('residents.removeCancel')}
            </Button>
            <Button
              variant="destructive"
              disabled={busy}
              onClick={() => {
                const r = removing
                setRemoving(null)
                if (r) void run(() => api.council.removeResident(token, slug, r.user_id), t('residents.removed', { name: r.full_name || r.email }))
              }}
            >
              {t('residents.remove')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {pages > 1 && (
        <nav aria-label={t('residents.pages')} className="flex items-center justify-between gap-3 border-t pt-4">
          <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage(page - 1)}>
            {t('residents.previous')}
          </Button>
          <span className="num text-muted-foreground">{t('residents.page', { page, pages })}</span>
          <Button variant="outline" size="sm" disabled={page >= pages} onClick={() => setPage(page + 1)}>
            {t('residents.next')}
          </Button>
        </nav>
      )}
    </section>
  )
}
