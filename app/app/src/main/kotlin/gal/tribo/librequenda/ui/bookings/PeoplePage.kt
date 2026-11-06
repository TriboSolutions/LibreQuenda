package gal.tribo.librequenda.ui.bookings

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import gal.tribo.librequenda.R
import gal.tribo.librequenda.data.Booking
import gal.tribo.librequenda.data.BookingPeople
import gal.tribo.librequenda.data.Participant
import gal.tribo.librequenda.data.fetchBookingPeople
import gal.tribo.librequenda.data.removeParticipant
import gal.tribo.librequenda.ui.components.ConfirmSheet
import gal.tribo.librequenda.ui.facility.Pushed
import gal.tribo.librequenda.ui.profile.ProfileActions
import gal.tribo.librequenda.ui.profile.Texts
import gal.tribo.librequenda.ui.profile.userMessage
import gal.tribo.librequenda.ui.theme.Lq
import kotlinx.coroutines.CancellationException

/**
 * The owner's «who is coming», from their booking's page: themselves, then who joined,
 * each with «take out». Someone with an account is told and cannot join it again.
 */
@Composable
fun PeoplePage(booking: Booking?, actions: ProfileActions, onBack: () -> Unit, onChanged: () -> Unit) {
    var last by remember { mutableStateOf<Booking?>(null) }
    if (booking != null) last = booking
    Pushed(open = booking != null, onBack = onBack) {
        val b = last ?: return@Pushed
        val texts = Texts.current()
        var people by remember(b.id) { mutableStateOf<BookingPeople?>(null) }
        var error by remember(b.id) { mutableStateOf<String?>(null) }
        var reload by remember { mutableIntStateOf(0) }
        var toRemove by remember { mutableStateOf<Participant?>(null) }
        LaunchedEffect(b.id, reload) {
            try { people = actions.authed { fetchBookingPeople(it, b.id) }; error = null }
            catch (c: CancellationException) { throw c } catch (e: Exception) { error = e.userMessage(texts) }
        }
        Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(horizontal = 16.dp).padding(bottom = 24.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
            Text(stringResource(R.string.invite_people), style = MaterialTheme.typography.headlineMedium, color = Lq.colors.foreground, modifier = Modifier.semantics { heading() })
            Text("${b.facilityName} · ${bookingDay(b.startsAt)}, ${time(b.startsAt)}–${time(b.endsAt)}", style = MaterialTheme.typography.bodyMedium, color = Lq.colors.mutedForeground)
            val list = people
            when {
                error != null && list == null -> Text(error!!, color = Lq.colors.destructive, modifier = Modifier.padding(vertical = 12.dp))
                list == null -> LinearProgressIndicator(Modifier.fillMaxWidth().padding(vertical = 16.dp))
                else -> {
                    PersonRow("${stringResource(R.string.invite_you)} · ${stringResource(R.string.invite_owner)}", null, bold = true)
                    list.participants.forEach { p ->
                        HorizontalDivider(color = Lq.colors.border)
                        PersonRow(p.name, p.age?.takeIf { p.isGuest }?.let { stringResource(R.string.invite_guest_age, it.toString()) }) {
                            if (!list.closed) TextButton(onClick = { toRemove = p }) {
                                Text(stringResource(R.string.invite_remove_yes), color = Lq.colors.destructive)
                            }
                        }
                    }
                    if (list.participants.isEmpty()) Text(stringResource(R.string.invite_nobody), style = MaterialTheme.typography.bodyMedium, color = Lq.colors.mutedForeground, modifier = Modifier.padding(top = 8.dp))
                    error?.let { Text(it, color = Lq.colors.destructive, modifier = Modifier.padding(top = 8.dp)) }
                }
            }
        }
        toRemove?.let { p ->
            ConfirmSheet(
                title = stringResource(R.string.invite_remove, p.name),
                text = stringResource(if (p.isGuest) R.string.invite_remove_guest_confirm else R.string.invite_remove_confirm, p.name),
                confirm = stringResource(R.string.invite_remove_yes),
                confirming = stringResource(R.string.invite_removing),
                keep = stringResource(R.string.invite_remove_keep),
                onDismiss = { toRemove = null },
                onConfirm = {
                    actions.authed { removeParticipant(it, b.id, p.id) }
                    toRemove = null
                    reload++
                    onChanged()
                },
                errorOf = { it.userMessage(texts) },
            )
        }
    }
}

@Composable
private fun PersonRow(name: String, detail: String?, bold: Boolean = false, action: @Composable () -> Unit = {}) {
    Row(Modifier.fillMaxWidth().heightIn(min = 48.dp).padding(vertical = 4.dp), verticalAlignment = Alignment.CenterVertically) {
        Column(Modifier.weight(1f)) {
            Text(name, style = MaterialTheme.typography.bodyLarge, fontWeight = if (bold) FontWeight.Medium else null, color = Lq.colors.foreground)
            if (detail != null) Text(detail, style = MaterialTheme.typography.bodySmall, color = Lq.colors.mutedForeground)
        }
        action()
    }
}
