package gal.tribo.librequenda.ui.bookings

import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.PickVisualMediaRequest
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.selection.selectable
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.RadioButton
import androidx.compose.material3.RadioButtonDefaults
import androidx.compose.material3.Switch
import androidx.compose.material3.SwitchDefaults
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import coil.compose.AsyncImage
import gal.tribo.librequenda.R
import gal.tribo.librequenda.data.Api
import gal.tribo.librequenda.data.Booking
import gal.tribo.librequenda.data.FacilityDay
import gal.tribo.librequenda.data.LocationException
import gal.tribo.librequenda.data.preciseLocation
import gal.tribo.librequenda.data.ISSUE_CATEGORIES
import gal.tribo.librequenda.data.StoredImage
import gal.tribo.librequenda.data.changeBooking
import gal.tribo.librequenda.data.fetchFacilityDay
import gal.tribo.librequenda.data.reportIssue
import gal.tribo.librequenda.data.uploadImage
import gal.tribo.librequenda.ui.components.DateSheet
import gal.tribo.librequenda.ui.components.LqTextField
import gal.tribo.librequenda.ui.components.Notice
import gal.tribo.librequenda.ui.components.SecondaryButton
import gal.tribo.librequenda.ui.components.SuccessOverlay
import gal.tribo.librequenda.ui.entrance.BusyButton
import gal.tribo.librequenda.ui.facility.Pushed
import gal.tribo.librequenda.ui.profile.ProfileActions
import gal.tribo.librequenda.ui.profile.Texts
import gal.tribo.librequenda.ui.profile.userMessage
import gal.tribo.librequenda.ui.theme.Lq
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.launch
import java.time.Instant
import java.time.LocalDate
import java.time.ZoneId

private val ZONE = ZoneId.of("Europe/Madrid")
private val IMAGE_TYPES = setOf("image/jpeg", "image/png", "image/webp")
private const val MAX_IMAGE = 5 * 1024 * 1024
private const val MAX_DESCRIPTION = 1000

