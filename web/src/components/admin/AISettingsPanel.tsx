import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { SettingsRow } from '@/components/SettingsRow'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import { useAsync } from '@/hooks/useAsync'
import { api, type AISettings, type AISettingsInput } from '@/lib/api'

/** Where each kind of API usually lives, as a hint in the address field. */
const PLACEHOLDER = { openai: 'https://api.openai.com/v1', anthropic: 'https://api.anthropic.com' }

/**
 * The language model behind the councils' «ask the data» (the indicators): any
 * OpenAI-compatible API (a local one too) or Anthropic's. The key is saved sealed
 * and never shown again: left empty, the saved one stays.
 */
export function AISettingsPanel() {
  const { t } = useTranslation('admin')
  const auth = useAuth()
  const token = auth.user?.access_token ?? ''
  const saved = useAsync(() => api.admin.aiSettings(token), 'ai-settings')
  return (
    <SettingsRow id="ai-settings-title" title={t('aiSettings.title')} description={t('aiSettings.help')}>
      <div className="grid content-start gap-4 rounded-lg border bg-card p-5">
        {saved.data ? <AISettingsForm saved={saved.data} token={token} onSaved={saved.reload} /> : <Skeleton className="h-40 w-full" />}
      </div>
    </SettingsRow>
  )
}

function AISettingsForm({ saved, token, onSaved }: { saved: AISettings; token: string; onSaved: () => void }) {
  const { t } = useTranslation('admin')
  const [form, setForm] = useState<AISettingsInput>({ enabled: saved.enabled, provider: saved.provider, base_url: saved.base_url, model: saved.model, api_key: '' })
  const [busy, setBusy] = useState(false)
  const [status, setStatus] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  const set = <K extends keyof AISettingsInput>(key: K, value: AISettingsInput[K]) => setForm((f) => ({ ...f, [key]: value }))

  const save = async (event: FormEvent) => {
    event.preventDefault()
    setBusy(true)
    setStatus(null)
    setError(null)
    try {
      await api.admin.saveAISettings(token, form)
      setForm({ ...form, api_key: '' })
      onSaved()
      setStatus(t('aiSettings.saved'))
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(false)
    }
  }

  return (
    <form onSubmit={(e) => void save(e)} className="grid gap-4">
      <div className="flex items-start gap-3">
        <Switch id="ai-enabled" checked={form.enabled} onCheckedChange={(on) => set('enabled', on)} aria-describedby="ai-enabled-help" />
        <div className="grid gap-1">
          <Label htmlFor="ai-enabled">{t('aiSettings.enabled')}</Label>
          <p id="ai-enabled-help" className="text-sm text-muted-foreground">
            {t('aiSettings.enabledHelp')}
          </p>
        </div>
      </div>
      <div className="grid gap-1.5">
        <Label htmlFor="ai-provider">{t('aiSettings.provider')}</Label>
        <Select value={form.provider} onValueChange={(value) => set('provider', value as AISettingsInput['provider'])}>
          <SelectTrigger id="ai-provider" className="w-full sm:w-80">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="openai">{t('aiSettings.providerOpenai')}</SelectItem>
            <SelectItem value="anthropic">{t('aiSettings.providerAnthropic')}</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div className="grid gap-1.5">
        <Label htmlFor="ai-url">{t('aiSettings.baseUrl')}</Label>
        <Input
          id="ai-url"
          type="url"
          required
          maxLength={255}
          placeholder={PLACEHOLDER[form.provider]}
          value={form.base_url}
          onChange={(e) => set('base_url', e.target.value)}
          aria-describedby="ai-url-help"
        />
        <p id="ai-url-help" className="text-xs text-muted-foreground">
          {t('aiSettings.baseUrlHelp')}
        </p>
      </div>
      <div className="grid gap-1.5">
        <Label htmlFor="ai-model">{t('aiSettings.model')}</Label>
        <Input id="ai-model" required maxLength={100} value={form.model} onChange={(e) => set('model', e.target.value)} />
      </div>
      <div className="grid gap-1.5">
        <Label htmlFor="ai-key">{t('aiSettings.apiKey')}</Label>
        <Input
          id="ai-key"
          type="password"
          autoComplete="off"
          maxLength={500}
          placeholder={saved.has_key ? '••••••••••••••••' : ''}
          value={form.api_key}
          onChange={(e) => set('api_key', e.target.value)}
          aria-describedby="ai-key-help"
        />
        <p id="ai-key-help" className="text-xs text-muted-foreground">
          {saved.has_key && `${t('aiSettings.apiKeyKept')} `}
          {t('aiSettings.apiKeyHelp')}
        </p>
      </div>
      <Alert>
        <AlertDescription>{t('aiSettings.privacy')}</AlertDescription>
      </Alert>
      <div aria-live="polite" className="empty:hidden">
        {error && (
          <Alert variant="destructive">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}
        {status && <p className="text-sm text-muted-foreground">{status}</p>}
      </div>
      <div>
        <Button type="submit" disabled={busy}>
          {busy ? t('aiSettings.saving') : t('aiSettings.save')}
        </Button>
      </div>
    </form>
  )
}
