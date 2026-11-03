package gal.tribo.librequenda.ui.flash

import android.Manifest
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.selection.selectable
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.RadioButton
import androidx.compose.material3.RadioButtonDefaults
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.Stable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.foundation.gestures.detectTapGestures
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import gal.tribo.librequenda.BuildConfig
import gal.tribo.librequenda.R
import gal.tribo.librequenda.SessionState
import gal.tribo.librequenda.data.LocationException
import gal.tribo.librequenda.data.NearbyFacility
import gal.tribo.librequenda.data.Position
import gal.tribo.librequenda.data.fetchNearby
import gal.tribo.librequenda.data.fetchPersonalQr
import gal.tribo.librequenda.data.fetchPersonalQrStatus
import gal.tribo.librequenda.data.preciseLocation
import gal.tribo.librequenda.ui.components.Notice
import gal.tribo.librequenda.ui.components.PrimaryButton
import gal.tribo.librequenda.ui.entrance.BusyButton
import gal.tribo.librequenda.ui.entrance.CodeWay
import gal.tribo.librequenda.ui.entrance.DoorQrWay
import gal.tribo.librequenda.ui.entrance.KeepBright
import gal.tribo.librequenda.ui.entrance.LocalAccent
import gal.tribo.librequenda.ui.entrance.NfcWay
import gal.tribo.librequenda.ui.entrance.QrImage
import gal.tribo.librequenda.ui.entrance.StaffList
import gal.tribo.librequenda.ui.entrance.WAY_ORDER
import gal.tribo.librequenda.ui.entrance.WayChips
import gal.tribo.librequenda.ui.entrance.entranceMessages
import gal.tribo.librequenda.ui.entrance.rememberDoorAttempt
import gal.tribo.librequenda.ui.home.Reminder
import gal.tribo.librequenda.ui.profile.ProfileActions
import gal.tribo.librequenda.ui.profile.Texts
import gal.tribo.librequenda.ui.profile.userMessage
import gal.tribo.librequenda.ui.theme.Lq
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

/*
  Flash booking, as on the web (FlashBookingDialog): at a facility with a free
  turn, book it on the spot and go in, whichever way the facility takes.
  - Reader: the personal QR. It changes every minute and works once, so a
    screenshot soon stops working; codes are only asked for while it shows.
  - Entry code: the phone's location finds the facility, then its code books and checks in at once.
  - QR at the door and NFC tag: read here, sent with the location, booked and in.
  - Staff: who to ask at the facility the phone is at.
  A booking already made goes in from My bookings instead.
*/
@Composable
fun FlashScreen(
    session: SessionState,
    actions: ProfileActions,
    onSignIn: () -> Unit,
    onProfile: () -> Unit,
    onBooked: (message: String, bookingId: String?) -> Unit,
    modifier: Modifier = Modifier,
) {
    if (!session.signedIn) {
        SignedOut(session, onSignIn, modifier)
        return
    }
    // what matters here takes the flash tab's own deep blue
    CompositionLocalProvider(LocalAccent provides Lq.colors.primaryDeep) { FlashWays(session, actions, onProfile, onBooked, modifier) }
}