/** The page's title, large under the back arrow, and what it is about. */
@Composable
private fun Heading(title: String, intro: String) {
    Column(Modifier.padding(horizontal = 16.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
        Text(title, style = MaterialTheme.typography.headlineMedium, color = Lq.colors.foreground, modifier = Modifier.semantics { heading() })
        Text(intro, style = MaterialTheme.typography.bodyLarge, color = Lq.colors.mutedForeground)
    }
}

/**
 * Something wrong at the facility, while the booking is under way: what kind,
 * what happens, a photo if they want, whether it is urgent and whether the
 * staff may contact them. It lands in the council's maintenance list.
 */
@Composable
fun ReportIssuePage(booking: Booking?, actions: ProfileActions, onBack: () -> Unit) {
    var last by remember { mutableStateOf<Booking?>(null) }
    if (booking != null) last = booking
    Pushed(open = booking != null, onBack = onBack) {
        val b = last ?: return@Pushed
        val context = LocalContext.current
        val scope = rememberCoroutineScope()
        val texts = Texts.current()
        var category by remember(b.id) { mutableStateOf<String?>(null) }
        var description by remember(b.id) { mutableStateOf("") }
        var photo by remember(b.id) { mutableStateOf<StoredImage?>(null) }
        var urgent by remember(b.id) { mutableStateOf(false) }
        var share by remember(b.id) { mutableStateOf(false) }
        var busy by remember { mutableStateOf(false) }
        var uploading by remember { mutableStateOf(false) }
        var error by remember { mutableStateOf<String?>(null) }
        var sent by remember(b.id) { mutableStateOf(false) }
        val chooseText = stringResource(R.string.issue_choose)
        val deniedText = stringResource(R.string.loc_denied)
        val unavailableText = stringResource(R.string.loc_unavailable)

        // sent from the facility only: the precise location first (asked once), then the report
        fun send() = scope.launch {
            busy = true; error = null
            try {
                val position = preciseLocation(context)
                actions.authed { reportIssue(it, b.facilityId, b.id, category!!, description.trim(), photo?.ref, urgent, share, position) }
                sent = true
            } catch (c: CancellationException) { throw c } catch (e: LocationException) {
                error = if (e.denied) deniedText else unavailableText
            } catch (e: Exception) { error = e.userMessage(texts) }
            busy = false
        }
        val askLocation = rememberLauncherForActivityResult(ActivityResultContracts.RequestPermission()) { ok -> if (ok) send() else error = deniedText }

        val pickPhoto = rememberLauncherForActivityResult(ActivityResultContracts.PickVisualMedia()) { uri ->
            if (uri == null) return@rememberLauncherForActivityResult
            val type = context.contentResolver.getType(uri)
            if (type !in IMAGE_TYPES) { error = texts.wrongType; return@rememberLauncherForActivityResult }
            scope.launch {
                uploading = true; error = null
                try {
                    val bytes = context.contentResolver.openInputStream(uri)!!.use { it.readBytes() }
                    if (bytes.size > MAX_IMAGE) error = texts.tooLarge
                    else photo = actions.authed { uploadImage(it, "issues", bytes, type!!) }
                } catch (c: CancellationException) { throw c } catch (e: Exception) { error = e.userMessage(texts) }
                uploading = false
            }
        }

        Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState()).navigationBarsPadding().padding(bottom = 32.dp)) {
            Heading(stringResource(R.string.issue_report), stringResource(R.string.issue_intro, b.facilityName))
            Spacer(Modifier.height(24.dp))
            Column(Modifier.padding(horizontal = 16.dp), verticalArrangement = Arrangement.spacedBy(20.dp)) {
                // the kinds, all in view: one tap, no list to open
                Column {
                    Text(stringResource(R.string.issue_category), style = MaterialTheme.typography.labelLarge, color = Lq.colors.foreground)
                    Spacer(Modifier.height(4.dp))
                    ISSUE_CATEGORIES.forEach { c ->
                        Row(
                            Modifier.fillMaxWidth().selectable(category == c, role = Role.RadioButton) { category = c; error = null }.padding(vertical = 4.dp),
                            verticalAlignment = Alignment.CenterVertically,
                        ) {
                            RadioButton(category == c, onClick = null, colors = RadioButtonDefaults.colors(selectedColor = Lq.colors.primary))
                            Spacer(Modifier.size(12.dp))
                            Text(categoryLabel(c), style = MaterialTheme.typography.bodyLarge, color = Lq.colors.foreground)
                        }
                    }
                }
                LqTextField(
                    stringResource(R.string.issue_what), description, { if (it.length <= MAX_DESCRIPTION) description = it },
                    help = stringResource(R.string.issue_what_help), minLines = 4,
                )
                // a photo, if it helps
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text(stringResource(R.string.issue_photo), style = MaterialTheme.typography.labelLarge, color = Lq.colors.foreground)
                    photo?.let { p ->
                        AsyncImage(Api.absolute(p.url), contentDescription = null, contentScale = androidx.compose.ui.layout.ContentScale.Crop,
                            modifier = Modifier.fillMaxWidth().height(160.dp).clip(RoundedCornerShape(12.dp)).background(Lq.colors.muted))
                    }
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        SecondaryButton(
                            stringResource(if (uploading) R.string.image_uploading else if (photo == null) R.string.image_upload else R.string.image_change),
                            { pickPhoto.launch(PickVisualMediaRequest(ActivityResultContracts.PickVisualMedia.ImageOnly)) },
                            enabled = !uploading && !busy, height = 40.dp,
                        )
                        if (photo != null) SecondaryButton(stringResource(R.string.image_remove), { photo = null }, enabled = !busy, height = 40.dp)
                    }
                }
                SwitchRow(stringResource(R.string.issue_urgent), stringResource(R.string.issue_urgent_help), urgent) { urgent = it }
                SwitchRow(stringResource(R.string.issue_share), null, share) { share = it }
                Text(stringResource(R.string.issue_location_note), style = MaterialTheme.typography.bodySmall, color = Lq.colors.mutedForeground)
                Notice(error, error = true)
                BusyButton(stringResource(if (busy) R.string.issue_sending else R.string.issue_send), busy, enabled = description.isNotBlank() && !uploading) {
                    if (category == null) { error = chooseText; return@BusyButton }
                    askLocation.launch(android.Manifest.permission.ACCESS_FINE_LOCATION) // asks once; answers at once afterwards
                }
            }
        }
        if (sent) SuccessOverlay(stringResource(R.string.issue_sent), onDone = onBack)
    }
}

