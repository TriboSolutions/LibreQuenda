import { createContext, useContext, useEffect, useRef, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { matchPath, useLocation } from 'react-router'
import { ChevronDown, Loader2, Mic, MicOff, Phone, PhoneOff } from 'lucide-react'
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

type VoiceLine = { available: boolean; open: () => void }
const VoiceContext = createContext<VoiceLine>({ available: false, open: () => {} })

/**
 * «Reserva falando»: a call with the voice line, in a panel with live captions of both
 * sides (the WCAG captions of the call, folded until opened), mute and hang up. Available only when the
 * installation runs the voice line, on the public pages; signed in, the call can book
 * for the person. Opened from the top bar (VoiceButton) or a facility's turns (VoiceLink).
 */
export function VoiceProvider({ children }: { children: ReactNode }) {
  const { pathname } = useLocation()
  const [config, setConfig] = useState<VoiceConfig | null>(null)
  const [open, setOpen] = useState(false)

  useEffect(() => {
    void loadVoiceConfig().then(setConfig)
  }, [])

  const onPublicPage =
    PUBLIC_PAGES.some((p) => matchPath(p, pathname)) && !pathname.split('/').some((s) => STAFF_SEGMENTS.has(s))

  return (
    <VoiceContext.Provider value={{ available: !!config && onPublicPage, open: () => setOpen(true) }}>
      {children}
      {config && open && <CallPanel config={config} onClose={() => setOpen(false)} />}
    </VoiceContext.Provider>
  )
}

/** The top bar's phone icon. */
export function VoiceButton() {
  const { t } = useTranslation('voice')
  const { available, open } = useContext(VoiceContext)
  if (!available) return null
  return (
    <Button variant="ghost" size="icon" onClick={open} aria-haspopup="dialog" aria-label={t('button')} title={t('button')}>
      <Phone />
    </Button>
  )
}

/** A quiet line under a facility's turns: «Prefires falar? Reserva por voz». */
export function VoiceLink() {
  const { t } = useTranslation('voice')
  const { available, open } = useContext(VoiceContext)
  if (!available) return null
  return (
    <p className="text-sm text-muted-foreground">
      {t('linkLead')}{' '}
      <button
        type="button"
        onClick={open}
        aria-haspopup="dialog"
        className="inline-flex items-center gap-1 rounded-sm font-medium text-primary underline-offset-4 hover:underline focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none"
      >
        <Phone className="size-3.5" aria-hidden />
        {t('link')}
      </button>
    </p>
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
  // the conversation in writing, folded until the person opens it
  const [showCaptions, setShowCaptions] = useState(false)
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
      // the agent books with this token until the call ends: a fresh one, if it would expire before
      let token = auth.user?.access_token
      if (token && (auth.user?.expires_in ?? 0) < config.maxCallSeconds) {
        token = (await auth.signinSilent().catch(() => null))?.access_token ?? token
      }
      const ticket = await startVoiceSession(token)
      // loaded on the first call only: the audio client is large and most visits never call
      const [{ PipecatClient }, { WebSocketTransport, ProtobufFrameSerializer }] = await Promise.all([
        import('@pipecat-ai/client-js'),
        import('@pipecat-ai/websocket-transport'),
      ])
      let turn = 0
      const pipecat = new PipecatClient({
        transport: keepPlayingAfterInterruptions(
          new WebSocketTransport({
            serializer: new ProtobufFrameSerializer(),
            recorderSampleRate: 16000,
            playerSampleRate: 16000,
          }),
        ),
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
          <DialogDescription>{t(signedIn ? 'descriptionSignedIn' : config.guestBooking ? 'descriptionGuest' : 'descriptionSignedOut')}</DialogDescription>
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
                <button
                  type="button"
                  onClick={() => setShowCaptions((shown) => !shown)}
                  aria-expanded={showCaptions}
                  aria-controls="voice-captions-log"
                  className="inline-flex items-center gap-1 rounded-sm text-muted-foreground hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none"
                >
                  <ChevronDown className={`size-4 transition-transform ${showCaptions ? 'rotate-180' : ''}`} aria-hidden />
                  {t('captions')}
                </button>
              </h3>
              <ol
                id="voice-captions-log"
                hidden={!showCaptions}
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

type Player = { interrupt: () => Promise<unknown>; interruptedTrackIds: Record<string, boolean> }

/**
 * @pipecat-ai/websocket-transport 1.8.2 plays all of the assistant's audio on the track "default" and,
 * on the first interruption, marks that track as interrupted for good: everything the assistant says
 * afterwards is dropped (the captions keep coming). Clear the mark once the interruption has flushed.
 * Fixed in 1.8.3, too new yet for pnpm's minimum release age: drop this when upgrading.
 */
function keepPlayingAfterInterruptions<T extends object>(transport: T): T {
  const player = (transport as unknown as { _mediaManager?: { _wavStreamPlayer?: Player } })._mediaManager?._wavStreamPlayer
  if (player) {
    const interrupt = player.interrupt.bind(player)
    player.interrupt = async () => {
      const offset = await interrupt()
      delete player.interruptedTrackIds.default
      return offset
    }
  }
  return transport
}
