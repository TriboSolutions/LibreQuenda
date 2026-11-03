import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Check, Copy } from 'lucide-react'
import { SettingsRow } from '@/components/SettingsRow'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import { useAsync } from '@/hooks/useAsync'
import { languages } from '@/i18n'
import { api } from '@/lib/api'

const ALL = '*'
const AUTO = 'auto'

/** The council settings' "Widget for your website": a live preview and the <iframe> code to paste. */
export function WidgetSettings({ slug, token }: { slug: string; token: string }) {
  const { t } = useTranslation('admin')
  const facilities = useAsync(() => api.council.facilities(token, slug), `widget-facilities:${slug}`)
  const [facility, setFacility] = useState(ALL)
  const [theme, setTheme] = useState(AUTO)
  const [lang, setLang] = useState(AUTO)
  const [height, setHeight] = useState(420)
  const [copied, setCopied] = useState(false)

  const query = new URLSearchParams()
  if (theme !== AUTO) query.set('theme', theme)
  if (lang !== AUTO) query.set('lang', lang)
  const path = `/${slug}/widget${facility === ALL ? '' : `/${facility}`}${query.size ? `?${query}` : ''}`
  const src = `${window.location.origin}${path}`
  const code = `<iframe src="${src}" title="${t('widget.iframeTitle')}" width="100%" height="${height}" style="border:0" loading="lazy"></iframe>`

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(code)
      setCopied(true)
      window.setTimeout(() => setCopied(false), 2000)
    } catch {
      // no clipboard (http, permissions): the code stays selectable in the box
    }
  }

  return (
    <SettingsRow id="sheet-widget" title={t('widget.title')} description={t('widget.help')}>
      <div className="grid content-start gap-4 rounded-lg border bg-card p-5">
        <div className="grid gap-3 sm:grid-cols-2">
          <div className="grid gap-1.5">
            <Label htmlFor="widget-facility">{t('widget.facility')}</Label>
            <Select value={facility} onValueChange={setFacility}>
              <SelectTrigger id="widget-facility" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={ALL}>{t('widget.allFacilities')}</SelectItem>
                {(facilities.data?.facilities ?? [])
                  .filter((f) => f.is_active)
                  .map((f) => (
                    <SelectItem key={f.id} value={f.slug}>
                      {f.name}
                    </SelectItem>
                  ))}
              </SelectContent>
            </Select>
          </div>
          <div className="grid gap-1.5">
            <Label htmlFor="widget-height">{t('widget.height')}</Label>
            <Input
              id="widget-height"
              type="number"
              min={200}
              max={2000}
              step={20}
              value={height}
              onChange={(e) => setHeight(Math.min(2000, Math.max(200, Number(e.target.value) || 420)))}
              className="num"
            />
          </div>
          <div className="grid gap-1.5">
            <Label htmlFor="widget-theme">{t('widget.theme')}</Label>
            <Select value={theme} onValueChange={setTheme}>
              <SelectTrigger id="widget-theme" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={AUTO}>{t('widget.themeAuto')}</SelectItem>
                <SelectItem value="light">{t('widget.themeLight')}</SelectItem>
                <SelectItem value="dark">{t('widget.themeDark')}</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="grid gap-1.5">
            <Label htmlFor="widget-lang">{t('widget.language')}</Label>
            <Select value={lang} onValueChange={setLang}>
              <SelectTrigger id="widget-lang" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={AUTO}>{t('widget.languageAuto')}</SelectItem>
                {languages.map((l) => (
                  <SelectItem key={l.code} value={l.code} lang={l.code}>
                    {l.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>
        <div className="grid gap-1.5">
          <Label htmlFor="widget-code">{t('widget.code')}</Label>
          <Textarea id="widget-code" readOnly rows={3} value={code} onFocus={(e) => e.currentTarget.select()} className="font-mono text-xs" />
          <div>
            <Button type="button" variant="outline" size="sm" onClick={() => void copy()}>
              {copied ? <Check aria-hidden /> : <Copy aria-hidden />}
              {copied ? t('widget.copied') : t('widget.copy')}
            </Button>
            <span role="status" className="sr-only">
              {copied ? t('widget.copied') : ''}
            </span>
          </div>
        </div>
        <div className="grid gap-1.5">
          <p className="text-sm font-medium">{t('widget.preview')}</p>
          <iframe src={path} title={t('widget.iframeTitle')} height={height} className="w-full rounded-md border" />
        </div>
      </div>
    </SettingsRow>
  )
}
