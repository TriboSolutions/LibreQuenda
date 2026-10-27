import { useCallback, useEffect, useState } from 'react'
import { useAuth } from 'react-oidc-context'
import { useMe } from '@/hooks/useMe'
import { api, type Notifications } from '@/lib/api'

const POLL_MS = 30_000

/** Polls GET /me/notifications every 30 s while the tab is visible, and at once when it comes back. */
export function useNotifications() {
  const auth = useAuth()
  const { me } = useMe()
  const token = auth.user?.access_token
  const [data, setData] = useState<Notifications | null>(null)
  const [error, setError] = useState<Error | null>(null)

  useEffect(() => {
    // GET /me (MeProvider) creates the user's row first
    if (!token || !me) return
    let active = true
    const load = () => {
      if (document.hidden) return
      api.notifications
        .list(token)
        .then((next) => {
          if (!active) return
          setData(next)
          setError(null)
        })
        .catch((e: Error) => active && setError(e))
    }
    load()
    const timer = window.setInterval(load, POLL_MS)
    document.addEventListener('visibilitychange', load)
    return () => {
      active = false
      window.clearInterval(timer)
      document.removeEventListener('visibilitychange', load)
    }
  }, [token, me])

  const markAllRead = useCallback(async () => {
    if (!token) return
    setData((current) => current && { unread_count: 0, notifications: current.notifications.map((n) => ({ ...n, read: true })) })
    await api.notifications.markAllRead(token).catch(() => {}) // the next poll shows the truth
  }, [token])

  const signedIn = !!token && !!me
  return { data: signedIn ? data : null, error: signedIn ? error : null, markAllRead }
}
