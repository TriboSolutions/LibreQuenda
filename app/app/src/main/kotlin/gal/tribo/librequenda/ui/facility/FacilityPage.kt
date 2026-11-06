package gal.tribo.librequenda.ui.facility

import android.content.Intent
import android.net.Uri
import androidx.activity.compose.BackHandler
import androidx.annotation.DrawableRes
import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.core.tween
import androidx.compose.animation.slideInHorizontally
import androidx.compose.animation.slideOutHorizontally
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.selection.toggleable
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Switch
import androidx.compose.material3.SwitchDefaults
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.pluralStringResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import coil.compose.AsyncImage
import gal.tribo.librequenda.R
import gal.tribo.librequenda.data.MAX_PUBLIC_DESCRIPTION
import gal.tribo.librequenda.data.FacilityDay
import gal.tribo.librequenda.data.PublicTurn
import gal.tribo.librequenda.data.Slot
import gal.tribo.librequenda.data.joinPublicGame
import gal.tribo.librequenda.data.book
import gal.tribo.librequenda.data.fetchFacilityDay
import gal.tribo.librequenda.notify.Notifier
import gal.tribo.librequenda.ui.components.CheckRow
import gal.tribo.librequenda.ui.components.DateSheet
import gal.tribo.librequenda.ui.components.LqTextField
import gal.tribo.librequenda.ui.components.Notice
import gal.tribo.librequenda.ui.components.PrimaryButton
import gal.tribo.librequenda.ui.components.SuccessOverlay
import gal.tribo.librequenda.ui.profile.ProfileActions
import gal.tribo.librequenda.ui.profile.Texts
import gal.tribo.librequenda.ui.profile.userMessage
import gal.tribo.librequenda.ui.theme.Lq
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.launch
import java.time.Duration
import java.time.Instant
import java.time.LocalDate
import java.time.ZoneId
import java.time.format.DateTimeFormatter

// The councils are all in Galicia today, as on the web
private val ZONE = ZoneId.of("Europe/Madrid")

/** What to confirm: a free turn to book, or someone's public booking to join ([turn]). */
data class Checkout(val day: FacilityDay, val startsAt: Instant, val endsAt: Instant, val turn: PublicTurn? = null)

/** Which facility to open, on which day, and the turn already chosen (from the search), if any. */
data class FacilityTarget(val councilSlug: String, val facilitySlug: String, val date: LocalDate, val slot: Instant? = null)

/** A page pushed in from the right, over everything, with the back arrow alone at the top. */
@Composable
internal fun Pushed(open: Boolean, onBack: () -> Unit, content: @Composable () -> Unit) {
    BackHandler(enabled = open, onBack = onBack)
    AnimatedVisibility(visible = open, enter = slideInHorizontally(tween(260)) { it }, exit = slideOutHorizontally(tween(220)) { it }) {
        Column(
            Modifier.fillMaxSize().background(Lq.colors.card)
                .clickable(interactionSource = null, indication = null) {} // taps stay on the page
                .statusBarsPadding()
        ) {
            IconButton(onClick = onBack, modifier = Modifier.padding(start = 4.dp, top = 4.dp)) {
                Icon(painterResource(R.drawable.ic_arrow_left), contentDescription = stringResource(R.string.back), tint = Lq.colors.foreground)
            }
            // a box, so a page can lay something over itself (the success)
            Box(Modifier.fillMaxSize()) { content() }
        }
    }
}

/**
 * A facility: its photo and what it is, the day's free turns (the one chosen in
 * the search already selected), where it is and its rules; «Book 16:00» at the
 * bottom leads to the confirmation.
 */
