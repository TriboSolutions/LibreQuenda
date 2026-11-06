import { Sparkles } from 'lucide-react'
import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import type { AnalyticsAnswer, AnalyticsStep } from '@/lib/api'

const MAX_QUESTION = 1000
const SHOWN_ROWS = 50

type Props = {
  ask: (question: string) => Promise<AnalyticsAnswer>
  scope: string // whose data it looks at, said under the title
}

/**
 * «Ask the data»: a question in the person's words; a language model writes SQL on
 * the read-only analytics views and answers. Every query it ran shows below the
 * answer with what it returned, so each figure can be checked. The answer is plain
 * text: nothing the model writes is read as markup.
 */
export function AnalyticsAssistant({ ask, scope }: Props) {
  const { t } = useTranslation('admin')
  const [question, setQuestion] = useState('')
  const [busy, setBusy] = useState(false)
  const [answer, setAnswer] = useState<AnalyticsAnswer | null>(null)
  const [error, setError] = useState<string | null>(null)

  const submit = async (event: FormEvent) => {
    event.preventDefault()
    if (!question.trim()) return
    setBusy(true)
    setError(null)
    setAnswer(null)
    try {
      setAnswer(await ask(question.trim()))
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(false)
    }
  }

  return (
    <section aria-labelledby="assistant-title" className="grid gap-4 rounded-lg border bg-card p-5">
      <div className="grid gap-1">
        <h2 id="assistant-title" className="flex items-center gap-2 text-lg font-semibold">
          <Sparkles className="size-5 text-primary" aria-hidden />
          {t('analytics.title')}
        </h2>
        <p className="text-sm text-muted-foreground">{t('analytics.help', { scope })}</p>
      </div>
      <form onSubmit={(e) => void submit(e)} className="grid gap-2">
        <Label htmlFor="assistant-question">{t('analytics.question')}</Label>
        <Textarea
          id="assistant-question"
          rows={2}
          maxLength={MAX_QUESTION}
          placeholder={t('analytics.placeholder')}
          value={question}
          onChange={(e) => setQuestion(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) void submit(e)
          }}
        />
        <div>
          <Button type="submit" disabled={busy || !question.trim()}>
            {busy ? t('analytics.thinking') : t('analytics.ask')}
          </Button>
        </div>
      </form>
      <div aria-live="polite" className="grid gap-4 empty:hidden">
        {error && (
          <Alert variant="destructive">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}
        {answer && (
          <>
            <p className="whitespace-pre-wrap">{answer.answer}</p>
            {answer.steps.length > 0 && (
              <details className="rounded-md border p-3">
                <summary className="cursor-pointer text-sm font-medium">{t('analytics.queries', { count: answer.steps.length })}</summary>
                <ol className="mt-3 grid gap-4">
                  {answer.steps.map((step, i) => (
                    <StepResult key={i} step={step} />
                  ))}
                </ol>
              </details>
            )}
          </>
        )}
      </div>
    </section>
  )
}

function StepResult({ step }: { step: AnalyticsStep }) {
  const { t } = useTranslation('admin')
  const result = step.result
  return (
    <li className="grid min-w-0 gap-2">
      <pre className="overflow-x-auto rounded-md bg-muted p-2 text-xs">
        <code>{step.sql}</code>
      </pre>
      {step.error && <p className="text-sm text-destructive">{step.error}</p>}
      {result && (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-xs text-muted-foreground">
                {result.columns.map((c) => (
                  <th key={c} scope="col" className="py-1 pr-4 font-normal">
                    {c}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {result.rows.slice(0, SHOWN_ROWS).map((row, r) => (
                <tr key={r} className="border-t">
                  {row.map((value, c) => (
                    <td key={c} className="num py-1 pr-4">
                      {value === null ? '—' : String(value)}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
          {(result.rows.length > SHOWN_ROWS || result.truncated) && (
            <p className="mt-1 text-xs text-muted-foreground">{t('analytics.moreRows', { shown: Math.min(SHOWN_ROWS, result.rows.length) })}</p>
          )}
        </div>
      )}
    </li>
  )
}
