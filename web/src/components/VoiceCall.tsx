import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { matchPath, useLocation } from 'react-router'
import { Loader2, Mic, MicOff, Phone, PhoneOff } from 'lucide-react'
import type { PipecatClient } from '@pipecat-ai/client-js'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { loadVoiceConfig, startVoiceSession, VoiceError, voiceSocketUrl, type VoiceConfig } from '@/lib/voice'

// pages where a call makes sense: finding and booking, not administering or the door's reader
const PUBLIC_PAGES = ['/', '/reservas', '/:slug', '/:slug/:facility']
const STAFF_SEGMENTS = new Set(['admin', 'datos', 'lector', 'callback', 'perfil', 'invitacion', 'widget'])

type Phase = 'idle' | 'connecting' | 'live' | 'ended' | 'error'
type Problem = 'mic' | 'busy' | 'unavailable' | 'dropped'
type Line = { id: string; who: 'you' | 'agent'; text: string }

/**
 * «Reserva falando»: a call with the voice line, in a panel with live captions of both
 * sides (the WCAG captions of the call), mute and hang up. Shown only when the
 * installation runs the voice line; signed in, the call can book for the person.
 */
export function VoiceCall() {
  const { t } = useTranslation('voice')
  const { pathname } = useLocation()
  const [config, setConfig] = useState<VoiceConfig | null>(null)
  const [open, setOpen] = useState(false)

  useEffect(() => {
    void loadVoiceConfig().then(setConfig)
  }, [])

  const onPublicPage =
    PUBLIC_PAGES.some((p) => matchPath(p, pathname)) && !pathname.split('/').some((s) => STAFF_SEGMENTS.has(s))
  if (!config || (!onPublicPage && !open)) return null

  return (
    <>
      <Button
        onClick={() => setOpen(true)}
        className="fixed right-4 bottom-4 z-40 h-auto rounded-full p-3.5 shadow-lg hover:shadow-xl sm:right-6 sm:bottom-6 sm:px-5"
        aria-haspopup="dialog"
      >
        <Phone className="size-5" aria-hidden />
        <span className="sr-only sm:not-sr-only">{t('button')}</span>
      </Button>
      {open && <CallPanel config={config} onClose={() => setOpen(false)} />}
    </>
  )
}