@OptIn(ExperimentalLayoutApi::class)
@Composable
fun FacilityPage(
    target: FacilityTarget?,
    signedIn: Boolean,
    me: gal.tribo.librequenda.data.Me? = null, // signed in: the councils that verified them, and their pending requests
    actions: ProfileActions? = null, // to ask the council to verify them as its resident
    onSignIn: () -> Unit, // signed out, the page is to look at: booking asks to sign in
    onBack: () -> Unit,
    onCheckout: (Checkout) -> Unit, // book a free turn, or join a public booking: both confirm first
    token: suspend () -> String? = { null }, // signed in, the day also says who joined each public booking
) {
    var last by remember { mutableStateOf<FacilityTarget?>(null) }
    if (target != null) last = target
    var participants by remember { mutableStateOf<PublicTurn?>(null) } // their own page, over this one
    if (target == null) participants = null
    Pushed(open = target != null, onBack = onBack) {
        val t = last ?: return@Pushed
        val context = LocalContext.current
        var date by remember(t) { mutableStateOf(t.date) }
        var selected by remember(t) { mutableStateOf(t.slot) }
        var day by remember(t) { mutableStateOf<FacilityDay?>(null) }
        var failed by remember(t) { mutableStateOf(false) }
        var picking by remember { mutableStateOf(false) }
        val now = Instant.now()
        LaunchedEffect(t, date) {
            failed = false
            try { day = fetchFacilityDay(t.councilSlug, t.facilitySlug, date, if (signedIn) token() else null) } catch (c: CancellationException) { throw c } catch (_: Exception) { failed = true }
        }
        val d = day
        val residentOf = me?.residentOf.orEmpty() // their residents-only facilities take them
        Column(Modifier.fillMaxSize()) {
            Column(Modifier.weight(1f).verticalScroll(rememberScrollState())) {
                when {
                    d == null && failed -> Text(stringResource(R.string.search_load_error), color = Lq.colors.destructive, modifier = Modifier.padding(16.dp))
                    d == null -> LinearProgressIndicator(Modifier.fillMaxWidth().padding(16.dp))
                    else -> {
                        AsyncImage(d.imageUrl, contentDescription = null, contentScale = ContentScale.Crop,
                            modifier = Modifier.padding(horizontal = 16.dp).fillMaxWidth().height(190.dp).clip(RoundedCornerShape(16.dp)).background(Lq.colors.muted))
                        Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                            Text(d.name, style = MaterialTheme.typography.headlineMedium, color = Lq.colors.foreground, modifier = Modifier.semantics { heading() })
                            Text(listOf(d.councilName, d.sport).filter { it.isNotBlank() }.joinToString(" · "), style = MaterialTheme.typography.bodyMedium, color = Lq.colors.mutedForeground)
                            if (d.details.residentsOnly) gal.tribo.librequenda.ui.components.ResidentsOnlyPill(Modifier.padding(top = 4.dp))
                            // the residents' stars after their bookings; nothing until someone rated
                            gal.tribo.librequenda.ui.components.RatingBadge(d.rating, Modifier.padding(top = 2.dp))
                            if (d.description.isNotBlank()) {
                                Spacer(Modifier.height(6.dp))
                                Text(d.description, style = MaterialTheme.typography.bodyMedium, color = Lq.colors.foreground)
                            }
                        }
                        // the day and its free turns
                        Row(Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp), verticalAlignment = Alignment.CenterVertically) {
                            Text(stringResource(R.string.book_title), style = MaterialTheme.typography.titleLarge, color = Lq.colors.foreground, modifier = Modifier.weight(1f))
                            DayChip(dayLabel(date)) { picking = true }
                        }
                        // residents only: say so before anyone picks a turn the API would refuse
                        if (d.details.residentsOnly && d.councilSlug !in residentOf) {
                            val asking = signedIn && me != null && actions != null // they can ask the council here
                            Text(
                                if (asking) stringResource(R.string.resident_not_verified, d.councilName)
                                else stringResource(if (signedIn) R.string.residents_only_not_verified else R.string.residents_only_sign_in, d.councilName),
                                style = MaterialTheme.typography.bodyMedium, color = Lq.colors.pillNeutralInk,
                                modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp).fillMaxWidth().clip(RoundedCornerShape(12.dp))
                                    .background(Lq.colors.pillNeutralBg).padding(14.dp),
                            )
                            // asking to be verified (or the request waiting) is the button at the foot, in place of booking
                        }
                        // free turns to book, and other people's public bookings to join (green, with their spots left)
                        val free = d.slots.filter { it.free }
                        val publics = d.publicTurns.filter { it.endsAt.isAfter(now) }
                        val times = (free.map { it.startsAt } + publics.map { it.startsAt }).distinct().sorted()
                        if (times.isEmpty()) {
                            Text(stringResource(R.string.no_free), color = Lq.colors.mutedForeground, modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp))
                        } else {
                            FlowRow(Modifier.padding(horizontal = 16.dp, vertical = 8.dp), horizontalArrangement = Arrangement.spacedBy(8.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                                times.forEach { at ->
                                    val turn = publics.firstOrNull { it.startsAt == at }
                                    val on = at == selected
                                    val ground = when { on -> Lq.colors.primary; turn != null -> Lq.colors.pillPositiveBg; else -> Lq.colors.muted }
                                    val ink = when { on -> Lq.colors.primaryForeground; turn != null -> Lq.colors.pillPositiveInk; else -> Lq.colors.foreground }
                                    Column(
                                        Modifier.clip(RoundedCornerShape(8.dp)).background(ground)
                                            .clickable(role = Role.RadioButton) { selected = at }
                                            .padding(horizontal = 18.dp, vertical = if (turn != null) 6.dp else 12.dp),
                                        horizontalAlignment = Alignment.CenterHorizontally,
                                    ) {
                                        Text(time(at), style = MaterialTheme.typography.labelLarge, color = ink)
                                        if (turn?.isEvent == true) Text(stringResource(R.string.event_label), style = MaterialTheme.typography.labelSmall, color = ink)
                                        if (turn != null) Text(
                                            pluralStringResource(R.plurals.spots_left, turn.spotsLeft.coerceAtLeast(0), turn.spotsLeft.coerceAtLeast(0)),
                                            style = MaterialTheme.typography.labelSmall, color = ink,
                                        )
                                    }
                                }
                            }
                        }
                        // the public booking chosen: what they are doing, and who is coming
                        publics.firstOrNull { it.startsAt == selected }?.let { turn ->
                            Column(
                                Modifier.padding(horizontal = 16.dp, vertical = 8.dp).fillMaxWidth().clip(RoundedCornerShape(12.dp))
                                    .background(Lq.colors.pillPositiveBg).padding(14.dp),
                                verticalArrangement = Arrangement.spacedBy(8.dp),
                            ) {
                                Row(verticalAlignment = Alignment.CenterVertically) {
                                    Icon(painterResource(R.drawable.ic_users), contentDescription = null, tint = Lq.colors.pillPositiveInk, modifier = Modifier.size(20.dp))
                                    Spacer(Modifier.size(8.dp))
                                    // whose it is: who booked answers for it
                                    val what = when {
                                        turn.isEvent -> stringResource(R.string.event_label)
                                        turn.owner != null && turn.owner.name.isNotBlank() -> stringResource(R.string.public_by, turn.owner.name)
                                        else -> stringResource(R.string.public_label)
                                    }
                                    Text(
                                        "$what · ${time(turn.startsAt)}–${time(turn.endsAt)}",
                                        style = MaterialTheme.typography.titleSmall, color = Lq.colors.pillPositiveInk,
                                    )
                                }
                                if (turn.isEvent) Text(turn.title, style = MaterialTheme.typography.titleMedium, color = Lq.colors.foreground)
                                // only the council's verified residents join it, as they book here
                                if (d.details.residentsOnly) gal.tribo.librequenda.ui.components.ResidentsOnlyPill()
                                Text(turn.description.ifBlank { stringResource(if (turn.isEvent) R.string.event_no_description else R.string.public_no_description) }, style = MaterialTheme.typography.bodyMedium, color = Lq.colors.foreground)
                                Text(stringResource(R.string.people_count, turn.peopleCount, turn.capacity), style = MaterialTheme.typography.labelLarge, color = Lq.colors.foreground)
                                // who answers for it; who joined, signed in only, on a page of their own
                                turn.owner?.let { PublicPersonRow(it, organiser = true) }
                                if (signedIn && turn.joined.isNotEmpty()) {
                                    ChevronRow(stringResource(R.string.public_see_people), Lq.colors.pillPositiveInk) { participants = turn }
                                }
                            }
                        }
                        HorizontalDivider(Modifier.padding(16.dp), color = Lq.colors.border)
                        Column(Modifier.padding(horizontal = 16.dp).padding(bottom = 24.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
                            val f = d.details
                            // with a pin, the whole row opens the maps app
                            val directions = if (f.latitude != null && f.longitude != null) {
                                {
                                    val geo = Intent(Intent.ACTION_VIEW, Uri.parse("geo:0,0?q=${f.latitude},${f.longitude}(${Uri.encode(d.name)})"))
                                    runCatching { context.startActivity(geo) }
                                    Unit
                                }
                            } else null
                            Detail(R.drawable.ic_map_pin, stringResource(R.string.directions), onClick = directions) {
                                Text(f.address.ifBlank { stringResource(R.string.no_location) }, style = MaterialTheme.typography.bodyMedium, color = Lq.colors.mutedForeground)
                            }
                            Detail(R.drawable.ic_clipboard_list, stringResource(R.string.conditions)) {
                                if (f.slotMinutes > 0) Bullet(pluralStringResource(R.plurals.rule_slot, f.slotMinutes, f.slotMinutes))
                                Bullet(pluralStringResource(R.plurals.rule_cancel, f.minCancelHours, f.minCancelHours))
                                f.minAge?.let { Bullet(stringResource(R.string.min_age, it)) }
                                if (f.residentsOnly) Bullet(stringResource(R.string.residents_only_short))
                            }
                            Detail(R.drawable.ic_scroll_text, stringResource(R.string.usage_rules)) {
                                Text(f.usageRules.ifBlank { stringResource(R.string.no_rules) }, style = MaterialTheme.typography.bodyMedium, color = Lq.colors.mutedForeground)
                            }
                        }
                    }
                }
            }
            // always at hand: book the chosen turn, or join the chosen public booking
            val slot = d?.slots?.firstOrNull { it.startsAt == selected && it.free }
            val turn = d?.publicTurns?.firstOrNull { it.startsAt == selected && it.endsAt.isAfter(now) }
            HorizontalDivider(color = Lq.colors.border)
            Box(Modifier.navigationBarsPadding().padding(16.dp)) {
                when {
                    !signedIn -> PrimaryButton(stringResource(R.string.sign_in_to_book), onSignIn, Modifier.fillMaxWidth())
                    // residents only and not one of them: booking would be refused, so ask to be one instead
                    d != null && d.details.residentsOnly && d.councilSlug !in residentOf && me != null && actions != null ->
                        ResidentRequestButton(Council(d.councilSlug, d.councilName, d.councilPhone), me, actions, Modifier.fillMaxWidth())
                    turn != null && !turn.startsAt.isAfter(now) -> PrimaryButton(stringResource(R.string.public_started), {}, Modifier.fillMaxWidth(), enabled = false)
                    turn != null && turn.spotsLeft <= 0 -> PrimaryButton(stringResource(R.string.public_full), {}, Modifier.fillMaxWidth(), enabled = false)
                    turn != null -> PrimaryButton("${stringResource(R.string.join)} · ${time(turn.startsAt)}", { if (d != null) onCheckout(Checkout(d, turn.startsAt, turn.endsAt, turn)) }, Modifier.fillMaxWidth())
                    else -> PrimaryButton(
                        if (slot != null) stringResource(R.string.book_at, time(slot.startsAt)) else stringResource(R.string.book_title),
                        { if (d != null && slot != null) onCheckout(Checkout(d, slot.startsAt, slot.endsAt)) },
                        Modifier.fillMaxWidth(),
                        enabled = slot != null,
                    )
                }
            }
        }
        if (picking) {
            val today = LocalDate.now(ZONE)
            val until = today.plusDays(((day?.maxAdvanceHours ?: (14 * 24)) / 24).toLong().coerceAtLeast(0))
            DateSheet(stringResource(R.string.day), date, from = today, until = until, onSelect = {
                date = it
                if (selected?.atZone(ZONE)?.toLocalDate() != it) selected = null // the turn chosen was on another day
            }, onDismiss = { picking = false })
        }
    }
    ParticipantsPage(participants, onBack = { participants = null })
}

