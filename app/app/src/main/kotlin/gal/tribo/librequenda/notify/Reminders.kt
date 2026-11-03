package gal.tribo.librequenda.notify

import android.app.AlarmManager
import android.app.PendingIntent
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.graphics.Bitmap
import android.graphics.drawable.BitmapDrawable
import android.os.Build
import coil.ImageLoader
import coil.request.ImageRequest
import coil.request.SuccessResult
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeoutOrNull
import androidx.core.content.edit
import gal.tribo.librequenda.R
import gal.tribo.librequenda.data.Booking
import java.time.Duration
import java.time.Instant

/**
 * Each upcoming booking's reminder, 15 minutes before it starts, as a local alarm:
 * the app knows its bookings, so this one needs no server and is on time even
 * offline, the app closed, or the phone asleep. Scheduled again whenever the
 * bookings load, from the background check, and after a restart.
 */
object Reminders {
    private val BEFORE: Duration = Duration.ofMinutes(15) // as the email reminder
    private const val PREFS = "reminders"
    private const val KEY_SCHEDULED = "scheduled" // booking ids with an alarm set

    fun schedule(context: Context, upcoming: List<Booking>) {
        val alarms = context.getSystemService(AlarmManager::class.java)
        val now = Instant.now()
        val due = upcoming.filter { !it.cancelled && it.startsAt.minus(BEFORE).isAfter(now) }
        val prefs = context.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
        // the ones gone (cancelled, changed, left) lose their alarm
        val before = prefs.getStringSet(KEY_SCHEDULED, emptySet()).orEmpty()
        (before - due.map { it.id }.toSet()).forEach { alarms.cancel(pending(context, it, null)) }

        due.forEach { b ->
            val at = b.startsAt.minus(BEFORE).toEpochMilli()
            val intent = pending(context, b.id, b)
            // exact when Android lets us; otherwise as close as it allows, still through Doze
            val exact = Build.VERSION.SDK_INT < Build.VERSION_CODES.S || alarms.canScheduleExactAlarms()
            if (exact) alarms.setExactAndAllowWhileIdle(AlarmManager.RTC_WAKEUP, at, intent)
            else alarms.setAndAllowWhileIdle(AlarmManager.RTC_WAKEUP, at, intent)
        }
        prefs.edit { putStringSet(KEY_SCHEDULED, due.map { it.id }.toSet()) }
    }

    /** Signed out: no reminders of someone else's bookings on this phone. */
    fun clear(context: Context) = schedule(context, emptyList())

    private fun pending(context: Context, id: String, b: Booking?): PendingIntent {
        val intent = Intent(context, ReminderReceiver::class.java).setAction("reminder:$id")
        if (b != null) {
            intent.putExtra("id", id).putExtra("facility", b.facilityName)
                .putExtra("starts", b.startsAt.toEpochMilli())
                .putExtra("image", b.imageUrl)
        }
        return PendingIntent.getBroadcast(context, id.hashCode(), intent, PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE)
    }
}

/**
 * The alarm went off: the facility's name and photo, when it starts with a
 * countdown, and «remember to check in» (there are several ways: reader, code,
 * door QR, staff). Tapping it, or «See booking», opens their bookings.
 */
class ReminderReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        val id = intent.getStringExtra("id") ?: return
        val facility = intent.getStringExtra("facility").orEmpty()
        val starts = Instant.ofEpochMilli(intent.getLongExtra("starts", 0))
        val image = intent.getStringExtra("image")
        val pending = goAsync() // a few seconds to fetch the photo
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val picture = image?.let { withTimeoutOrNull(5_000) { loadBitmap(context, it) } }
                Notifier.show(
                    context, id.hashCode(), Notifier.CHANNEL_REMINDERS,
                    title = facility,
                    text = context.getString(R.string.reminder_text, Notifier.time(starts)),
                    open = Notifier.openBooking(id),
                    picture = picture,
                    countdownTo = starts,
                    action = context.getString(R.string.see_booking),
                )
            } finally {
                pending.finish()
            }
        }
    }

    private suspend fun loadBitmap(context: Context, url: String): Bitmap? {
        val result = ImageLoader(context).execute(ImageRequest.Builder(context).data(url).allowHardware(false).size(720, 360).build())
        return (result as? SuccessResult)?.drawable?.let { (it as? BitmapDrawable)?.bitmap }
    }
}

/** After a restart the alarms are gone: the background check schedules them again. */
class BootReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        if (intent.action == Intent.ACTION_BOOT_COMPLETED) NoticesWorker.runNow(context)
    }
}
