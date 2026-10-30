import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { BookingRulesFields, BookingRulesSummary } from '@/components/admin/BookingRulesFields'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { api, type AdminFacility, type BookingRules } from '@/lib/api'

type Props = { slug: string; facility: AdminFacility; token: string; onSaved: (facility: AdminFacility) => void }

/** The facility's Reserva tab: the council's booking window, or its own, saved on their own. */
export function FacilityBookingRules({ slug, facility, token, onSaved }: Props) {
  const { t } = useTranslation('admin')
  const [custom, setCustom] = useState(facility.custom_rules)
  const [rules, setRules] = useState<BookingRules>(facility.rules)
  const [busy, setBusy] = useState(false)
  const [notice, setNotice] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  const save = async (event: FormEvent) => {
    event.preventDefault()
    setBusy(true)
    setError(null)
    setNotice(null)
    try {
      const updated = await api.council.saveFacilityRules(token, slug, facility.id, { custom_rules: custom, rules })
      onSaved(updated)
      setRules(updated.rules)
      setNotice(t('rules.facilitySaved'))
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(false)
    }
  }

  return (
    <section aria-labelledby="sec-window" className="grid gap-4 rounded-lg border bg-card p-5">
      <div>
        <h2 id="sec-window" className="text-lg font-semibold">
          {t('rules.councilTitle')}
        </h2>
        <p className="mt-1 max-w-[70ch] text-muted-foreground">{t('rules.facilityHelp')}</p>
      </div>
      <form onSubmit={(e) => void save(e)} className="grid gap-5">
        <div className="flex items-start gap-3">
          <Switch
            id="f-council-rules"
            checked={!custom}
            aria-describedby="f-council-rules-help"
            onCheckedChange={(inherit) => {
              setCustom(!inherit)
              // its own window starts from the council's, to change only what differs
              if (!inherit && !facility.custom_rules) setRules(facility.council_rules)
            }}
          />
          <div className="grid gap-1">
            <Label htmlFor="f-council-rules">{t('rules.useCouncil')}</Label>
            <p id="f-council-rules-help" className="text-sm text-muted-foreground">
              {t('rules.useCouncilHelp')}{' '}
              <Link to={`/${slug}/admin/axustes`} className="text-primary underline underline-offset-4">
                {t('rules.openCouncil')}
              </Link>
            </p>
          </div>
        </div>
        {custom ? <BookingRulesFields id="f-own-rules" value={rules} onChange={setRules} /> : <BookingRulesSummary rules={facility.council_rules} />}
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
        <div className="flex justify-end border-t pt-4">
          <Button type="submit" disabled={busy}>
            {busy ? t('rules.saving') : t('rules.save')}
          </Button>
        </div>
      </form>
    </section>
  )
}