/** Everyone in a public booking or an event, each with the stars others gave them: who booked it first. */
@Composable
private fun ParticipantsPage(turn: PublicTurn?, onBack: () -> Unit) {
    var last by remember { mutableStateOf<PublicTurn?>(null) }
    if (turn != null) last = turn
    Pushed(open = turn != null, onBack = onBack) {
        val p = last ?: return@Pushed
        Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(horizontal = 16.dp).padding(bottom = 24.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Text(stringResource(R.string.public_people_title), style = MaterialTheme.typography.headlineMedium, color = Lq.colors.foreground, modifier = Modifier.semantics { heading() })
            // what it is, in the public booking's green, as on the facility's turns
            Column(
                Modifier.fillMaxWidth().clip(RoundedCornerShape(12.dp)).background(Lq.colors.pillPositiveBg).padding(14.dp),
                verticalArrangement = Arrangement.spacedBy(6.dp),
            ) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Icon(painterResource(R.drawable.ic_users), contentDescription = null, tint = Lq.colors.pillPositiveInk, modifier = Modifier.size(20.dp))
                    Spacer(Modifier.size(8.dp))
                    val what = when {
                        p.isEvent -> p.title
                        p.owner != null && p.owner.name.isNotBlank() -> stringResource(R.string.public_by, p.owner.name)
                        else -> stringResource(R.string.public_label)
                    }
                    Text("$what · ${time(p.startsAt)}–${time(p.endsAt)}", style = MaterialTheme.typography.titleSmall, color = Lq.colors.pillPositiveInk)
                }
                Text(stringResource(R.string.people_count, p.peopleCount, p.capacity), style = MaterialTheme.typography.labelLarge, color = Lq.colors.foreground)
            }
            Spacer(Modifier.height(4.dp))
            p.owner?.let { PublicPersonRow(it, organiser = true) }
            p.joined.forEach {
                HorizontalDivider(color = Lq.colors.border)
                PublicPersonRow(it, organiser = false)
            }
        }
    }
}

