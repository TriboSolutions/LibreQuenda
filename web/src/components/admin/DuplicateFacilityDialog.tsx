import { Copy } from 'lucide-react'
import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { api, type AdminFacilitySummary } from '@/lib/api'
import { slugify } from '@/lib/slug'

type Props = { slug: string; token: string; facilities: AdminFacilitySummary[] }

/**
 * Beside «New facility»: a copy of one of the council's facilities, with every setting
 * (hours, booking window, tags, check-in ways), under a name and address of its own.
 * It is created switched off and opens in the editor, to change what differs.
 */
export function DuplicateFacilityDialog({ slug, token, facilities }: Props) {
  const { t } = useTranslation('admin')
  const navigate = useNavigate()
  const [open, setOpen] = useState(false)
  const [source, setSource] = useState('')
  const [name, setName] = useState('')
  const [address, setAddress] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const pick = (id: string) => {
    setSource(id)
    const original = facilities.find((f) => f.id === id)
    if (!original) return
    const copy = t('duplicate.copyName', { name: original.name })
    setName(copy)
    setAddress(slugify(copy))
  }

  const reset = (next: boolean) => {
    setOpen(next)
    if (!next) {
      setSource('')
      setName('')
      setAddress('')
      setError(null)
    }
  }

  const submit = async (event: FormEvent) => {
    event.preventDefault()
    if (!source) return
    setSaving(true)
    setError(null)
    try {
      const copy = await api.council.duplicateFacility(token, slug, source, { name: name.trim(), slug: address.trim() })
      navigate(`/${slug}/admin/facilities/${copy.id}`, { state: { duplicated: true } })
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
      setSaving(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={reset}>
      <DialogTrigger asChild>
        <Button variant="outline">
          <Copy />
          {t('duplicate.open')}
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-md">
        <form onSubmit={submit} className="grid gap-4">
          <DialogHeader>
            <DialogTitle>{t('duplicate.title')}</DialogTitle>
            <DialogDescription>{t('duplicate.help')}</DialogDescription>
          </DialogHeader>
          <div className="grid gap-2">
            <Label htmlFor="duplicate-source">{t('duplicate.source')}</Label>
            <Select value={source} onValueChange={pick}>
              <SelectTrigger id="duplicate-source" className="w-full">
                <SelectValue placeholder={t('duplicate.sourcePlaceholder')} />
              </SelectTrigger>
              <SelectContent>
                {facilities.map((f) => (
                  <SelectItem key={f.id} value={f.id}>
                    {f.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="grid gap-2">
            <Label htmlFor="duplicate-name">{t('duplicate.name')}</Label>
            <Input
              id="duplicate-name"
              required
              maxLength={255}
              disabled={!source}
              value={name}
              onChange={(e) => {
                setName(e.target.value)
                setAddress(slugify(e.target.value))
              }}
            />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="duplicate-slug">{t('duplicate.slug')}</Label>
            <Input
              id="duplicate-slug"
              required
              maxLength={64}
              pattern="[a-z0-9]+(-[a-z0-9]+)*"
              disabled={!source}
              value={address}
              onChange={(e) => setAddress(e.target.value)}
              aria-describedby="duplicate-slug-help"
            />
            <p id="duplicate-slug-help" className="text-xs text-muted-foreground">
              {t('duplicate.slugHelp', { url: `${window.location.host}/${slug}/${address || '…'}` })}
            </p>
          </div>
          <div aria-live="polite" className="empty:hidden">
            {error && (
              <Alert variant="destructive">
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => reset(false)}>
              {t('duplicate.cancel')}
            </Button>
            <Button type="submit" disabled={!source || saving}>
              {saving ? t('duplicate.saving') : t('duplicate.submit')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