function CallPanel({ config, onClose }: { config: VoiceConfig; onClose: () => void }) {
  const { t } = useTranslation('voice')
  const auth = useAuth()
  const [phase, setPhase] = useState<Phase>('idle')
  const [problem, setProblem] = useState<Problem | null>(null)
  const [lines, setLines] = useState<Line[]>([])
  const [muted, setMuted] = useState(false)
  const [activity, setActivity] = useState('')
  const client = useRef<PipecatClient | null>(null)
  const log = useRef<HTMLOListElement>(null)
  const signedIn = !!auth.user?.access_token

  useEffect(() => {
    log.current?.lastElementChild?.scrollIntoView({ block: 'nearest' })
  }, [lines])

  // closing the panel, or leaving the page, hangs up
  useEffect(() => () => void client.current?.disconnect(), [])

  const say = (who: Line['who'], id: string, text: string) =>
    setLines((all) => {
      const at = all.findIndex((l) => l.id === id)
      if (at === -1) return [...all, { id, who, text }]
      const next = [...all]
      next[at] = { id, who, text }
      return next
    })

  const fail = (why: Problem) => {
    setProblem(why)
    setPhase('error')
    void client.current?.disconnect()
    client.current = null
  }

  const call = async () => {
    setProblem(null)
    setLines([])
    setMuted(false)
    setPhase('connecting')
    try {
      const ticket = await startVoiceSession(auth.user?.access_token)
      // loaded on the first call only: the audio client is large and most visits never call
      const [{ PipecatClient }, { WebSocketTransport, ProtobufFrameSerializer }] = await Promise.all([
        import('@pipecat-ai/client-js'),
        import('@pipecat-ai/websocket-transport'),
      ])
      let turn = 0
      const pipecat = new PipecatClient({
        transport: new WebSocketTransport({
          serializer: new ProtobufFrameSerializer(),
          recorderSampleRate: 16000,
          playerSampleRate: 16000,
        }),
        enableMic: true,
        enableCam: false,
        callbacks: {
          onBotReady: () => setPhase('live'),
          onUserInput: (data) => {
            if (data.final && data.text.trim()) say('you', `you-${++turn}`, data.text.trim())
          },
          onBotOutput: (data) => {
            if (data.text.trim()) say('agent', `agent-${data.segment_id ?? ++turn}`, data.text.trim())
          },
          onLLMFunctionCallStarted: () => setActivity(t('looking')),
          onBotStartedSpeaking: () => setActivity(''),
          onDeviceError: () => fail('mic'),
          onDisconnected: () =>
            setPhase((now) => (now === 'live' || now === 'connecting' ? 'ended' : now)),
        },
      })
      client.current = pipecat
      await pipecat.connect({ wsUrl: voiceSocketUrl(), token: ticket })
    } catch (e) {
      if (e instanceof VoiceError) fail(e.message as Problem)
      else if (e instanceof DOMException && (e.name === 'NotAllowedError' || e.name === 'NotFoundError')) fail('mic')
      else fail('dropped')
    }
  }

  const hangUp = () => {
    void client.current?.disconnect()
    client.current = null
    setPhase('ended')
    setActivity('')
  }

  const toggleMute = () => {
    client.current?.enableMic(muted)
    setMuted(!muted)
  }

  const { voice, text } = config.thirdParties
  const minutes = Math.round(config.maxCallSeconds / 60)

  return (
    <Dialog open onOpenChange={(next) => !next && onClose()}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{t('title')}</DialogTitle>
          <DialogDescription>{t(signedIn ? 'descriptionSignedIn' : 'descriptionSignedOut')}</DialogDescription>
        </DialogHeader>

        {phase === 'idle' && (
          <div className="grid gap-2 text-sm text-muted-foreground">
            <p>{t('howItWorks', { minutes })}</p>
            <p>
              {voice.length > 0 ? t('privacy.voice', { names: voice.join(', ') }) : t('privacy.voiceLocal')}{' '}
              {text.length > 0 ? t('privacy.text', { names: text.join(', ') }) : t('privacy.textLocal')}{' '}
              {t('privacy.notKept')}
            </p>
          </div>
        )}

        {phase !== 'idle' && (
          <div className="grid gap-3">
            <p role="status" className="flex items-center gap-2 text-sm font-medium">
              {phase === 'connecting' && <Loader2 className="size-4 animate-spin" aria-hidden />}
              {phase === 'live' && <span className="size-2.5 rounded-full bg-primary" aria-hidden />}
              {t(`phase.${phase}`)}
              {phase === 'live' && muted && ` · ${t('muted')}`}
              {activity && ` · ${activity}`}
            </p>
            {problem && (
              <Alert variant="destructive">
                <AlertDescription>{t(`problem.${problem}`)}</AlertDescription>
              </Alert>
            )}
            <section aria-labelledby="voice-captions" className="grid gap-1">
              <h3 id="voice-captions" className="text-sm font-medium">
                {t('captions')}
              </h3>
              <ol
                ref={log}
                role="log"
                aria-live="polite"
                tabIndex={0}
                className="grid max-h-72 min-h-24 gap-2 overflow-y-auto rounded-md border p-3 text-sm focus-visible:ring-[3px] focus-visible:ring-ring focus-visible:outline-none"
              >
                {lines.length === 0 && <li className="text-muted-foreground">{t('captionsEmpty')}</li>}
                {lines.map((line) => (
                  <li key={line.id}>
                    <span className="font-semibold">{t(line.who)}: </span>
                    {line.text}
                  </li>
                ))}
              </ol>
            </section>
          </div>
        )}

        <DialogFooter>
          {phase === 'live' || phase === 'connecting' ? (
            <>
              <Button variant="outline" onClick={toggleMute} aria-pressed={muted} disabled={phase !== 'live'}>
                {muted ? <MicOff aria-hidden /> : <Mic aria-hidden />}
                {t('mute')}
              </Button>
              <Button variant="destructive" onClick={hangUp}>
                <PhoneOff aria-hidden />
                {t('hangUp')}
              </Button>
            </>
          ) : (
            <Button onClick={() => void call()}>
              <Phone aria-hidden />
              {phase === 'idle' ? t('call') : t('callAgain')}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