/**
 * The checkout, for booking a turn or joining a public booking alike: where and
 * when, how long, until when it can be cancelled for free, the rules to accept;
 * booking, whether to make it public; joining, what it is and who is coming.
 * Done, it says so with an Android notification and goes on to the booking.
 */
@Composable
fun ConfirmBookingPage(choice: Checkout?, actions: ProfileActions, onBack: () -> Unit, onBooked: (String) -> Unit) {
    var last by remember { mutableStateOf<Checkout?>(null) }
    if (choice != null) last = choice
    Pushed(open = choice != null, onBack = onBack) {
        val checkout = last ?: return@Pushed
        val d = checkout.day
        val turn = checkout.turn
        val slot = Slot(checkout.startsAt, checkout.endsAt, "free")
        val context = LocalContext.current
        val scope = rememberCoroutineScope()
        val texts = Texts.current()
        var accepted by remember(choice) { mutableStateOf(false) }
        var public by remember(choice) { mutableStateOf(false) }
        var description by remember(choice) { mutableStateOf("") }
        var busy by remember { mutableStateOf(false) }
        var error by remember(choice) { mutableStateOf<String?>(null) }
        var done by remember(choice) { mutableStateOf<String?>(null) } // the booking's id, once booked: the celebration first
        val f = d.details
        val minutes = Duration.between(slot.startsAt, slot.endsAt).toMinutes().toInt()
        val bookedTitle = stringResource(if (turn != null) R.string.joined else R.string.notification_booking_confirmed)
        val seeBooking = stringResource(R.string.see_booking)
        // the title says «confirmed»; the text, where and when
        val bookedText = "${d.name} · ${dayInline(slot.startsAt)}, ${time(slot.startsAt)}"

        Column(Modifier.fillMaxSize()) {
            Column(Modifier.weight(1f).verticalScroll(rememberScrollState()).padding(horizontal = 16.dp), verticalArrangement = Arrangement.spacedBy(20.dp)) {
                Text(if (turn?.isEvent == true) turn.title else stringResource(if (turn != null) R.string.public_turn_title else R.string.confirm_title, d.name), style = MaterialTheme.typography.headlineMedium, color = Lq.colors.foreground, modifier = Modifier.semantics { heading() })
                Column(verticalArrangement = Arrangement.spacedBy(14.dp)) {
                    Fact(R.drawable.ic_calendar, "${dayLabel(slot.startsAt.atZone(ZONE).toLocalDate())} · ${time(slot.startsAt)}–${time(slot.endsAt)}")
                    Fact(R.drawable.ic_map_pin, listOf(d.councilName, f.address).filter { it.isNotBlank() }.joinToString(" · "))
                    Fact(R.drawable.ic_clock, "${stringResource(R.string.duration)}: ${stringResource(R.string.minutes, minutes)}")
                    val until = slot.startsAt.minus(Duration.ofHours(f.minCancelHours.toLong()))
                    if (turn == null) Fact(R.drawable.ic_x_circle, "${stringResource(R.string.cancellation)}: ${stringResource(R.string.cancellation_until, "${dayInline(until)} ${time(until)}")}")
                    f.minAge?.let { Fact(R.drawable.ic_user, stringResource(R.string.min_age, it)) }
                    if (f.residentsOnly) Fact(R.drawable.ic_user, stringResource(R.string.residents_only_short))
                }
                // joining: what they are doing, and who is coming
                if (turn != null) {
                    HorizontalDivider(color = Lq.colors.border)
                    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        Text(stringResource(if (turn.isEvent) R.string.event_label else R.string.public_label), style = MaterialTheme.typography.titleMedium, color = Lq.colors.foreground)
                        Text(turn.description.ifBlank { stringResource(if (turn.isEvent) R.string.event_no_description else R.string.public_no_description) }, style = MaterialTheme.typography.bodyMedium, color = Lq.colors.mutedForeground)
                        if (!turn.isEvent && turn.owner != null && turn.owner.name.isNotBlank()) Text(stringResource(R.string.public_by, turn.owner.name), style = MaterialTheme.typography.bodyLarge, color = Lq.colors.foreground)
                        Fact(R.drawable.ic_users, stringResource(R.string.people_count, turn.peopleCount, turn.capacity))
                        turn.owner?.let { PublicPersonRow(it, organiser = true) }
                        Text(pluralStringResource(R.plurals.public_spots, turn.spotsLeft, turn.spotsLeft), style = MaterialTheme.typography.labelLarge, color = Lq.colors.pillPositiveInk)
                    }
                }
                HorizontalDivider(color = Lq.colors.border)
                Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                    Text(stringResource(R.string.rules_label), style = MaterialTheme.typography.titleMedium, color = Lq.colors.foreground)
                    Text(f.usageRules.ifBlank { stringResource(R.string.no_rules) }, style = MaterialTheme.typography.bodyMedium, color = Lq.colors.mutedForeground)
                    CheckRow(stringResource(R.string.accept_rules), accepted, { accepted = it })
                }
                // anyone signed in can join it, while there are spots (facilities for more than one)
                if (turn == null && d.capacity > 1) {
                    HorizontalDivider(color = Lq.colors.border)
                    Row(Modifier.fillMaxWidth().toggleable(value = public, role = Role.Switch) { public = it }, verticalAlignment = Alignment.CenterVertically) {
                        Column(Modifier.weight(1f)) {
                            Text(stringResource(R.string.make_public), style = MaterialTheme.typography.bodyLarge, color = Lq.colors.foreground)
                            Text(stringResource(R.string.make_public_help), style = MaterialTheme.typography.bodySmall, color = Lq.colors.mutedForeground)
                        }
                        Spacer(Modifier.size(12.dp))
                        Switch(public, null, colors = SwitchDefaults.colors(checkedTrackColor = Lq.colors.primary, checkedThumbColor = Lq.colors.primaryForeground))
                    }
                    if (public) LqTextField(stringResource(R.string.public_description), description, { description = it }, help = stringResource(R.string.public_description_help), minLines = 3, maxLength = MAX_PUBLIC_DESCRIPTION)
                }
                Notice(error, error = true)
                Spacer(Modifier.height(8.dp))
            }
            HorizontalDivider(color = Lq.colors.border)
            Box(Modifier.navigationBarsPadding().padding(16.dp)) {
                BusyPrimary(
                    busy,
                    stringResource(
                        when {
                            turn != null && busy -> R.string.joining
                            turn != null -> R.string.join
                            busy -> R.string.confirming
                            else -> R.string.confirm_booking
                        }
                    ),
                    {
                        scope.launch {
                            busy = true; error = null
                            try {
                                val id = if (turn != null) {
                                    actions.authed { joinPublicGame(it, turn.id) }
                                    turn.id // joined, it is the same booking
                                } else {
                                    actions.authed { book(it, d.id, slot.startsAt, public, if (public) description.trim() else "") }
                                }
                                // «Booking confirmed», as any app says it
                                Notifier.show(
                                    context, id.hashCode(), Notifier.CHANNEL_NOTICES, bookedTitle, bookedText,
                                    open = Notifier.openBooking(id), action = seeBooking,
                                )
                                done = id
                            } catch (c: CancellationException) { throw c } catch (e: Exception) { error = e.userMessage(texts) }
                            busy = false
                        }
                    },
                    enabled = accepted,
                )
            }
        }
        done?.let { id -> SuccessOverlay(bookedTitle, bookedText) { onBooked(id) } }
    }
}

