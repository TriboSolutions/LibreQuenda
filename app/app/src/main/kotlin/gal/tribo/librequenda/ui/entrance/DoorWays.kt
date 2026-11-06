package gal.tribo.librequenda.ui.entrance

import android.Manifest
import android.app.Activity
import android.content.Intent
import android.graphics.Bitmap
import android.graphics.Color as AndroidColor
import android.net.Uri
import android.nfc.NfcAdapter
import android.nfc.tech.Ndef
import android.os.Handler
import android.os.Looper
import android.provider.Settings
import android.view.WindowManager
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.Stable
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.scale
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.graphics.luminance
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.LifecycleResumeEffect
import com.google.zxing.BarcodeFormat
import com.google.zxing.EncodeHintType
import com.google.zxing.qrcode.QRCodeWriter
import com.journeyapps.barcodescanner.ScanContract
import com.journeyapps.barcodescanner.ScanOptions
import gal.tribo.librequenda.R
import gal.tribo.librequenda.data.Door
import gal.tribo.librequenda.data.EntranceResult
import gal.tribo.librequenda.data.LocationException
import gal.tribo.librequenda.data.Position
import gal.tribo.librequenda.data.StaffContact
import gal.tribo.librequenda.data.checkInAtEntrance
import gal.tribo.librequenda.data.parseDoorQr
import gal.tribo.librequenda.data.parseNfcTag
import gal.tribo.librequenda.data.preciseLocation
import gal.tribo.librequenda.ui.components.LqTextField
import gal.tribo.librequenda.ui.components.Notice
import gal.tribo.librequenda.ui.components.SecondaryButton
import gal.tribo.librequenda.ui.profile.ProfileActions
import gal.tribo.librequenda.ui.profile.Texts
import gal.tribo.librequenda.ui.profile.userMessage
import gal.tribo.librequenda.ui.theme.Lq
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.launch

/*
  The ways in at a facility's door, shared by a booking's check-in page and the
  flash booking: the reader's QR, the entry code, the printed QR, the NFC tag and
  the staff. The phone's location goes with the code, the QR and the tag: the API
  checks it is at the facility and never stores it.
*/

/**
 * The colour of what matters on the page (the chosen way, the main button, the NFC mark): the app's blue,
 * or the flash booking's deep blue on its page, as its tab when selected.
 */
val LocalAccent = staticCompositionLocalOf<Color?> { null }

@Composable
private fun accent() = LocalAccent.current ?: Lq.colors.primary

/** Ink on the accent: the theme's on the app's blue; white or dark on another, whichever reads. */
@Composable
private fun onAccent() = LocalAccent.current?.let { if (it.luminance() > 0.5f) Color(0xFF0B1220) else Color.White } ?: Lq.colors.primaryForeground

/** The ways, in the order the web shows them. */
val WAY_ORDER = listOf("reader", "entry_code", "printed_qr", "nfc", "staff")

/** One chip per way, the chosen one filled in the app's blue. */
@Composable
fun WayChips(ways: List<String>, selected: String?, onSelect: (String) -> Unit) {
    LazyRow(contentPadding = PaddingValues(horizontal = 16.dp), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        items(ways) { w -> WayChip(w, w == selected) { onSelect(w) } }
    }
}

