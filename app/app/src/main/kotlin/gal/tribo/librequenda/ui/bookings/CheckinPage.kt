package gal.tribo.librequenda.ui.bookings

import androidx.activity.compose.BackHandler
import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.core.tween
import androidx.compose.animation.slideInHorizontally
import androidx.compose.animation.slideOutHorizontally
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import gal.tribo.librequenda.R
import gal.tribo.librequenda.data.Booking
import gal.tribo.librequenda.ui.components.Notice
import gal.tribo.librequenda.ui.components.SuccessOverlay
import gal.tribo.librequenda.ui.entrance.CodeWay
import gal.tribo.librequenda.ui.entrance.DoorQrWay
import gal.tribo.librequenda.ui.entrance.KeepBright
import gal.tribo.librequenda.ui.entrance.NfcWay
import gal.tribo.librequenda.ui.entrance.QrImage
import gal.tribo.librequenda.ui.entrance.StaffList
import gal.tribo.librequenda.ui.entrance.WAY_ORDER
import gal.tribo.librequenda.ui.entrance.WayChips
import gal.tribo.librequenda.ui.entrance.rememberDoorAttempt
import gal.tribo.librequenda.ui.profile.ProfileActions
import gal.tribo.librequenda.ui.theme.Lq
import java.time.ZoneId
import java.time.format.DateTimeFormatter

private val ZONE = ZoneId.of("Europe/Madrid")

/**
 * Checking in, a page of its own: the ways this facility takes, one chip each
 * (the reader's QR, the door's code, its printed QR, its NFC tag, the staff), and
 * what the entrance answered. Done, it goes back to the booking, now green.
 */
@Composable
fun CheckinPage(booking: Booking?, actions: ProfileActions, onBack: () -> Unit, onCheckedIn: () -> Unit) {
    var last by remember { mutableStateOf<Booking?>(null) }
    if (booking != null) last = booking
    BackHandler(enabled = booking != null, onBack = onBack)
    AnimatedVisibility(visible = booking != null, enter = slideInHorizontally(tween(260)) { it }, exit = slideOutHorizontally(tween(220)) { it }) {
        val b = last ?: return@AnimatedVisibility
        val ways = WAY_ORDER.filter { it in b.checkinWays }
        var way by androidx.compose.runtime.saveable.rememberSaveable(b.id) { mutableStateOf(ways.firstOrNull()) } // kept across the camera
        var success by remember(b.id) { mutableStateOf<String?>(null) } // in: the celebration, then back to the booking
        Column(
            Modifier.fillMaxSize().background(Lq.colors.card)
                .clickable(interactionSource = null, indication = null) {} // taps stay on the page
                .statusBarsPadding()
        ) {
            IconButton(onClick = onBack, modifier = Modifier.padding(start = 4.dp, top = 4.dp)) {
                Icon(painterResource(R.drawable.ic_arrow_left), contentDescription = stringResource(R.string.back), tint = Lq.colors.foreground)
            }
            Column(Modifier.fillMaxWidth().verticalScroll(rememberScrollState()).navigationBarsPadding().padding(bottom = 32.dp)) {
                Column(Modifier.padding(horizontal = 16.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                    Text(stringResource(R.string.checkin_title), style = MaterialTheme.typography.headlineMedium, color = Lq.colors.foreground, modifier = Modifier.semantics { heading() })
                    val time = DateTimeFormatter.ofPattern("HH:mm")
                    Text("${b.facilityName} · ${b.startsAt.atZone(ZONE).format(time)}–${b.endsAt.atZone(ZONE).format(time)}", style = MaterialTheme.typography.bodyLarge, color = Lq.colors.mutedForeground)
                }
                Spacer(Modifier.height(20.dp))
                if (b.checkedIn) {
                    Box(Modifier.padding(horizontal = 16.dp)) {
                        Notice(stringResource(R.string.checkin_done_at, b.checkedInAt?.atZone(ZONE)?.format(DateTimeFormatter.ofPattern("HH:mm")).orEmpty()), error = false)
                    }
                    return@Column
                }
                if (ways.size > 1) {
                    Text(stringResource(R.string.checkin_ways), style = MaterialTheme.typography.titleSmall, color = Lq.colors.foreground, modifier = Modifier.padding(horizontal = 16.dp))
                    Spacer(Modifier.height(10.dp))
                    WayChips(ways, way) { way = it }
                    Spacer(Modifier.height(20.dp))
                }
                Box(Modifier.padding(horizontal = 16.dp)) {
                    val attempt = rememberDoorAttempt(actions, key = b.id to way) { message, _ -> success = message }
                    val here = b.councilSlug to b.facilitySlug
                    when (way) {
                        "reader" -> ReaderWay(b)
                        "entry_code" -> CodeWay(attempt, b.councilSlug, b.facilitySlug, book = false)
                        "printed_qr" -> DoorQrWay(attempt, stringResource(R.string.way_printed_qr_help), book = false, only = here)
                        "nfc" -> NfcWay(attempt, stringResource(R.string.nfc_checkin_help), book = false, only = here)
                        "staff" -> StaffWay(b)
                    }
                }
            }
        }
        success?.let { message -> SuccessOverlay(message, onDone = onCheckedIn) }
    }
}

/** The reader at the door: the booking's QR, large, with the screen at full brightness while it shows. */
@Composable
private fun ReaderWay(b: Booking) {
    KeepBright()
    Column(Modifier.fillMaxWidth(), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(16.dp)) {
        Text(stringResource(R.string.way_reader_help), style = MaterialTheme.typography.bodyLarge, color = Lq.colors.foreground)
        QrImage(b.checkinCode, stringResource(R.string.way_reader))
    }
}

/** The staff mark the entry: who to ask, where, and a call when they have a phone. */
@Composable
private fun StaffWay(b: Booking) {
    Column(Modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(12.dp)) {
        Text(stringResource(if (b.staffContacts.isEmpty()) R.string.way_staff_help else R.string.way_staff_help_contacts), style = MaterialTheme.typography.bodyLarge, color = Lq.colors.foreground)
        StaffList(b.staffContacts)
    }
}