/** The confirm button, with a spinner while it books. */
@Composable
private fun BusyPrimary(busy: Boolean, text: String, onClick: () -> Unit, enabled: Boolean) {
    androidx.compose.material3.Button(
        onClick = onClick, enabled = enabled && !busy, modifier = Modifier.fillMaxWidth().height(48.dp), shape = RoundedCornerShape(6.dp),
    ) {
        if (busy) {
            androidx.compose.material3.CircularProgressIndicator(Modifier.size(18.dp), strokeWidth = 2.dp, color = Lq.colors.mutedForeground)
            Spacer(Modifier.size(10.dp))
        }
        Text(text, style = MaterialTheme.typography.labelLarge)
    }
}

@Composable
private fun DayChip(label: String, onClick: () -> Unit) {
    Row(
        Modifier.height(32.dp).clip(RoundedCornerShape(16.dp)).background(Lq.colors.primary).clickable(role = Role.Button, onClick = onClick).padding(start = 12.dp, end = 10.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(label, color = Lq.colors.primaryForeground, style = MaterialTheme.typography.labelMedium)
        Spacer(Modifier.size(4.dp))
        Icon(painterResource(R.drawable.ic_chevron_down), contentDescription = null, tint = Lq.colors.primaryForeground, modifier = Modifier.size(14.dp))
    }
}

@Composable
private fun Detail(@DrawableRes icon: Int, title: String, onClick: (() -> Unit)? = null, content: @Composable () -> Unit) {
    Row(
        (if (onClick != null) Modifier.fillMaxWidth().clickable(role = Role.Button, onClick = onClick) else Modifier.fillMaxWidth()),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(painterResource(icon), contentDescription = null, tint = Lq.colors.foreground, modifier = Modifier.align(Alignment.Top).padding(top = 1.dp).size(22.dp))
        Spacer(Modifier.size(16.dp))
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(4.dp)) {
            Text(title, style = MaterialTheme.typography.titleSmall, color = Lq.colors.foreground)
            content()
        }
        if (onClick != null) Icon(painterResource(R.drawable.ic_chevron_right), contentDescription = null, tint = Lq.colors.mutedForeground, modifier = Modifier.size(20.dp))
    }
}

