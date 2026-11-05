package gal.tribo.librequenda.notify

import android.Manifest
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.graphics.Bitmap
import androidx.core.app.NotificationCompat
import androidx.core.app.NotificationManagerCompat
import androidx.core.content.ContextCompat
import gal.tribo.librequenda.MainActivity
import gal.tribo.librequenda.R
import gal.tribo.librequenda.data.AppNotification
import java.time.Instant
import java.time.LocalDate
import java.time.ZoneId
import java.time.format.DateTimeFormatter

/**
 * The app's Android notifications, without push: the reminders come from local
 * alarms (Reminders), every other notice from the background check (NoticesWorker).
 * They look like any app's: in the bar, on the lock screen, opening the app.
 */
object Notifier {
    const val CHANNEL_REMINDERS = "reminders"
    const val CHANNEL_NOTICES = "notices"

    /** Where a tapped notification opens the app (MainActivity reads it). */
    const val EXTRA_OPEN = "open"
    const val OPEN_BOOKINGS = "bookings"
    const val OPEN_NOTICES = "notices"

    /** One booking's page: «booking:<id>». */
    fun openBooking(id: String) = "booking:$id"

    /** A booking that ended: its stars, in the rating sheet. */
    const val OPEN_RATE = "rate:"
    fun openRating(id: String) = OPEN_RATE + id

    // The councils are all in Galicia today, as on the web
    private val ZONE = ZoneId.of("Europe/Madrid")

    /** The two channels, so people can silence one and keep the other (Settings › Notifications). */
    fun createChannels(context: Context) {
        val manager = context.getSystemService(NotificationManager::class.java)
        manager.createNotificationChannel(
            NotificationChannel(CHANNEL_REMINDERS, context.getString(R.string.channel_reminders), NotificationManager.IMPORTANCE_HIGH)
                .apply { description = context.getString(R.string.channel_reminders_help) }
        )
        manager.createNotificationChannel(
            NotificationChannel(CHANNEL_NOTICES, context.getString(R.string.channel_notices), NotificationManager.IMPORTANCE_DEFAULT)
                .apply { description = context.getString(R.string.channel_notices_help) }
        )
    }

    fun canPost(context: Context): Boolean =
        ContextCompat.checkSelfPermission(context, Manifest.permission.POST_NOTIFICATIONS) == PackageManager.PERMISSION_GRANTED &&
            NotificationManagerCompat.from(context).areNotificationsEnabled()