@Composable
private fun FlashWays(session: SessionState, actions: ProfileActions, onProfile: () -> Unit, onBooked: (String, String?) -> Unit, modifier: Modifier) {
    var way by rememberSaveable { mutableStateOf("reader") } // kept across the camera
    Column(modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(top = 20.dp, bottom = 40.dp)) {
        Column(Modifier.padding(horizontal = 16.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
            val bookedText = stringResource(R.string.entrance_booked)
            Text(
                stringResource(R.string.flash_title), style = MaterialTheme.typography.headlineMedium, color = Lq.colors.foreground,
                modifier = Modifier.semantics { heading() }
                    // debug builds only: a long press on the title plays the celebration, to try it without booking
                    .then(if (BuildConfig.DEBUG) Modifier.pointerInput(Unit) { detectTapGestures(onLongPress = { onBooked(bookedText, null) }) } else Modifier),
            )
            Text(stringResource(R.string.flash_help), style = MaterialTheme.typography.bodyLarge, color = Lq.colors.mutedForeground)
        }
        Spacer(Modifier.height(20.dp))
        session.me?.let { me ->
            // booking needs both, as on the web
            if (!me.profileComplete) Reminder(stringResource(R.string.incomplete_title), stringResource(R.string.qr_incomplete), stringResource(R.string.incomplete_action), onProfile)
            else if (!me.emailVerified) Reminder(stringResource(R.string.unverified_title), stringResource(R.string.unverified_body), stringResource(R.string.unverified_action), onProfile)
        }
        Text(stringResource(R.string.flash_ways), style = MaterialTheme.typography.titleSmall, color = Lq.colors.foreground, modifier = Modifier.padding(horizontal = 16.dp))
        Spacer(Modifier.height(10.dp))
        WayChips(WAY_ORDER, way) { way = it }
        Spacer(Modifier.height(20.dp))
        Box(Modifier.padding(horizontal = 16.dp)) {
            val attempt = rememberDoorAttempt(actions, key = way, onGranted = onBooked)
            when (way) {
                "reader" -> ReaderFlash(actions, onBooked)
                "entry_code" -> CodeFlash(actions, onBooked)
                "printed_qr" -> DoorQrWay(attempt, stringResource(R.string.flash_door_help), book = true)
                "nfc" -> NfcWay(attempt, stringResource(R.string.nfc_flash_help), book = true)
                "staff" -> StaffFlash(actions)
            }
        }
    }
}

@Composable
private fun SignedOut(session: SessionState, onSignIn: () -> Unit, modifier: Modifier) {
    Column(
        modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(horizontal = 24.dp, vertical = 40.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        Box(Modifier.size(88.dp).clip(CircleShape).background(Lq.colors.brandBlue), contentAlignment = Alignment.Center) {
            Icon(painterResource(R.drawable.ic_zap), contentDescription = null, tint = Color.White, modifier = Modifier.size(40.dp))
        }
        Text(stringResource(R.string.flash_title), style = MaterialTheme.typography.headlineLarge, color = Lq.colors.foreground)
        Text(stringResource(R.string.flash_help), style = MaterialTheme.typography.bodyLarge, color = Lq.colors.mutedForeground, textAlign = TextAlign.Center)
        Text(stringResource(R.string.flash_signed_out), style = MaterialTheme.typography.bodyMedium, color = Lq.colors.mutedForeground, textAlign = TextAlign.Center)
        PrimaryButton(stringResource(R.string.sign_in), onSignIn, Modifier.fillMaxWidth(), enabled = !session.busy)
    }
}

/** The personal QR for the reader: a new one every refresh_seconds while it shows, the screen at full brightness. */
@Composable
private fun ReaderFlash(actions: ProfileActions, onBooked: (String, String?) -> Unit) {
    KeepBright()
    val texts = Texts.current()
    var code by remember { mutableStateOf<String?>(null) }
    var failed by remember { mutableStateOf<String?>(null) }
    var left by remember { mutableIntStateOf(0) }
    val failedText = stringResource(R.string.qr_failed)
    val bookedText = stringResource(R.string.entrance_booked)
    var shown by remember { mutableStateOf(emptyList<String>()) } // the codes shown lately: this one and the one before
    // the reader books on its side: ask what became of the codes shown, to celebrate and open the booking
    LaunchedEffect(Unit) {
        while (true) {
            delay(2000)
            for (c in shown) {
                val status = runCatching { actions.authed { fetchPersonalQrStatus(it, c) } }.getOrNull() ?: continue
                if (status.used) {
                    onBooked(bookedText, status.bookingId)
                    return@LaunchedEffect
                }
            }
        }
    }
    LaunchedEffect(Unit) {
        while (true) {
            val wait = try {
                val qr = actions.authed { fetchPersonalQr(it) }
                code = qr.code; failed = null
                shown = (listOf(qr.code) + shown).take(2)
                qr.refreshSeconds
            } catch (c: CancellationException) { throw c } catch (e: Exception) {
                failed = e.userMessage(texts).ifBlank { failedText }
                10 // try again soon
            }
            for (s in wait downTo 1) { left = s; delay(1000) }
        }
    }
    Column(Modifier.fillMaxWidth(), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(16.dp)) {
        Text(stringResource(R.string.flash_reader_help), style = MaterialTheme.typography.bodyLarge, color = Lq.colors.foreground, modifier = Modifier.fillMaxWidth())
        val c = code
        if (c != null) {
            QrImage(c, stringResource(R.string.way_reader))
            Text(stringResource(R.string.qr_refresh_in, left.toString()), style = MaterialTheme.typography.bodySmall, color = Lq.colors.mutedForeground)
        } else if (failed == null) {
            Box(Modifier.fillMaxWidth(0.8f).height(240.dp), contentAlignment = Alignment.Center) {
                androidx.compose.material3.CircularProgressIndicator(color = Lq.colors.primaryDeep)
            }
        }
        Notice(failed, error = true)
        Column(Modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(6.dp)) {
            Text(stringResource(R.string.qr_private), style = MaterialTheme.typography.bodyMedium, color = Lq.colors.mutedForeground)
            Text(stringResource(R.string.qr_brightness), style = MaterialTheme.typography.bodyMedium, color = Lq.colors.mutedForeground)
        }
    }
}

/** The facilities the phone is at, found by its precise location (checked, never stored). */
@Stable
private class NearbySearch {
    var busy by mutableStateOf(false)
    var position by mutableStateOf<Position?>(null)
    var places by mutableStateOf<List<NearbyFacility>?>(null)
    var problem by mutableStateOf<String?>(null)
    var find: () -> Unit = {}
}

@Composable
private fun rememberNearby(actions: ProfileActions, keep: (NearbyFacility) -> Boolean): NearbySearch {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val texts = Texts.current()
    val messages = entranceMessages()
    val deniedText = stringResource(R.string.loc_denied)
    val unavailableText = stringResource(R.string.loc_unavailable)
    val search = remember { NearbySearch() }
    fun run() = scope.launch {
        search.busy = true; search.problem = null
        try {
            val where = preciseLocation(context)
            val answer = actions.authed { fetchNearby(it, where) }
            if (answer.problem.isNotBlank()) search.problem = messages[answer.problem] ?: messages["refused"]
            else { search.position = where; search.places = answer.facilities.filter(keep) }
        } catch (c: CancellationException) { throw c } catch (e: LocationException) {
            search.problem = if (e.denied) deniedText else unavailableText
        } catch (e: Exception) {
            search.problem = e.userMessage(texts)
        }
        search.busy = false
    }
    val ask = rememberLauncherForActivityResult(ActivityResultContracts.RequestPermission()) { ok -> if (ok) run() else search.problem = deniedText }
    search.find = { if (!search.busy) ask.launch(Manifest.permission.ACCESS_FINE_LOCATION) }
    return search
}

/** «Find where I am», or «find again» once looked. */
@Composable
private fun FindButton(search: NearbySearch, idle: String) {
    val text = when {
        search.busy -> stringResource(R.string.flash_locating)
        search.places != null -> stringResource(R.string.flash_again)
        else -> idle
    }
    BusyButton(text, search.busy) { search.find() }
}

/** The entry code: the facility by the phone's location, then its code books and checks in at once. */
@Composable
private fun CodeFlash(actions: ProfileActions, onBooked: (String, String?) -> Unit) {
    val search = rememberNearby(actions) { it.entryCode }
    var chosen by remember { mutableStateOf<String?>(null) }
    val places = search.places
    Column(Modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(16.dp)) {
        Text(stringResource(R.string.flash_code_help), style = MaterialTheme.typography.bodyLarge, color = Lq.colors.foreground)
        if (places.isNullOrEmpty()) {
            if (places != null) Notice(stringResource(R.string.flash_code_none), error = true)
            FindButton(search, stringResource(R.string.flash_locate))
            Notice(search.problem, error = true)
            return@Column
        }
        // the nearest is chosen as soon as they are found
        val place = places.firstOrNull { "${it.councilSlug}/${it.slug}" == chosen } ?: places.first()
        Text(stringResource(R.string.flash_facility), style = MaterialTheme.typography.titleSmall, color = Lq.colors.foreground)
        Column {
            places.forEachIndexed { i, p ->
                if (i > 0) HorizontalDivider(color = Lq.colors.border)
                PlaceRow(p, selected = p == place) { chosen = "${p.councilSlug}/${p.slug}" }
            }
        }
        val attempt = rememberDoorAttempt(actions, key = place.councilSlug to place.slug, onGranted = onBooked)
        CodeWay(attempt, place.councilSlug, place.slug, book = true, at = search.position, help = null)
    }
}

@Composable
private fun PlaceRow(place: NearbyFacility, selected: Boolean, onClick: () -> Unit) {
    Row(
        Modifier.fillMaxWidth().selectable(selected, role = Role.RadioButton, onClick = onClick).padding(vertical = 6.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        RadioButton(selected, onClick = null, colors = RadioButtonDefaults.colors(selectedColor = Lq.colors.primaryDeep))
        Spacer(Modifier.size(12.dp))
        Column(Modifier.weight(1f)) {
            Text(place.name, style = MaterialTheme.typography.bodyLarge, color = Lq.colors.foreground)
            Text(place.councilName, style = MaterialTheme.typography.bodySmall, color = Lq.colors.mutedForeground)
        }
    }
}

/** Staff to ask on site: the people the council lists for the facility the phone is at. */
@Composable
private fun StaffFlash(actions: ProfileActions) {
    val search = rememberNearby(actions) { it.staffContacts.isNotEmpty() }
    val places = search.places
    Column(Modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(16.dp)) {
        Text(stringResource(R.string.flash_staff_help), style = MaterialTheme.typography.bodyLarge, color = Lq.colors.foreground)
        if (places.isNullOrEmpty()) {
            if (places != null) Notice(stringResource(R.string.flash_staff_none), error = true)
            FindButton(search, stringResource(R.string.flash_staff_locate))
        } else places.forEach { place ->
            Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                Text("${place.name} · ${place.councilName}", style = MaterialTheme.typography.titleSmall, color = Lq.colors.foreground)
                StaffList(place.staffContacts)
            }
        }
        Notice(search.problem, error = true)
    }
}
