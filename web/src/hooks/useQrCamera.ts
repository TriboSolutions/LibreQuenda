import { useEffect, useRef, useState } from 'react'
import jsQR from 'jsqr'

export type CameraState = 'starting' | 'running' | 'denied' | 'missing' | 'insecure' | 'failed'

const SCAN_EVERY_MS = 150
const SAME_CODE_PAUSE_MS = 6000 // the same QR held in front of the camera is read once

/**
 * The camera, reading QR codes: the back one on a phone, whatever there is on a
 * kiosk. onCode gets each code read; nothing is read while paused() is true.
 * getUserMedia only exists in a secure context (https or localhost).
 */
export function useQrCamera(onCode: (code: string) => void, paused: () => boolean) {
  const video = useRef<HTMLVideoElement>(null)
  const canvas = useRef<HTMLCanvasElement | null>(null)
  const last = useRef({ code: '', at: 0 })
  const handlers = useRef({ onCode, paused })
  useEffect(() => {
    handlers.current = { onCode, paused }
  })
  const [camera, setCamera] = useState<CameraState>(() =>
    'getUserMedia' in (navigator.mediaDevices ?? {}) ? 'starting' : window.isSecureContext ? 'missing' : 'insecure',
  )
  const [attempt, setAttempt] = useState(0)
  const [mirrored, setMirrored] = useState(true)

  useEffect(() => {
    if (!('getUserMedia' in (navigator.mediaDevices ?? {}))) return // no camera API: see the initial state
    let stream: MediaStream | null = null
    let stopped = false
    navigator.mediaDevices
      .getUserMedia({ video: { facingMode: { ideal: 'environment' }, width: { ideal: 1280 } }, audio: false })
      .then(async (s) => {
        if (stopped) return s.getTracks().forEach((track) => track.stop())
        stream = s
        const facing = s.getVideoTracks()[0]?.getSettings().facingMode
        setMirrored(facing !== 'environment') // a camera that faces the person reads like a mirror
        if (video.current) {
          video.current.srcObject = s
          await video.current.play().catch(() => {})
        }
        setCamera('running')
      })
      .catch((error: DOMException) => {
        if (stopped) return
        setCamera(error.name === 'NotAllowedError' ? 'denied' : error.name === 'NotFoundError' || error.name === 'OverconstrainedError' ? 'missing' : 'failed')
      })
    return () => {
      stopped = true
      stream?.getTracks().forEach((track) => track.stop())
    }
  }, [attempt])

  // reading: a frame every SCAN_EVERY_MS
  useEffect(() => {
    if (camera !== 'running') return
    const timer = setInterval(() => {
      const source = video.current
      canvas.current ??= document.createElement('canvas')
      const target = canvas.current
      if (handlers.current.paused() || !source || source.readyState < source.HAVE_ENOUGH_DATA) return
      const scale = Math.min(1, 640 / source.videoWidth) // smaller frames decode faster
      target.width = Math.round(source.videoWidth * scale)
      target.height = Math.round(source.videoHeight * scale)
      const context = target.getContext('2d', { willReadFrequently: true })
      if (!context) return
      context.drawImage(source, 0, 0, target.width, target.height)
      const frame = context.getImageData(0, 0, target.width, target.height)
      const found = jsQR(frame.data, frame.width, frame.height, { inversionAttempts: 'dontInvert' })
      if (!found?.data) return
      const at = Date.now()
      if (found.data === last.current.code && at - last.current.at < SAME_CODE_PAUSE_MS) return
      last.current = { code: found.data, at }
      handlers.current.onCode(found.data)
    }, SCAN_EVERY_MS)
    return () => clearInterval(timer)
  }, [camera])

  const retry = () => {
    setCamera('starting')
    setAttempt((n) => n + 1)
  }

  return { video, camera, mirrored, retry }
}
