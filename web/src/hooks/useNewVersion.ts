import { useEffect, useState } from 'react'
import { checkApiVersion, NEW_VERSION_EVENT, newerApiVersion } from '@/lib/api'

/** The API's version once it is newer than this page's (a deployment since the page loaded), else "". */
export function useNewVersion() {
  const [version, setVersion] = useState(newerApiVersion)
  useEffect(() => {
    const update = () => setVersion(newerApiVersion())
    update() // noticed before this mounted
    window.addEventListener(NEW_VERSION_EVENT, update)
    return () => window.removeEventListener(NEW_VERSION_EVENT, update)
  }, [])
  return version
}

const RELOADED_FOR = 'librequenda:reloaded-for'
const CHECK_EVERY_MS = 5 * 60 * 1000

/**
 * For screens nobody is in front of (the door reader, the widget in a council's
 * website): reload by themselves on a new version, which they ask for every few
 * minutes since they may go long without a call. Once per version, so a page
 * still served old (a cache in between) does not reload in a loop.
 */
export function useReloadOnNewVersion() {
  const version = useNewVersion()
  useEffect(() => {
    const timer = window.setInterval(() => void checkApiVersion(), CHECK_EVERY_MS)
    return () => window.clearInterval(timer)
  }, [])
  useEffect(() => {
    if (!version) return
    try {
      if (sessionStorage.getItem(RELOADED_FOR) === version) return
      sessionStorage.setItem(RELOADED_FOR, version)
    } catch {
      return // without storage there is no loop guard: better stale than reloading forever
    }
    window.location.reload()
  }, [version])
}
