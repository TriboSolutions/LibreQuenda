package gal.tribo.librequenda.data

import android.content.Context
import androidx.core.content.edit
import org.json.JSONArray
import java.time.Instant

/**
 * The upcoming bookings as the API last gave them, with their reader QR, kept in
 * the app's private storage so they still show at a door without coverage.
 * Only the account's own: signing out or deleting the account forgets them.
 */
object BookingsCache {
    data class Saved(val upcoming: List<Booking>, val savedAt: Instant)

    /** Keeps the upcoming ones of a fetch, for the account [subject]. */
    fun save(context: Context, subject: String, bookings: MyBookings) {
        val upcoming = bookings.upcomingJson ?: return
        prefs(context).edit {
            putString(KEY_SUBJECT, subject)
            putString(KEY_UPCOMING, upcoming)
            putLong(KEY_SAVED_AT, System.currentTimeMillis())
        }
    }

    /** What was kept for [subject] (any account when unknown, as offline before /me answers), without the bookings already over. */
    fun load(context: Context, subject: String?): Saved? {
        val p = prefs(context)
        if (subject != null && p.getString(KEY_SUBJECT, null) != subject) return null
        val raw = p.getString(KEY_UPCOMING, null) ?: return null
        val now = Instant.now()
        val list = runCatching {
            val a = JSONArray(raw)
            List(a.length()) { a.getJSONObject(it).toBooking() }
        }.getOrNull() ?: return null
        return Saved(list.filter { it.endsAt.isAfter(now) }, Instant.ofEpochMilli(p.getLong(KEY_SAVED_AT, 0)))
    }

    fun clear(context: Context) {
        prefs(context).edit { clear() }
    }

    private fun prefs(context: Context) = context.getSharedPreferences("offline_bookings", Context.MODE_PRIVATE)
    private const val KEY_SUBJECT = "subject"
    private const val KEY_UPCOMING = "upcoming"
    private const val KEY_SAVED_AT = "saved_at"
}