/** A line that opens something: its label and the right chevron, 48dp to tap. */
@Composable
private fun ChevronRow(label: String, color: androidx.compose.ui.graphics.Color, onClick: () -> Unit) {
    Row(
        Modifier.fillMaxWidth().heightIn(min = 48.dp).clickable(role = Role.Button, onClick = onClick),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(label, style = MaterialTheme.typography.titleSmall, color = color, modifier = Modifier.weight(1f))
        Icon(painterResource(R.drawable.ic_chevron_right), contentDescription = null, tint = color, modifier = Modifier.size(20.dp))
    }
}

@Composable
private fun Fact(@DrawableRes icon: Int, text: String) {
    Row(verticalAlignment = Alignment.CenterVertically) {
        Icon(painterResource(icon), contentDescription = null, tint = Lq.colors.foreground, modifier = Modifier.size(22.dp))
        Spacer(Modifier.size(16.dp))
        Text(text, style = MaterialTheme.typography.bodyLarge, color = Lq.colors.foreground)
    }
}

@Composable
private fun Bullet(text: String) = Text("• $text", style = MaterialTheme.typography.bodyMedium, color = Lq.colors.mutedForeground)

@Composable
private fun dayLabel(d: LocalDate): String {
    val locale = LocalConfiguration.current.locales[0]
    val today = LocalDate.now(ZONE)
    return when (d) {
        today -> stringResource(R.string.today)
        today.plusDays(1) -> stringResource(R.string.tomorrow)
        else -> d.format(DateTimeFormatter.ofPattern("EEE d MMM", locale)).replaceFirstChar { it.titlecase(locale) }
    }
}

@Composable
private fun dayInline(at: Instant): String =
    at.atZone(ZONE).format(DateTimeFormatter.ofPattern("EEEE d MMMM", LocalConfiguration.current.locales[0]))

private fun time(at: Instant): String = at.atZone(ZONE).format(DateTimeFormatter.ofPattern("HH:mm"))

/** One person in a public booking: first name, whether they booked it, and the stars others gave them. */
@Composable
private fun PublicPersonRow(person: gal.tribo.librequenda.data.PublicPerson, organiser: Boolean) {
    Row(Modifier.fillMaxWidth().heightIn(min = 40.dp).padding(vertical = 4.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        Text(
            buildString {
                append(person.name.ifBlank { stringResource(R.string.public_person) })
                if (organiser) append(" · ").append(stringResource(R.string.public_organiser))
            },
            style = MaterialTheme.typography.bodyMedium, fontWeight = if (organiser) FontWeight.Medium else null, color = Lq.colors.foreground,
            modifier = Modifier.weight(1f),
        )
        if (person.rating != null) gal.tribo.librequenda.ui.components.RatingBadge(person.rating)
        else Text(stringResource(R.string.rating_none), style = MaterialTheme.typography.bodySmall, color = Lq.colors.mutedForeground)
    }
}
