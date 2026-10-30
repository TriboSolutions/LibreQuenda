import type { BookingRules } from '@/lib/api'

/** A new facility's window before it is saved: the council's then replaces it unless it keeps its own. */
export const DEFAULT_RULES: BookingRules = {
  max_advance_hours: 14 * 24,
  min_advance_minutes: 0,
  min_cancel_hours: 2,
  max_active_bookings_per_user: 2,
  max_bookings_per_day: 0,
  max_turns_in_a_row: 0,
  no_show_penalty: false,
  no_show_limit: 3,
  no_show_window_days: 30,
  no_show_block_days: 7,
}