@Composable
private fun categoryLabel(c: String) = stringResource(
    when (c) {
        "equipment" -> R.string.issue_cat_equipment
        "lighting" -> R.string.issue_cat_lighting
        "cleaning" -> R.string.issue_cat_cleaning
        "changing_rooms" -> R.string.issue_cat_changing_rooms
        "accessibility" -> R.string.issue_cat_accessibility
        "safety" -> R.string.issue_cat_safety
        else -> R.string.issue_cat_other
    }
)

@Composable
private fun SwitchRow(label: String, help: String?, checked: Boolean, onChange: (Boolean) -> Unit) {
    Row(
        Modifier.fillMaxWidth().clip(RoundedCornerShape(6.dp)).clickable(role = Role.Switch) { onChange(!checked) }.padding(vertical = 4.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column(Modifier.weight(1f)) {
            Text(label, style = MaterialTheme.typography.bodyLarge, color = Lq.colors.foreground)
            if (help != null) Text(help, style = MaterialTheme.typography.bodySmall, color = Lq.colors.mutedForeground)
        }
        Spacer(Modifier.size(12.dp))
        Switch(checked, onCheckedChange = null, colors = SwitchDefaults.colors(checkedTrackColor = Lq.colors.primary, checkedThumbColor = Lq.colors.primaryForeground))
    }
}

/**
 * Moving the booking to another free turn of the same facility, any day it can
 * be booked: the day (a calendar in a sheet), its free turns, the current one
 * marked; «Move to Thursday, 18:00» at the bottom. Who joined, the QR and the
 * invite link stay with the booking.
 */
@OptIn(ExperimentalLayoutApi::class)
@Composable
fun ChangeBookingPage(booking: Booking?, actions: ProfileActions, onBack: () -> Unit, onChanged: () -> Unit) {
    var last by remember { mutableStateOf<Booking?>(null) }
    if (booking != null) last = booking
    Pushed(open = booking != null, onBack = onBack) {
        val b = last ?: return@Pushed
        val scope = rememberCoroutineScope()
        val texts = Texts.current()
        var date by remember(b.id) { mutableStateOf(b.startsAt.atZone(ZONE).toLocalDate()) }
        var picked by remember(b.id) { mutableStateOf<Instant?>(null) }
        var day by remember(b.id) { mutableStateOf<FacilityDay?>(null) }
        var failed by remember { mutableStateOf(false) }
        var picking by remember { mutableStateOf(false) }
        var busy by remember { mutableStateOf(false) }
        var error by remember { mutableStateOf<String?>(null) }
        var reload by remember { mutableIntStateOf(0) }
        var done by remember(b.id) { mutableStateOf(false) }
        LaunchedEffect(b.id, date, reload) {
            failed = false; day = null
            try { day = fetchFacilityDay(b.councilSlug, b.facilitySlug, date) } catch (c: CancellationException) { throw c } catch (_: Exception) { failed = true }
        }
        val now = Instant.now()

        Column(Modifier.fillMaxSize()) {
            Column(Modifier.weight(1f).verticalScroll(rememberScrollState())) {
                Heading(
                    stringResource(R.string.change),
                    stringResource(R.string.change_intro, bookingDayInline(b.startsAt), time(b.startsAt), time(b.endsAt)),
                )
                Spacer(Modifier.height(20.dp))
                Row(Modifier.fillMaxWidth().padding(horizontal = 16.dp), verticalAlignment = Alignment.CenterVertically) {
                    Text(stringResource(R.string.change_when), style = MaterialTheme.typography.titleMedium, color = Lq.colors.foreground, modifier = Modifier.weight(1f))
                    // the day, as on the facility page: it opens the calendar
                    Row(
                        Modifier.clip(RoundedCornerShape(20.dp)).background(Lq.colors.muted).clickable { picking = true }.padding(horizontal = 14.dp, vertical = 8.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Icon(painterResource(R.drawable.ic_calendar), contentDescription = null, tint = Lq.colors.foreground, modifier = Modifier.size(18.dp))
                        Spacer(Modifier.size(8.dp))
                        Text(bookingDay(date.atStartOfDay(ZONE).plusHours(12).toInstant()), style = MaterialTheme.typography.labelLarge, color = Lq.colors.foreground)
                    }
                }
                Spacer(Modifier.height(16.dp))
                val d = day
                when {
                    d == null && failed -> Text(stringResource(R.string.search_load_error), color = Lq.colors.destructive, modifier = Modifier.padding(horizontal = 16.dp))
                    d == null -> LinearProgressIndicator(Modifier.fillMaxWidth().padding(16.dp), color = Lq.colors.primary)
                    else -> {
                        // the free turns still to come, and the booking's own (marked, not to pick)
                        val turns = d.slots.filter { (it.free && it.startsAt.isAfter(now)) || it.startsAt == b.startsAt }
                        if (turns.isEmpty()) {
                            Text(stringResource(R.string.change_closed_day), color = Lq.colors.mutedForeground, modifier = Modifier.padding(horizontal = 16.dp))
                        } else FlowRow(Modifier.padding(horizontal = 16.dp), horizontalArrangement = Arrangement.spacedBy(8.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                            turns.forEach { slot ->
                                val current = slot.startsAt == b.startsAt
                                val on = slot.startsAt == picked
                                val ground = when { on -> Lq.colors.primary; current -> Lq.colors.accent; else -> Lq.colors.muted }
                                val ink = when { on -> Lq.colors.primaryForeground; current -> Lq.colors.accentForeground; else -> Lq.colors.foreground }
                                Column(
                                    Modifier.clip(RoundedCornerShape(8.dp)).background(ground)
                                        .selectable(on, enabled = !current && !busy, role = Role.RadioButton) { picked = slot.startsAt; error = null }
                                        .padding(horizontal = 18.dp, vertical = if (current) 6.dp else 12.dp),
                                    horizontalAlignment = Alignment.CenterHorizontally,
                                ) {
                                    Text(time(slot.startsAt), style = MaterialTheme.typography.labelLarge, color = ink)
                                    if (current) Text(stringResource(R.string.change_current), style = MaterialTheme.typography.labelSmall, color = ink)
                                }
                            }
                        }
                    }
                }
                Spacer(Modifier.height(16.dp))
                Box(Modifier.padding(horizontal = 16.dp)) { Notice(error, error = true) }
            }
            // always at hand: move it to the turn chosen
            HorizontalDivider(color = Lq.colors.border)
            Box(Modifier.navigationBarsPadding().padding(16.dp)) {
                val to = picked
                val label = when {
                    busy -> stringResource(R.string.change_saving)
                    to != null -> stringResource(R.string.change_confirm, bookingDayInline(to), time(to))
                    else -> stringResource(R.string.change_pick)
                }
                BusyButton(label, busy, enabled = to != null) {
                    val at = to ?: return@BusyButton
                    scope.launch {
                        busy = true; error = null
                        try {
                            actions.authed { changeBooking(it, b.id, at) }
                            done = true
                        } catch (c: CancellationException) { throw c } catch (e: Exception) {
                            error = e.userMessage(texts)
                            // taken meanwhile: the day again, nothing chosen
                            if ((e as? Api.HttpException)?.status == 409) { picked = null; reload++ }
                        }
                        busy = false
                    }
                }
            }
        }
        if (picking) {
            val today = LocalDate.now(ZONE)
            val until = today.plusDays(((day?.maxAdvanceHours ?: (14 * 24)) / 24).toLong().coerceAtLeast(0))
            DateSheet(stringResource(R.string.day), date, from = today, until = until, onSelect = { date = it; picked = null }, onDismiss = { picking = false })
        }
        if (done) SuccessOverlay(stringResource(R.string.change_done)) { onChanged() }
    }
}
