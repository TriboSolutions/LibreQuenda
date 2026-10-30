import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { BookingRulesFields } from '@/components/admin/BookingRulesFields'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { useAsync } from '@/hooks/useAsync'
import { api, type BookingRules } from '@/lib/api'

/** The council's booking window, in the council sheet: every facility without its own inherits it. */
export function BookingRulesSettings({ slug, token }: { slug: string; token: string }) {
  const { t } = useTranslation('admin')
  const saved = useAsync(() => api.council.bookingRules(token, slug), `booking-rules:${slug}`)
  const [form, setForm] = useState<BookingRules | null>(null)
  const [busy, setBusy] = useState(false)
  const [notice, setNotice] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  // the form starts from each load of the saved window
  const [loaded, setLoaded] = useState<BookingRules | null>(null)
  if (saved.data && saved.data !== loaded) {
    setLoaded(saved.data)
    setForm(saved.data)
  }

  const save = async (event: FormEvent) => {
    event.preventDefault()
    if (!form) return
    setBusy(true)
    setError(null)
    setNotice(null)
    try {
      await api.council.saveBookingRules(token, slug, form)
      saved.reload()
      setNotice(t('rules.saved'))
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(false)
    }
  }

  return (
    <section aria-labelledby="sheet-rules" className="grid content-start gap-3 rounded-lg border bg-card p-5">
      <div>
        <h2 id="sheet-rules" className="text-lg font-semibold">
          {t('rules.councilTitle')}
        </h2>
        <p className="text-sm text-muted-foreground">{t('rules.councilHelp')}</p>
      </div>
      {saved.error ? (
        <p className="text-destructive">{saved.error.message}</p>
      ) : !form ? (
        <Skeleton className="h-40 w-full" />
      ) : (
        <form onSubmit={(e) => void save(e)} className="grid gap-4">
          <BookingRulesFields id="council-rules" value={form} onChange={setForm} council />
          <div aria-live="polite" className="empty:hidden">
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
          <div>
            <Button type="submit" disabled={busy}>
              {busy ? t('rules.saving') : t('rules.save')}
            </Button>
          </div>
        </form>
      )}
    </section>
  )
}
