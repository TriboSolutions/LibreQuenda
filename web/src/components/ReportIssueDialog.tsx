import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { CircleCheck } from 'lucide-react'
import { ImagePicker } from '@/components/ImagePicker'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import { api, type IssueCategory, type StoredImage } from '@/lib/api'

const ISSUE_CATEGORIES: IssueCategory[] = ['equipment', 'lighting', 'cleaning', 'changing_rooms', 'accessibility', 'safety', 'other']
const MAX_DESCRIPTION = 1000

type Props = {
  facility: { id: string; name: string } | null // null: closed
  bookingId: string // reported from this booking
  onClose: () => void
}

/** A resident tells the council about a problem at a facility: it lands in the council's maintenance list. */
export function ReportIssueDialog({ facility, bookingId, onClose }: Props) {
  const { t } = useTranslation('facility')
  const auth = useAuth()
  const [category, setCategory] = useState<IssueCategory | ''>('')
  const [description, setDescription] = useState('')
  const [photo, setPhoto] = useState<StoredImage | null>(null)
  const [urgent, setUrgent] = useState(false)
  const [shareContact, setShareContact] = useState(false)
  const [busy, setBusy] = useState(false)
  const [sent, setSent] = useState(false)
  const [error, setError] = useState<string | null>(null)

  if (!facility) return <Dialog open={false} />

  const submit = async (event: FormEvent) => {
    event.preventDefault()
    const token = auth.user?.access_token
    if (!token) return
    if (!category) {
      setError(t('issue.chooseCategory'))
      return
    }
    setBusy(true)
    setError(null)
    try {
      await api.reportIssue(token, {
        facility_id: facility.id,
        booking_id: bookingId,
        category,
        description,
        photo_ref: photo?.ref,
        urgent,
        share_contact: shareContact,
      })
      setSent(true)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(false)
    }
  }

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{t('issue.title')}</DialogTitle>
          <DialogDescription>{t('issue.description', { facility: facility.name })}</DialogDescription>
        </DialogHeader>

        {sent ? (
          <div role="status" className="grid gap-4">
            <p className="flex items-start gap-2 rounded-md bg-pill-positive-bg px-3 py-2 text-pill-positive-ink">
              <CircleCheck className="mt-0.5 size-4 shrink-0" aria-hidden />
              {t('issue.sent')}
            </p>
            <DialogFooter>
              <Button onClick={onClose}>{t('issue.close')}</Button>
            </DialogFooter>
          </div>
        ) : (
          <form onSubmit={(e) => void submit(e)} className="grid gap-4">
            <div className="grid gap-2">
              <Label htmlFor="issue-category">{t('issue.category')}</Label>
              <Select value={category} onValueChange={(value) => setCategory(value as IssueCategory)}>
                <SelectTrigger id="issue-category" className="w-full">
                  <SelectValue placeholder={t('issue.chooseCategory')} />
                </SelectTrigger>
                <SelectContent>
                  {ISSUE_CATEGORIES.map((c) => (
                    <SelectItem key={c} value={c}>
                      {t(`issue.categories.${c}`)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="grid gap-2">
              <Label htmlFor="issue-description">{t('issue.what')}</Label>
              <Textarea
                id="issue-description"
                rows={4}
                required
                maxLength={MAX_DESCRIPTION}
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                aria-describedby="issue-description-help"
              />
              <p id="issue-description-help" className="text-xs text-muted-foreground">
                {t('issue.whatHelp')}
              </p>
            </div>
            <ImagePicker label={t('issue.photo')} folder="issues" value={photo} onChange={setPhoto} />
            <div className="flex items-start gap-2">
              <Checkbox id="issue-urgent" checked={urgent} onCheckedChange={(v) => setUrgent(v === true)} aria-describedby="issue-urgent-help" className="mt-0.5" />
              <div className="grid gap-0.5">
                <Label htmlFor="issue-urgent" className="font-normal">
                  {t('issue.urgent')}
                </Label>
                <p id="issue-urgent-help" className="text-xs text-muted-foreground">
                  {t('issue.urgentHelp')}
                </p>
              </div>
            </div>
            <div className="flex items-start gap-2">
              <Checkbox id="issue-contact" checked={shareContact} onCheckedChange={(v) => setShareContact(v === true)} className="mt-0.5" />
              <Label htmlFor="issue-contact" className="font-normal">
                {t('issue.shareContact')}
              </Label>
            </div>
            <div aria-live="polite" className="empty:hidden">
              {error && (
                <Alert variant="destructive">
                  <AlertDescription>{error}</AlertDescription>
                </Alert>
              )}
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={onClose}>
                {t('issue.cancel')}
              </Button>
              <Button type="submit" disabled={busy}>
                {busy ? t('issue.sending') : t('issue.send')}
              </Button>
            </DialogFooter>
          </form>
        )}
      </DialogContent>
    </Dialog>
  )
}