    /**
     * [picture]: the facility's photo, a thumbnail beside the text and large when
     * expanded; [countdownTo]: Android's own countdown to that moment in the header;
     * [action]: a button that opens the same place as tapping.
     */
    fun show(
        context: Context,
        id: Int,
        channel: String,
        title: String,
        text: String,
        open: String,
        picture: Bitmap? = null,
        countdownTo: Instant? = null,
        action: String? = null,
        markRead: Boolean = false, // a «mark as read» button, for the notices
    ) {
        if (!canPost(context)) return
        createChannels(context)
        val intent = Intent(context, MainActivity::class.java)
            .putExtra(EXTRA_OPEN, open)
            .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP or Intent.FLAG_ACTIVITY_SINGLE_TOP)
        val tap = PendingIntent.getActivity(context, id, intent, PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE)
        val builder = NotificationCompat.Builder(context, channel)
            .setSmallIcon(R.drawable.ic_stat_librequenda)
            .setColor(ContextCompat.getColor(context, R.color.brand_blue))
            .setContentTitle(title)
            .setContentText(text)
            .setContentIntent(tap)
            .setAutoCancel(true)
            .setCategory(if (channel == CHANNEL_REMINDERS) NotificationCompat.CATEGORY_REMINDER else NotificationCompat.CATEGORY_EVENT)
        if (picture != null) {
            builder.setLargeIcon(picture)
                .setStyle(NotificationCompat.BigPictureStyle().bigPicture(picture).bigLargeIcon(null as Bitmap?).setSummaryText(text))
        } else {
            builder.setStyle(NotificationCompat.BigTextStyle().bigText(text))
        }
        if (countdownTo != null) {
            builder.setWhen(countdownTo.toEpochMilli()).setShowWhen(true).setUsesChronometer(true).setChronometerCountDown(true)
                .setTimeoutAfter(java.time.Duration.between(Instant.now(), countdownTo).toMillis().coerceAtLeast(0) + 60 * 60_000) // gone an hour after it starts
        }
        if (action != null) builder.addAction(0, action, tap)
        if (markRead) {
            val read = Intent(context, MarkReadReceiver::class.java).setAction("read:$id").putExtra(MarkReadReceiver.EXTRA_ID, id)
            builder.addAction(0, context.getString(R.string.mark_read),
                PendingIntent.getBroadcast(context, id, read, PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE))
        }
        val notification = builder.build()
        try {
            NotificationManagerCompat.from(context).notify(id, notification)
        } catch (_: SecurityException) {
            // the permission was taken away meanwhile
        }
    }

    /** A notice from the API in words, as the web's list says it (notifications.json). */
    fun describe(context: Context, n: AppNotification): Pair<String, String> {
        val day = n.startsAt?.let { day(context, it) }.orEmpty()
        val time = n.startsAt?.let { time(it) }.orEmpty()
        val type = when {
            n.type == "booking_cancelled_by_council" && n.closedFrom != null -> "booking_cancelled_by_closure"
            n.type == "booking_cancelled_by_council" && n.byStaff -> "booking_cancelled_by_staff"
            n.type == "rating_requested" && n.isPublic -> "rating_requested_public" // the others can be rated too
            else -> n.type
        }
        val text = textRes(type)?.let {
            // fixed order, whatever the language's: facility, day, time, council, name, from, until
            context.getString(it, n.facilityName, day, time, n.tenantName, n.name, n.closedFrom?.let(::date).orEmpty(), n.closedUntil?.let(::date).orEmpty())
        } ?: n.facilityName
        val title = when (type) {
            "booking_confirmed" -> R.string.notification_booking_confirmed
            "booking_reminder" -> R.string.notification_booking_reminder
            "booking_cancelled" -> R.string.notification_booking_cancelled
            "booking_cancelled_by_council", "booking_cancelled_by_closure", "booking_cancelled_by_staff" -> R.string.notification_booking_cancelled_by_council
            "issue_solved" -> R.string.notification_issue_solved
            "issue_rejected" -> R.string.notification_issue_rejected
            "rating_requested", "rating_requested_public" -> R.string.rating_title
            "resident_requested" -> R.string.notification_resident_requested // to the council's administrators
            "resident_verified" -> R.string.notification_resident_verified
            "resident_rejected" -> R.string.notification_resident_rejected
            else -> null
        }?.let(context::getString)
            ?: if (type == "council_admin_granted") context.getString(R.string.notification_council_admin_granted, n.tenantName)
            else context.getString(R.string.app_name)
        // the reminder as the app says it (no QR: there are several ways in)
        val body = if (type == "booking_reminder" && n.startsAt != null) {
            "${n.facilityName} · " + context.getString(R.string.reminder_text, time(n.startsAt))
        } else {
            // «Booking cancelled: X, today at 21:00.» under the title «Booking cancelled»: just «X, today at 21:00.»
            val locale = context.resources.configuration.locales[0]
            text.removePrefix("$title:").trim().takeIf { it != text }?.replaceFirstChar { it.titlecase(locale) } ?: text
        }
        return title to body
    }

    private fun textRes(type: String) = when (type) {
        "booking_confirmed" -> R.string.notif_booking_confirmed
        "booking_reminder" -> R.string.notif_booking_reminder
        "booking_changed" -> R.string.notif_booking_changed
        "booking_cancelled" -> R.string.notif_booking_cancelled
        "booking_cancelled_by_council" -> R.string.notif_booking_cancelled_by_council
        "booking_cancelled_by_closure" -> R.string.notif_booking_cancelled_by_closure
        "booking_cancelled_by_staff" -> R.string.notif_booking_cancelled_by_staff
        "council_admin_granted" -> R.string.notif_council_admin_granted
        "issue_solved" -> R.string.notif_issue_solved
        "issue_rejected" -> R.string.notif_issue_rejected
        "booking_joined" -> R.string.notif_booking_joined
        "issue_reported" -> R.string.notif_issue_reported
        "rating_requested" -> R.string.notif_rating_requested
        "rating_requested_public" -> R.string.notif_rating_requested_public
        "resident_requested" -> R.string.notif_resident_requested
        "resident_verified" -> R.string.notif_resident_verified
        "resident_rejected" -> R.string.notif_resident_rejected
        else -> null
    }

    /** «today», «tomorrow» or the date, as words inside a sentence. */
    fun day(context: Context, at: Instant): String {
        val locale = context.resources.configuration.locales[0]
        val date = at.atZone(ZONE).toLocalDate()
        val today = LocalDate.now(ZONE)
        return when (date) {
            today -> context.getString(R.string.today).lowercase(locale)
            today.plusDays(1) -> context.getString(R.string.tomorrow).lowercase(locale)
            else -> date.format(DateTimeFormatter.ofPattern("EEEE d MMMM", locale))
        }
    }

    fun time(at: Instant): String = at.atZone(ZONE).format(DateTimeFormatter.ofPattern("HH:mm"))

    private fun date(at: Instant): String = at.atZone(ZONE).format(DateTimeFormatter.ofPattern("d/M"))
}