@Composable
private fun WayChip(way: String, selected: Boolean, onClick: () -> Unit) {
    val (icon, label) = when (way) {
        "reader" -> R.drawable.ic_qr_code to R.string.way_reader
        "entry_code" -> R.drawable.ic_keypad to R.string.way_entry_code
        "printed_qr" -> R.drawable.ic_scan to R.string.way_printed_qr
        "nfc" -> R.drawable.ic_nfc to R.string.way_nfc
        else -> R.drawable.ic_users to R.string.way_staff
    }
    val ink = if (selected) onAccent() else Lq.colors.foreground
    Row(
        Modifier.height(40.dp).clip(RoundedCornerShape(20.dp)).background(if (selected) accent() else Lq.colors.muted)
            .clickable(role = Role.Tab, onClick = onClick).padding(horizontal = 14.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(painterResource(icon), contentDescription = null, tint = ink, modifier = Modifier.size(18.dp))
        Spacer(Modifier.size(8.dp))
        Text(stringResource(label), color = ink, style = MaterialTheme.typography.labelLarge)
    }
}

/** The main button, with a spinner while the entrance is asked. */
@Composable
fun BusyButton(text: String, busy: Boolean, enabled: Boolean = true, onClick: () -> Unit) {
    Button(
        onClick = onClick, enabled = enabled && !busy, modifier = Modifier.fillMaxWidth().height(48.dp), shape = RoundedCornerShape(6.dp),
        colors = ButtonDefaults.buttonColors(containerColor = accent(), contentColor = onAccent()),
    ) {
        if (busy) {
            CircularProgressIndicator(Modifier.size(18.dp), strokeWidth = 2.dp, color = Lq.colors.mutedForeground)
            Spacer(Modifier.size(10.dp))
        }
        Text(text, style = MaterialTheme.typography.labelLarge)
    }
}

/** The screen at full brightness while a QR shows, so the reader reads it. */
@Composable
fun KeepBright() {
    val context = LocalContext.current
    DisposableEffect(Unit) {
        val window = (context as? Activity)?.window
        val before = window?.attributes?.screenBrightness
        window?.let { it.attributes = it.attributes.apply { screenBrightness = WindowManager.LayoutParams.BRIGHTNESS_OVERRIDE_FULL } }
        onDispose { window?.let { w -> w.attributes = w.attributes.apply { screenBrightness = before ?: WindowManager.LayoutParams.BRIGHTNESS_OVERRIDE_NONE } } }
    }
}

/** A QR, large, black on white. */
@Composable
fun QrImage(content: String, label: String) {
    val qr = remember(content) { content.takeIf { it.isNotBlank() }?.let { qrBitmap(it, 720) } } ?: return
    Image(
        qr.asImageBitmap(), contentDescription = null,
        modifier = Modifier.fillMaxWidth(0.8f).clip(RoundedCornerShape(12.dp)).background(Color.White).padding(12.dp)
            .semantics { contentDescription = label },
    )
}

private fun qrBitmap(content: String, size: Int): Bitmap {
    val matrix = QRCodeWriter().encode(content, BarcodeFormat.QR_CODE, size, size, mapOf(EncodeHintType.MARGIN to 1))
    val pixels = IntArray(size * size) { i -> if (matrix[i % size, i / size]) AndroidColor.BLACK else AndroidColor.WHITE }
    return Bitmap.createBitmap(pixels, size, size, Bitmap.Config.ARGB_8888)
}

/** What the entrance answers, in words (the web's entrance.json). */
@Composable
fun entranceMessages(): Map<String, String> = mapOf(
    "checked_in" to stringResource(R.string.entrance_checked_in),
    "already_checked_in" to stringResource(R.string.entrance_already_checked_in),
    "booked" to stringResource(R.string.entrance_booked),
    "not_booked" to stringResource(R.string.entrance_not_booked),
    "too_early" to stringResource(R.string.entrance_too_early),
    "ended" to stringResource(R.string.entrance_ended),
    "cancelled" to stringResource(R.string.entrance_cancelled),
    "method_off" to stringResource(R.string.entrance_method_off),
    "invalid_qr" to stringResource(R.string.entrance_invalid_qr),
    "wrong_code" to stringResource(R.string.entrance_wrong_code),
    "no_facility_location" to stringResource(R.string.entrance_no_facility_location),
    "location_required" to stringResource(R.string.entrance_location_required),
    "location_inaccurate" to stringResource(R.string.entrance_location_inaccurate) + " " + stringResource(R.string.inaccurate_help),
    "too_far" to stringResource(R.string.entrance_too_far),
    "no_booking" to stringResource(R.string.entrance_no_booking),
    "refused" to stringResource(R.string.entrance_refused),
)

/** Why no turn could be booked at the door (not_booked's reason), in words (the web's reader.json). */
@Composable
fun notBookedReasons(): Map<String, String> = mapOf(
    "no_turn_soon" to stringResource(R.string.reason_no_turn_soon),
    "slot_booked" to stringResource(R.string.reason_slot_booked),
    "slot_closed" to stringResource(R.string.reason_slot_closed),
    "slot_too_far" to stringResource(R.string.reason_slot_too_far),
    "profile_incomplete" to stringResource(R.string.reason_profile_incomplete),
    "age_restricted" to stringResource(R.string.reason_age_restricted),
    "booking_limit_reached" to stringResource(R.string.reason_booking_limit_reached),
    "booking_overlaps" to stringResource(R.string.reason_booking_overlaps),
    "facility_not_found" to stringResource(R.string.reason_facility_not_found),
    "other" to stringResource(R.string.reason_other),
    "use_booking_qr" to stringResource(R.string.reason_use_booking_qr),
    "flash_off" to stringResource(R.string.reason_flash_off),
    "daily_limit_reached" to stringResource(R.string.reason_daily_limit_reached),
    "turns_in_a_row_limit" to stringResource(R.string.reason_turns_in_a_row_limit),
    "no_show_blocked" to stringResource(R.string.reason_no_show_blocked),
    "weekly_limit_reached" to stringResource(R.string.reason_weekly_limit_reached),
    "other_council_booking" to stringResource(R.string.reason_other_council_booking),
    "council_limit_reached" to stringResource(R.string.reason_council_limit_reached),
    "email_unverified" to stringResource(R.string.reason_email_unverified),
    "residents_only" to stringResource(R.string.reason_residents_only),
)

/** The entrance's answer in words: what happened and, when no turn could be booked, why («Esta quenda xa está reservada.»). */
fun entranceWords(result: EntranceResult, messages: Map<String, String>, reasons: Map<String, String>): String {
    val words = messages[result.result] ?: messages["refused"].orEmpty()
    if (result.result != "not_booked" || result.reason == null) return words
    return words + ". " + (reasons[result.reason] ?: reasons["other"].orEmpty())
}

/**
 * Going in at the door: asks for the precise location (once), finds where the
 * phone is, sends the proof and says what the entrance answered. A way in in
 * one step, whichever proof it carries.
 */
@Stable
class DoorAttempt internal constructor() {
    var busy by mutableStateOf(false); internal set
    var locating by mutableStateOf(false); internal set // the first step of busy: where the phone is
    var answer by mutableStateOf<String?>(null); internal set
    var granted by mutableStateOf(false); internal set
    internal var start: (Position?, suspend (String, Position) -> EntranceResult) -> Unit = { _, _ -> }

    /** Locates the phone (unless [at] is already known) and sends; [send] gets the access token and the position. */
    fun go(at: Position? = null, send: suspend (token: String, position: Position) -> EntranceResult) = start(at, send)

    /** A problem before asking the entrance (not this facility's QR…). */
    fun refuse(text: String) { granted = false; answer = text }

    /** The button's words: locating, checking, or the way's own. */
    @Composable
    fun label(idle: String): String = when {
        locating -> stringResource(R.string.locating)
        busy -> stringResource(R.string.checking)
        else -> idle
    }
}

@Composable
fun rememberDoorAttempt(actions: ProfileActions, key: Any? = null, onGranted: (message: String, bookingId: String?) -> Unit): DoorAttempt {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val texts = Texts.current()
    val messages = entranceMessages()
    val reasons = notBookedReasons()
    val deniedText = stringResource(R.string.loc_denied)
    val unavailableText = stringResource(R.string.loc_unavailable)
    val granted by rememberUpdatedState(onGranted)
    val attempt = remember(key) { DoorAttempt() }
    var pending by remember(key) { mutableStateOf<Pair<Position?, suspend (String, Position) -> EntranceResult>?>(null) }

    fun send(at: Position?, proof: suspend (String, Position) -> EntranceResult) = scope.launch {
        attempt.busy = true; attempt.locating = at == null; attempt.answer = null
        try {
            val position = at ?: preciseLocation(context)
            attempt.locating = false
            val result = actions.authed { proof(it, position) }
            attempt.granted = result.granted
            val words = entranceWords(result, messages, reasons)
            if (result.granted) granted(words, result.bookingId) else attempt.answer = words
        } catch (c: CancellationException) { throw c } catch (e: LocationException) {
            attempt.granted = false; attempt.answer = if (e.denied) deniedText else unavailableText
        } catch (e: Exception) {
            attempt.granted = false; attempt.answer = e.userMessage(texts)
        }
        attempt.busy = false; attempt.locating = false
    }

    val askLocation = rememberLauncherForActivityResult(ActivityResultContracts.RequestPermission()) { ok ->
        val (at, proof) = pending ?: return@rememberLauncherForActivityResult
        pending = null
        if (ok) send(at, proof) else attempt.refuse(deniedText)
    }
    attempt.start = { at, proof ->
        if (!attempt.busy) {
            if (at != null) send(at, proof)
            else { pending = at to proof; askLocation.launch(Manifest.permission.ACCESS_FINE_LOCATION) } // asks once; answers at once afterwards
        }
    }
    return attempt
}

/** The entry code typed at [councilSlug]/[facilitySlug]; with [book], a free turn is booked on the spot. */
@Composable
fun CodeWay(attempt: DoorAttempt, councilSlug: String, facilitySlug: String, book: Boolean, at: Position? = null, help: String? = stringResource(R.string.way_entry_code_help)) {
    var code by remember(councilSlug, facilitySlug) { mutableStateOf("") }
    Column(Modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(16.dp)) {
        help?.let { Text(it, style = MaterialTheme.typography.bodyLarge, color = Lq.colors.foreground) }
        LqTextField(stringResource(R.string.entry_code), code, { if (it.length <= 8) code = it }, help = stringResource(R.string.entry_code_help), keyboardType = KeyboardType.Number)
        BusyButton(attempt.label(stringResource(R.string.checkin_button)), attempt.busy, enabled = code.count(Char::isDigit) >= 4) {
            val digits = code.filter(Char::isDigit)
            attempt.go(at) { token, position -> checkInAtEntrance(token, councilSlug, facilitySlug, position, code = digits, book = book) }
        }
        Notice(attempt.answer, error = !attempt.granted)
    }
}

/** The printed QR on the door, read with the camera; [only] keeps it to one facility's door. */
@Composable
fun DoorQrWay(attempt: DoorAttempt, help: String, book: Boolean, only: Pair<String, String>? = null) {
    val notEntrance = stringResource(R.string.qr_not_entrance)
    val scan = rememberLauncherForActivityResult(ScanContract()) { result ->
        val text = result.contents ?: return@rememberLauncherForActivityResult
        val door = parseDoorQr(text)
        if (door == null || !door.isAt(only)) attempt.refuse(notEntrance)
        else attempt.go { token, position -> checkInAtEntrance(token, door.councilSlug, door.facilitySlug, position, qrToken = door.token, book = book) }
    }
    Column(Modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(16.dp)) {
        Text(help, style = MaterialTheme.typography.bodyLarge, color = Lq.colors.foreground)
        val prompt = stringResource(R.string.scan_qr)
        BusyButton(attempt.label(stringResource(R.string.open_camera)), attempt.busy) {
            scan.launch(ScanOptions().setDesiredBarcodeFormats(ScanOptions.QR_CODE).setPrompt(prompt).setBeepEnabled(false).setOrientationLocked(false))
        }
        Notice(attempt.answer, error = !attempt.granted)
    }
}

private fun Door.isAt(only: Pair<String, String>?) = only == null || (councilSlug == only.first && facilitySlug == only.second)

/** The NFC tag on the door: read while this shows, then sent with the location; [only] keeps it to one facility's door. */
@Composable
fun NfcWay(attempt: DoorAttempt, help: String, book: Boolean, only: Pair<String, String>? = null) {
    val context = LocalContext.current
    val notEntrance = stringResource(R.string.nfc_not_entrance)
    val nfc = rememberNfcReader { text ->
        val door = parseNfcTag(text)
        if (door == null || !door.isAt(only)) attempt.refuse(notEntrance)
        else attempt.go { token, position -> checkInAtEntrance(token, door.councilSlug, door.facilitySlug, position, nfcToken = door.token, book = book) }
    }
    Column(Modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(16.dp)) {
        Text(help, style = MaterialTheme.typography.bodyLarge, color = Lq.colors.foreground)
        when (nfc) {
            NfcState.Missing -> Notice(stringResource(R.string.nfc_missing), error = true)
            NfcState.Off -> {
                Notice(stringResource(R.string.nfc_off), error = true)
                SecondaryButton(stringResource(R.string.nfc_turn_on), { runCatching { context.startActivity(Intent(Settings.ACTION_NFC_SETTINGS)) } }, Modifier.fillMaxWidth())
            }
            NfcState.Ready -> NfcWaiting(attempt)
        }
        Notice(attempt.answer, error = !attempt.granted)
    }
}

/** Waiting for the tag: a pulse around the NFC mark, or the steps while it is checked. */
@Composable
private fun NfcWaiting(attempt: DoorAttempt) {
    val pulse = rememberInfiniteTransition(label = "nfc")
    val scale by pulse.animateFloat(1f, 1.18f, infiniteRepeatable(tween(900), RepeatMode.Reverse), label = "scale")
    Column(Modifier.fillMaxWidth().padding(vertical = 12.dp), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(14.dp)) {
        Box(contentAlignment = Alignment.Center) {
            Box(Modifier.size(120.dp).scale(if (attempt.busy) 1f else scale).clip(CircleShape).background(accent().copy(alpha = 0.14f)))
            Box(Modifier.size(84.dp).clip(CircleShape).background(accent()), contentAlignment = Alignment.Center) {
                if (attempt.busy) CircularProgressIndicator(Modifier.size(34.dp), strokeWidth = 3.dp, color = onAccent())
                else Icon(painterResource(R.drawable.ic_nfc), contentDescription = null, tint = onAccent(), modifier = Modifier.size(40.dp))
            }
        }
        Text(
            if (attempt.busy) attempt.label("") else stringResource(R.string.nfc_ready),
            style = MaterialTheme.typography.titleSmall, color = Lq.colors.foreground,
        )
    }
}

enum class NfcState { Missing, Off, Ready }

/**
 * Reads NFC tags while the screen is showing (reader mode: the app gets the tag
 * instead of the browser opening its address) and hands over the first address
 * on it. Checks again whether NFC is on each time the app comes back.
 */
@Composable
fun rememberNfcReader(onRead: (String) -> Unit): NfcState {
    val context = LocalContext.current
    val activity = context as? Activity
    val adapter = remember { NfcAdapter.getDefaultAdapter(context) }
    val read by rememberUpdatedState(onRead)
    var state by remember { mutableStateOf(if (adapter == null) NfcState.Missing else if (adapter.isEnabled) NfcState.Ready else NfcState.Off) }
    LifecycleResumeEffect(adapter, activity) {
        if (adapter != null) state = if (adapter.isEnabled) NfcState.Ready else NfcState.Off
        if (adapter != null && activity != null && adapter.isEnabled) {
            val main = Handler(Looper.getMainLooper())
            val flags = NfcAdapter.FLAG_READER_NFC_A or NfcAdapter.FLAG_READER_NFC_B or NfcAdapter.FLAG_READER_NFC_F or NfcAdapter.FLAG_READER_NFC_V
            adapter.enableReaderMode(activity, { tag ->
                val ndef = Ndef.get(tag) ?: return@enableReaderMode
                val message = runCatching { ndef.connect(); ndef.ndefMessage }.getOrNull() ?: ndef.cachedNdefMessage
                runCatching { ndef.close() }
                val uri = message?.records?.firstNotNullOfOrNull { record -> runCatching { record.toUri() }.getOrNull() } ?: return@enableReaderMode
                main.post { read(uri.toString()) }
            }, flags, null)
        }
        onPauseOrDispose { if (adapter != null && activity != null) runCatching { adapter.disableReaderMode(activity) } }
    }
    return state
}

/** Who to ask on site, where, and a call when they have a phone. */
@Composable
fun StaffList(contacts: List<StaffContact>) {
    val context = LocalContext.current
    Column(Modifier.fillMaxWidth()) {
        contacts.forEachIndexed { i, c ->
            if (i > 0) HorizontalDivider(color = Lq.colors.border)
            Row(Modifier.fillMaxWidth().padding(vertical = 8.dp), verticalAlignment = Alignment.CenterVertically) {
                Icon(painterResource(R.drawable.ic_user), contentDescription = null, tint = Lq.colors.foreground, modifier = Modifier.size(22.dp))
                Spacer(Modifier.size(16.dp))
                Column(Modifier.weight(1f)) {
                    Text(c.name, style = MaterialTheme.typography.bodyLarge, color = Lq.colors.foreground)
                    if (c.place.isNotBlank()) Text(c.place, style = MaterialTheme.typography.bodySmall, color = Lq.colors.mutedForeground)
                }
                if (c.phone.isNotBlank()) SecondaryButton(stringResource(R.string.call), {
                    runCatching { context.startActivity(Intent(Intent.ACTION_DIAL, Uri.parse("tel:" + c.phone))) }
                }, height = 40.dp)
            }
        }
    }
}
