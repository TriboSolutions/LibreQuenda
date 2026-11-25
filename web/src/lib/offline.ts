import { ApiError, type Booking, type CheckinMethod } from '@/lib/api'

/*
  The upcoming bookings as the API last gave them, with their reader QR, kept in
  this browser so they still show at a door without coverage (the service worker,
  web/sw.js, built by vite.config.ts, keeps the app itself). Only the signed-in
  account's own: signing out or deleting the account forgets them.
*/
const KEY = 'lq:offline-bookings'

/** The booking's QR shown to the reader, and the staff to ask: no coverage needed on the phone. */
export const OFFLINE_WAYS: CheckinMethod[] = ['reader', 'staff']

export type SavedBookings = { subject: string; saved_at: string; upcoming: Booking[] }

export function saveBookings(subject: string, upcoming: Booking[]) {
  try {
    localStorage.setItem(KEY, JSON.stringify({ subject, saved_at: new Date().toISOString(), upcoming } satisfies SavedBookings))
  } catch {
    // storage full or blocked: nothing kept, the page still works online
  }
}

/** What was kept for `subject` (or for whoever signed in last, when the session is gone), without the bookings already over. */
export function loadSavedBookings(subject: string | undefined): SavedBookings | null {
  try {
    const raw = localStorage.getItem(KEY)
    if (!raw) return null
    const saved = JSON.parse(raw) as SavedBookings
    if (subject && saved.subject !== subject) return null
    const now = Date.now()
    return { ...saved, upcoming: saved.upcoming.filter((booking) => Date.parse(booking.ends_at) > now) }
  } catch {
    return null
  }
}

export function forgetSavedBookings() {
  try {
    localStorage.removeItem(KEY)
  } catch {
    // nothing to forget
  }
}

/** The request never reached the API (no coverage), as opposed to an answer with an error. */
export function isNetworkError(error: unknown) {
  return error instanceof Error && !(error instanceof ApiError)
}
