import { Sparkles } from 'lucide-react'
import { lazy, Suspense, useEffect, useRef, useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import ReactMarkdown, { type Components } from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Skeleton } from '@/components/ui/skeleton'
import type { AnalyticsAnswer } from '@/lib/api'

// the charts library loads only when an answer brings a chart
const AnswerChart = lazy(() => import('@/components/admin/AnswerChart').then((m) => ({ default: m.AnswerChart })))

const MAX_QUESTION = 1000

type Props = {
  ask: (question: string) => Promise<AnalyticsAnswer>
}

type Exchange = { question: string; answer?: AnalyticsAnswer; error?: string }

/**
 * «Talk to the data»: questions in the person's words, one after another; a language
 * model writes SQL on the read-only analytics views and answers each, with charts when
 * they help (each with its table). The SQL it ran is not shown (it reads like a
 * debugging tool): the API logs it, per person.
 * Answers are Markdown (bold, lists, tables…), never HTML, and without links or images:
 * the model reads data people wrote, so nothing it writes may send the reader, or the
 * data, anywhere (an image URL would be fetched on its own). Each question
 * is answered on its own (the model does not see the earlier ones).
 */
export function AnalyticsAssistant({ ask }: Props) {
  const { t } = useTranslation('admin')
  const [question, setQuestion] = useState('')
  const [busy, setBusy] = useState(false)
  const [exchanges, setExchanges] = useState<Exchange[]>([])
  const end = useRef<HTMLDivElement>(null)

  useEffect(() => {
    end.current?.scrollIntoView?.({ block: 'nearest', behavior: 'smooth' })
  }, [exchanges.length, busy])

  const submit = async (event: FormEvent) => {
    event.preventDefault()
    const asked = question.trim()
    if (!asked || busy) return
    setBusy(true)
    setQuestion('')
    setExchanges((list) => [...list, { question: asked }])
    let outcome: Exchange
    try {
      outcome = { question: asked, answer: await ask(asked) }
    } catch (e) {
      outcome = {
        question: asked,
        error: e instanceof Error ? e.message : String(e),
      }
    }
    setExchanges((list) => [...list.slice(0, -1), outcome])
    setBusy(false)
  }

  return (
    <div className="grid gap-6">
      <ol aria-label={t('analytics.title')} aria-live="polite" className="grid gap-6">
        {exchanges.length === 0 && <li className="text-muted-foreground">{t('analytics.empty')}</li>}
        {exchanges.map((exchange, i) => (
          <li key={i} className="grid gap-3">
            <div className="ml-auto max-w-[85%] rounded-2xl rounded-br-sm bg-primary px-4 py-2 text-primary-foreground">
              <span className="sr-only">{t('analytics.you')}: </span>
              <p className="whitespace-pre-wrap">{exchange.question}</p>
            </div>
            <div className="flex max-w-full gap-3">
              <span className="mt-1 grid size-8 shrink-0 place-items-center rounded-full bg-muted" aria-hidden>
                <Sparkles className="size-4 text-primary" />
              </span>
              <div className="grid min-w-0 flex-1 gap-3">
                <span className="sr-only">{t('analytics.assistant')}: </span>
                {!exchange.answer && !exchange.error && <p className="text-muted-foreground">{t('analytics.thinking')}</p>}
                {exchange.error && (
                  <Alert variant="destructive">
                    <AlertDescription>{exchange.error}</AlertDescription>
                  </Alert>
                )}
                {exchange.answer && <Answer answer={exchange.answer} />}
              </div>
            </div>
          </li>
        ))}
      </ol>
      <div ref={end} />
      <form onSubmit={(e) => void submit(e)} className="sticky bottom-0 grid gap-2 border-t bg-background pt-4 pb-4">
        <Label htmlFor="assistant-question">{t('analytics.question')}</Label>
        <Textarea
          id="assistant-question"
          rows={2}
          maxLength={MAX_QUESTION}
          placeholder={t('analytics.placeholder')}
          value={question}
          onChange={(e) => setQuestion(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && !e.shiftKey) void submit(e)
          }}
        />
        <div className="flex flex-wrap gap-2">
          <Button type="submit" disabled={busy || !question.trim()}>
            {busy ? t('analytics.thinking') : t('analytics.ask')}
          </Button>
          {exchanges.length > 0 && (
            <Button type="button" variant="outline" disabled={busy} onClick={() => setExchanges([])}>
              {t('analytics.clear')}
            </Button>
          )}
        </div>
      </form>
    </div>
  )
}

function Answer({ answer }: { answer: AnalyticsAnswer }) {
  return (
    <>
      {(answer.charts ?? []).map((chart, i) => (
        <Suspense key={i} fallback={<Skeleton className="h-64 w-full" />}>
          <AnswerChart chart={chart} />
        </Suspense>
      ))}
      <div className="grid gap-3 leading-relaxed">
        <ReactMarkdown remarkPlugins={[remarkGfm]} allowedElements={MARKDOWN_ELEMENTS} unwrapDisallowed components={MARKDOWN}>
          {answer.answer}
        </ReactMarkdown>
      </div>
    </>
  )
}

// what the answer may use; links and images are left out (their text stays), raw HTML is never read
const MARKDOWN_ELEMENTS = ['p', 'strong', 'em', 'del', 'ul', 'ol', 'li', 'code', 'pre', 'blockquote', 'h1', 'h2', 'h3', 'h4', 'hr', 'br', 'table', 'thead', 'tbody', 'tr', 'th', 'td']

const MARKDOWN: Components = {
  h1: ({ children }) => <p className="text-lg font-semibold">{children}</p>,
  h2: ({ children }) => <p className="text-lg font-semibold">{children}</p>,
  h3: ({ children }) => <p className="font-semibold">{children}</p>,
  h4: ({ children }) => <p className="font-semibold">{children}</p>,
  strong: ({ children }) => <strong className="font-semibold">{children}</strong>,
  ul: ({ children }) => <ul className="grid list-disc gap-1 pl-5">{children}</ul>,
  ol: ({ children }) => <ol className="grid list-decimal gap-1 pl-5">{children}</ol>,
  blockquote: ({ children }) => <blockquote className="border-l-2 pl-3 text-muted-foreground">{children}</blockquote>,
  code: ({ children }) => <code className="rounded bg-muted px-1 py-0.5 text-[0.9em]">{children}</code>,
  pre: ({ children }) => <pre className="overflow-x-auto rounded-md bg-muted p-2 text-xs [&_code]:bg-transparent [&_code]:p-0">{children}</pre>,
  hr: () => <hr className="border-border" />,
  table: ({ children }) => (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">{children}</table>
    </div>
  ),
  th: ({ children }) => (
    <th scope="col" className="py-1 pr-4 text-left font-medium">
      {children}
    </th>
  ),
  td: ({ children }) => <td className="num border-t py-1 pr-4">{children}</td>,
}
