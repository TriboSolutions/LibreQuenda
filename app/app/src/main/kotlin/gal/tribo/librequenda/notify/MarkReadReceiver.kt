package gal.tribo.librequenda.notify

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import androidx.core.app.NotificationManagerCompat
import gal.tribo.librequenda.auth.AuthManager
import gal.tribo.librequenda.data.markNotificationsRead
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch

/**
 * «Mark as read» on a notice's notification: the notices are read at the API (it
 * marks them all, as the web's bell does), and the notification goes away, all
 * without opening the app.
 */
class MarkReadReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        val id = intent.getIntExtra(EXTRA_ID, 0)
        NotificationManagerCompat.from(context).cancel(id)
        val pending = goAsync()
        CoroutineScope(Dispatchers.IO).launch {
            val auth = AuthManager(context)
            try {
                auth.accessToken()?.let { markNotificationsRead(it) }
            } catch (_: Exception) {
                // offline: they stay unread, the app's notices page can mark them later
            } finally {
                auth.dispose()
                pending.finish()
            }
        }
    }

    companion object {
        const val EXTRA_ID = "id"
    }
}
