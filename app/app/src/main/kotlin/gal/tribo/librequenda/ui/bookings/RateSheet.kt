package gal.tribo.librequenda.ui.bookings

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import gal.tribo.librequenda.R
import gal.tribo.librequenda.data.Api
import gal.tribo.librequenda.data.BookingRating
import gal.tribo.librequenda.data.fetchRating
import gal.tribo.librequenda.data.saveRating
import gal.tribo.librequenda.ui.components.LqSheet
import gal.tribo.librequenda.ui.components.Notice
import gal.tribo.librequenda.ui.components.PrimaryButton
import gal.tribo.librequenda.ui.components.SecondaryButton
import gal.tribo.librequenda.ui.components.StarInput
import gal.tribo.librequenda.ui.profile.ProfileActions
import gal.tribo.librequenda.ui.theme.Lq
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.launch
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.time.format.FormatStyle

/**
 * Rating a booking that ended, from the bottom (as the web's RateDialog): the
 * facility's stars and, after a public booking, the others' with an account.
 * Opened from the history, from the notice, and from its Android notification.
 * Saved, [onSaved] closes it and celebrates, as checking in does.
 */
@Composable
fun RateSheet(bookingId: String, actions: ProfileActions, onDismiss: () -> Unit, onSaved: () -> Unit = {}) {
    var rating by remember(bookingId) { mutableStateOf<BookingRating?>(null) }
    var loadError by remember(bookingId) { mutableStateOf<String?>(null) }
    val failedText = stringResource(R.string.rating_load_error)
    LaunchedEffect(bookingId) {
        try { rating = actions.authed { fetchRating(it, bookingId) } }
        catch (c: CancellationException) { throw c }
        catch (e: Exception) { loadError = (e as? Api.HttpException)?.message?.takeUnless { it.startsWith("HTTP ") } ?: failedText }
    }
    val r = rating
    LqSheet(stringResource(R.string.rating_title), onDismiss) {
        when {
            loadError != null -> {
                Notice(loadError, error = true)
                Spacer(Modifier.height(20.dp))
                SecondaryButton(stringResource(R.string.rating_close), onDismiss, Modifier.fillMaxWidth())
            }
            r == null -> LinearProgressIndicator(Modifier.fillMaxWidth().padding(vertical = 24.dp))
            else -> RateForm(r, actions, onDismiss, onSaved)
        }
    }
}

@Composable
private fun RateForm(rating: BookingRating, actions: ProfileActions, onDismiss: () -> Unit, onSaved: () -> Unit) {
    val scope = rememberCoroutineScope()
    var facilityStars by remember { mutableIntStateOf(rating.facilityStars) }
    var people by remember { mutableStateOf(rating.people.associate { it.userId to it.stars }) }
    var busy by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    val saveError = stringResource(R.string.rating_save_error)
    // saved once, for good: the facility's stars are required (the others' can be left out)
    val ready = facilityStars > 0
    // rated already: the stars given, to look at only
    val readOnly = rating.rated

    Text(
        stringResource(R.string.rating_when, rating.facilityName, bookingDayInline(rating.startsAt), time(rating.startsAt)),
        style = MaterialTheme.typography.bodyLarge, color = Lq.colors.mutedForeground,
    )
    Spacer(Modifier.height(20.dp))
    if (!rating.canRate && !readOnly) {
        Text(stringResource(R.string.rating_closed), style = MaterialTheme.typography.bodyLarge, color = Lq.colors.foreground)
        Spacer(Modifier.height(20.dp))
        SecondaryButton(stringResource(R.string.rating_close), onDismiss, Modifier.fillMaxWidth())
        return
    }
    if (readOnly) {
        Text(stringResource(R.string.rating_done), style = MaterialTheme.typography.bodyLarge, color = Lq.colors.foreground)
        Spacer(Modifier.height(16.dp))
    }
    StarInput(stringResource(R.string.rating_facility, rating.facilityName), facilityStars, { facilityStars = it }, enabled = !busy && !readOnly)
    if (rating.people.isNotEmpty()) {
        Spacer(Modifier.height(16.dp))
        HorizontalDivider(color = Lq.colors.border)
        Spacer(Modifier.height(16.dp))
        Text(stringResource(R.string.rating_people_title), style = MaterialTheme.typography.titleMedium, color = Lq.colors.foreground, modifier = Modifier.semantics { heading() })
        if (!readOnly) {
            Spacer(Modifier.height(4.dp))
            Text(stringResource(R.string.rating_people_help), style = MaterialTheme.typography.bodyMedium, color = Lq.colors.mutedForeground)
        }
        Spacer(Modifier.height(12.dp))
        Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
            val unnamed = stringResource(R.string.people_unnamed)
            val organiser = stringResource(R.string.people_organiser)
            rating.people.forEach { p ->
                val label = p.name.ifBlank { unnamed } + if (p.isOwner) " · $organiser" else ""
                StarInput(label, people[p.userId] ?: 0, { people = people + (p.userId to it) }, enabled = !busy && !readOnly)
            }
        }
    }
    if (readOnly) {
        Spacer(Modifier.height(20.dp))
        SecondaryButton(stringResource(R.string.rating_close), onDismiss, Modifier.fillMaxWidth())
        return
    }
    Spacer(Modifier.height(16.dp))
    Text(stringResource(R.string.rating_private, until(rating)), style = MaterialTheme.typography.bodySmall, color = Lq.colors.mutedForeground)
    Notice(error, error = true)
    Spacer(Modifier.height(20.dp))
    PrimaryButton(
        stringResource(if (busy) R.string.rating_saving else R.string.rating_save),
        {
            scope.launch {
                busy = true; error = null
                try {
                    actions.authed { saveRating(it, rating.bookingId, facilityStars, people) }
                    onSaved()
                } catch (c: CancellationException) { throw c } catch (e: Exception) {
                    error = (e as? Api.HttpException)?.message?.takeUnless { it.startsWith("HTTP ") } ?: saveError
                }
                busy = false
            }
        },
        Modifier.fillMaxWidth(),
        enabled = ready && !busy,
    )
    Spacer(Modifier.height(10.dp))
    SecondaryButton(stringResource(R.string.rating_later), onDismiss, Modifier.fillMaxWidth(), enabled = !busy)
}

/** Until when the stars can change: the date and time, in words. */
@Composable
private fun until(rating: BookingRating): String {
    val locale = LocalConfiguration.current.locales[0]
    val at = rating.rateUntil.atZone(ZoneId.of("Europe/Madrid"))
    return at.format(DateTimeFormatter.ofLocalizedDate(FormatStyle.LONG).withLocale(locale)) + ", " + at.format(DateTimeFormatter.ofPattern("HH:mm", locale))
}
