package gal.tribo.librequenda.notify

import android.content.Context
import androidx.core.content.edit
import androidx.work.Constraints
import androidx.work.CoroutineWorker
import androidx.work.ExistingPeriodicWorkPolicy
import androidx.work.NetworkType
import androidx.work.OneTimeWorkRequestBuilder
import androidx.work.PeriodicWorkRequestBuilder
import androidx.work.WorkManager
import androidx.work.WorkerParameters
import gal.tribo.librequenda.auth.AuthManager
import gal.tribo.librequenda.data.fetchMyBookings
import gal.tribo.librequenda.data.fetchNotifications
import java.util.concurrent.TimeUnit

/**
 * The background check, every 15 minutes (Android's shortest): new notices from
 * the API become Android notifications, and the reminders are scheduled again
 * (a booking made on the web gets its alarm too). Android runs it with the app
 * closed and after a restart; no push service is involved.
 */
class NoticesWorker(context: Context, params: WorkerParameters) : CoroutineWorker(context, params) {

    override suspend fun doWork(): Result {
        val auth = AuthManager(applicationContext)
        try {
            if (!auth.isSignedIn) return Result.success()
            val token = auth.accessToken() ?: return Result.success()

            runCatching { fetchMyBookings(token) }.onSuccess { Reminders.schedule(applicationContext, it.upcoming) }

            val notices = fetchNotifications(token).items
            val prefs = applicationContext.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
            val seen = prefs.getStringSet(KEY_SEEN, null)
            if (seen == null) {
                // the first check only learns what is there: no flood of old notices
                prefs.edit { putStringSet(KEY_SEEN, notices.map { it.id }.toSet()) }
                return Result.success()
            }
            notices
                .filter { !it.read && it.id !in seen }
                .filter { it.type != "booking_reminder" } // the local alarm already said it, on time
                .filter { it.type != "booking_confirmed" } // the app said it when it booked
                .forEach { n ->
                    val (title, text) = Notifier.describe(applicationContext, n)
                    // the request to rate opens the stars straight away; the rest, the notices
                    val open = if (n.type == "rating_requested" && n.bookingId.isNotBlank()) Notifier.openRating(n.bookingId) else Notifier.OPEN_NOTICES
                    Notifier.show(applicationContext, n.id.hashCode(), Notifier.CHANNEL_NOTICES, title, text, open, markRead = true)
                }
            // what the API lists now has been handled; older ones it no longer lists cannot come back
            prefs.edit { putStringSet(KEY_SEEN, notices.map { it.id }.toSet()) }
            return Result.success()
        } catch (e: Exception) {
            android.util.Log.w("LibreQuenda", "notices: check failed", e)
            return Result.retry()
        } finally {
            auth.dispose()
        }
    }

    companion object {
        private const val NAME = "notices"
        private const val PREFS = "notices"
        private const val KEY_SEEN = "seen"
        private val ONLINE = Constraints.Builder().setRequiredNetworkType(NetworkType.CONNECTED).build()

        /** Once, at start: Android keeps it running every 15 minutes from then on. */
        fun start(context: Context) {
            WorkManager.getInstance(context).enqueueUniquePeriodicWork(
                NAME,
                ExistingPeriodicWorkPolicy.KEEP,
                PeriodicWorkRequestBuilder<NoticesWorker>(15, TimeUnit.MINUTES).setConstraints(ONLINE).build(),
            )
        }

        /** A check right now (after a restart, or on signing in). */
        fun runNow(context: Context) {
            WorkManager.getInstance(context).enqueue(OneTimeWorkRequestBuilder<NoticesWorker>().setConstraints(ONLINE).build())
        }

        /** Signed out: forget what was seen, so the next account starts clean. */
        fun reset(context: Context) {
            context.getSharedPreferences(PREFS, Context.MODE_PRIVATE).edit { clear() }
        }
    }
}
