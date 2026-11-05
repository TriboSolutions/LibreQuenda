package gal.tribo.librequenda.ui.bookings

import gal.tribo.librequenda.ui.components.ConfirmSheet
import gal.tribo.librequenda.ui.components.LqTextField
import androidx.activity.compose.BackHandler
import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.core.tween
import androidx.compose.animation.slideInHorizontally
import androidx.compose.animation.slideOutHorizontally
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.material3.IconButton
import androidx.compose.ui.graphics.Color
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.ui.text.style.TextAlign
import android.content.Intent
import android.net.Uri
import androidx.annotation.DrawableRes
import androidx.compose.foundation.selection.toggleable
import androidx.compose.material3.Switch
import androidx.compose.material3.SwitchDefaults
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.pluralStringResource
import androidx.compose.ui.semantics.Role
import gal.tribo.librequenda.data.inviteUrl
import gal.tribo.librequenda.data.leaveBooking
import gal.tribo.librequenda.data.setPublic
import gal.tribo.librequenda.data.shareInvite
import androidx.compose.foundation.background
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.material3.Icon
import androidx.compose.ui.res.painterResource
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
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
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import coil.compose.AsyncImage
import gal.tribo.librequenda.R
import gal.tribo.librequenda.SessionState
import gal.tribo.librequenda.data.Booking
import gal.tribo.librequenda.data.MyBookings
import gal.tribo.librequenda.data.cancelBooking
import gal.tribo.librequenda.data.fetchMyBookings
import gal.tribo.librequenda.ui.components.DangerButton
import gal.tribo.librequenda.ui.components.GhostButton
import gal.tribo.librequenda.ui.components.Notice
import gal.tribo.librequenda.ui.components.Pill
import gal.tribo.librequenda.ui.components.PrimaryButton
import gal.tribo.librequenda.ui.components.SecondaryButton
import gal.tribo.librequenda.ui.profile.ProfileActions
import gal.tribo.librequenda.ui.profile.Texts
import gal.tribo.librequenda.ui.profile.userMessage
import gal.tribo.librequenda.ui.theme.Lq
import gal.tribo.librequenda.ui.theme.onBrandBlue
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.launch
import java.time.Instant
import java.time.LocalDate
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.time.format.FormatStyle
import java.util.Locale

// Bookings can span councils; all of them are in Galicia today (as on the web)
private val ZONE = ZoneId.of("Europe/Madrid")

/** Their bookings, loaded once for the Bookings tab and its history page. */
class BookingsState internal constructor() {
    var data by mutableStateOf<MyBookings?>(null)
    var loading by mutableStateOf(false)
    var failed by mutableStateOf(false)
    var attempt by mutableIntStateOf(0)
    var notice by mutableStateOf<String?>(null) // «Booking cancelled…», on the list
    var loadingMore by mutableStateOf(false)
    fun reload() { attempt++ }

    /** The history's next page, added to what is shown. */
    suspend fun loadMoreHistory(actions: ProfileActions) {
        val d = data ?: return
        if (!d.historyHasMore || loadingMore) return
        loadingMore = true
        try {
            val next = actions.authed { fetchMyBookings(it, d.historyPage + 1) }
            data = d.copy(history = d.history + next.history, historyPage = next.historyPage, historyHasMore = next.historyHasMore)
        } catch (c: CancellationException) { throw c } catch (_: Exception) { /* the next scroll tries again */ }
        loadingMore = false
    }
}

@Composable
fun rememberBookingsState(session: SessionState, actions: ProfileActions): BookingsState {
    val state = remember { BookingsState() }
    val context = androidx.compose.ui.platform.LocalContext.current
    LaunchedEffect(session.signedIn, session.me?.id, state.attempt) {
        if (!session.signedIn || session.me == null) { state.data = null; return@LaunchedEffect } // GET /me first creates their row
        state.loading = true; state.failed = false
        try {
            state.data = actions.authed { fetchMyBookings(it) }
            // every upcoming booking gets its reminder alarm (and the gone ones lose it)
            state.data?.let { gal.tribo.librequenda.notify.Reminders.schedule(context, it.upcoming) }
        } catch (c: CancellationException) { throw c } catch (_: Exception) { state.failed = true }
        state.loading = false
    }
    return state
}

/**
 * Bookings tab, as the web's /reservas: only the upcoming bookings (made or
 * joined) and what can be done with them; the history is a page of its own.
 * Cancelling is here; checking in, inviting, changing and joining come next.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun BookingsScreen(
    session: SessionState,
    state: BookingsState,
    actions: ProfileActions,
    onSignIn: () -> Unit,
    onBookNow: () -> Unit,
    onHistory: () -> Unit,
    onOpen: (Booking) -> Unit,
    modifier: Modifier = Modifier,
) {
    PullToRefreshBox(isRefreshing = state.loading && state.data != null, onRefresh = state::reload, modifier = modifier.fillMaxSize()) {
        LazyColumn(
            Modifier.fillMaxSize(),
            contentPadding = PaddingValues(start = 16.dp, end = 16.dp, top = 24.dp, bottom = 40.dp),
        ) {
            item {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(stringResource(R.string.bookings_title), style = MaterialTheme.typography.headlineLarge, color = Lq.colors.foreground, modifier = Modifier.weight(1f))
                    // the past ones, on their own page: a link by the title, as «See all»
                    if (session.signedIn) {
                        Row(
                            Modifier.clip(RoundedCornerShape(8.dp)).clickable(onClick = onHistory).padding(start = 8.dp, top = 6.dp, bottom = 6.dp),
                            verticalAlignment = Alignment.CenterVertically,
                        ) {
                            Text(stringResource(R.string.bookings_history), style = MaterialTheme.typography.labelLarge, color = Lq.colors.primary)
                            Icon(painterResource(R.drawable.ic_chevron_right), contentDescription = null, tint = Lq.colors.primary, modifier = Modifier.size(18.dp))
                        }
                    }
                }
                Spacer(Modifier.height(4.dp))
                Text(stringResource(R.string.bookings_subtitle), style = MaterialTheme.typography.bodyMedium, color = Lq.colors.mutedForeground)
                Spacer(Modifier.height(24.dp))
            }
            val d = state.data
            when {
                !session.signedIn -> item {
                    Column(verticalArrangement = Arrangement.spacedBy(16.dp)) {
                        Text(stringResource(R.string.bookings_signed_out), color = Lq.colors.foreground)
                        PrimaryButton(stringResource(R.string.sign_in), onSignIn, Modifier.fillMaxWidth(), enabled = !session.busy)
                    }
                }
                state.failed && d == null -> item { LoadError(state::reload) }
                d == null -> item { LinearProgressIndicator(Modifier.fillMaxWidth()) }
                else -> {
                    if (state.notice != null) item { Notice(state.notice, error = false); Spacer(Modifier.height(12.dp)) }
                    if (d.upcoming.isEmpty()) item {
                        Column(Modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(12.dp)) {
                            Text(stringResource(R.string.bookings_empty_upcoming), color = Lq.colors.mutedForeground)
                            PrimaryButton(stringResource(R.string.bookings_book_now), onBookNow)
                        }
                    }
                    itemsIndexed(d.upcoming, key = { _, b -> b.id }) { i, b ->
                        if (i > 0) HorizontalDivider(color = Lq.colors.border)
                        UpcomingRow(b) { onOpen(b) }
                    }
                }
            }
        }
    }
}

/**
 * A booking, on a page of its own (pushed in from the right, over everything):
 * the photo, when, the entrance, quick actions, whether it is public, then the
 * details and rules; cancelling or leaving last, in red.
 */
@Composable
fun BookingPage(bookingId: String?, state: BookingsState, actions: ProfileActions, onBack: () -> Unit) {
    // the last one stays while the page slides out
    var last by remember { mutableStateOf<Booking?>(null) }
    val current = bookingId?.let { id -> state.data?.upcoming?.firstOrNull { it.id == id } }
    if (current != null) last = current
    BackHandler(enabled = bookingId != null, onBack = onBack)
    AnimatedVisibility(
        visible = bookingId != null && last != null,
        enter = slideInHorizontally(tween(260)) { it },
        exit = slideOutHorizontally(tween(220)) { it },
    ) {
        val booking = last ?: return@AnimatedVisibility
        val texts = Texts.current()
        val context = LocalContext.current
        val scope = rememberCoroutineScope()
        var toCancel by remember { mutableStateOf<Booking?>(null) }
        var toLeave by remember { mutableStateOf<Booking?>(null) }
        var error by remember { mutableStateOf<String?>(null) }
        var publishing by remember { mutableStateOf(false) } // the sheet to make it public, or edit its description
        var checkingIn by androidx.compose.runtime.saveable.rememberSaveable { mutableStateOf(false) } // the check-in page, over this one (kept across the camera)
        var changing by remember { mutableStateOf(false) } // another turn, on a page over this one
        var reporting by androidx.compose.runtime.saveable.rememberSaveable { mutableStateOf(false) } // a problem at the facility (kept across the photo picker)
        val shareText = stringResource(R.string.invite_share_text, booking.facilityName, bookingDayInline(booking.startsAt), time(booking.startsAt))

        Column(
            Modifier.fillMaxSize().background(Lq.colors.card)
                .clickable(interactionSource = null, indication = null) {} // taps stay on the page
                .statusBarsPadding()
        ) {
            IconButton(onClick = onBack, modifier = Modifier.padding(start = 4.dp, top = 4.dp)) {
                Icon(painterResource(R.drawable.ic_arrow_left), contentDescription = stringResource(R.string.back), tint = Lq.colors.foreground)
            }
            BookingContent(
                booking = booking,
                error = error,
                onInvite = {
                    scope.launch {
                        error = null
                        try {
                            val inviteToken = actions.authed { shareInvite(it, booking.id) }
                            // Android's own share sheet: WhatsApp, Telegram, email…
                            val send = Intent(Intent.ACTION_SEND).setType("text/plain").putExtra(Intent.EXTRA_TEXT, "$shareText\n${inviteUrl(inviteToken)}")
                            context.startActivity(Intent.createChooser(send, null))
                        } catch (c: CancellationException) { throw c } catch (e: Exception) { error = e.userMessage(texts) }
                    }
                },
                // making it private needs no more; making it public (or editing it) goes through a sheet, with the description
                onPublic = { public ->
                    if (public) publishing = true
                    else scope.launch {
                        error = null
                        try { actions.authed { setPublic(it, booking.id, false) }; state.reload() }
                        catch (c: CancellationException) { throw c } catch (e: Exception) { error = e.userMessage(texts) }
                    }
                },
                onEditDescription = { publishing = true },
                onDirections = { directions(context, booking) },
                onCancel = { toCancel = booking },
                onLeave = { toLeave = booking },
                onCheckin = { checkingIn = true },
                onChange = { changing = true },
                onReport = { reporting = true },
            )
        }

        ChangeBookingPage(
            booking = if (changing) booking else null,
            actions = actions,
            onBack = { changing = false },
            // moved: back to the booking, at its new time
            onChanged = { state.reload(); changing = false },
        )
        ReportIssuePage(booking = if (reporting) booking else null, actions = actions, onBack = { reporting = false })

        CheckinPage(
            booking = if (checkingIn) booking else null,
            actions = actions,
            onBack = { checkingIn = false },
            // in: back to the booking, now green
            onCheckedIn = { state.reload(); checkingIn = false },
        )

        if (publishing) {
            var description by remember { mutableStateOf(booking.publicDescription) }
            val editing = booking.isPublic
            ConfirmSheet(
                title = if (editing) stringResource(R.string.public_edit) else stringResource(R.string.public_confirm_title),
                text = if (editing) stringResource(R.string.public_toggle_help) else stringResource(R.string.public_confirm_text, booking.facilityName),
                confirm = if (editing) stringResource(R.string.public_save) else stringResource(R.string.public_confirm),
                confirming = stringResource(R.string.public_publishing),
                keep = if (editing) stringResource(R.string.cancel) else stringResource(R.string.public_keep_private),
                onDismiss = { publishing = false },
                onConfirm = {
                    actions.authed { setPublic(it, booking.id, true, description.trim()) }
                    publishing = false
                    state.reload()
                },
                errorOf = { it.userMessage(texts) },
                danger = false,
            ) {
                LqTextField(stringResource(R.string.public_description), description, { if (it.length <= 280) description = it }, help = stringResource(R.string.public_description_help))
            }
        }

        toLeave?.let { b ->
            val leftText = stringResource(R.string.notice_left, b.facilityName)
            ConfirmSheet(
                title = stringResource(R.string.leave_title),
                text = stringResource(R.string.leave_description, b.facilityName, bookingDayInline(b.startsAt), time(b.startsAt)),
                confirm = stringResource(R.string.leave_confirm),
                confirming = stringResource(R.string.leave_confirming),
                keep = stringResource(R.string.leave_keep),
                onDismiss = { toLeave = null },
                onConfirm = {
                    actions.authed { leaveBooking(it, b.id) }
                    toLeave = null
                    state.notice = leftText
                    state.reload()
                    onBack()
                },
                errorOf = { it.userMessage(texts) },
            )
        }
        toCancel?.let { b ->
            val cancelledText = stringResource(R.string.notice_cancelled, b.facilityName, bookingDayInline(b.startsAt))
            ConfirmSheet(
                title = stringResource(R.string.cancel_title),
                text = stringResource(R.string.cancel_description, b.facilityName, bookingDayInline(b.startsAt), time(b.startsAt)),
                confirm = stringResource(R.string.cancel_confirm),
                confirming = stringResource(R.string.cancel_confirming),
                keep = stringResource(R.string.cancel_keep),
                onDismiss = { toCancel = null },
                onConfirm = {
                    actions.authed { cancelBooking(it, b.id) }
                    toCancel = null
                    state.notice = cancelledText
                    state.reload()
                    onBack()
                },
                errorOf = { it.userMessage(texts) },
            )
        }
    }
}

/** The history page's content: every past booking with how it went. */
@Composable
fun BookingHistory(state: BookingsState, actions: ProfileActions, onRate: (Booking) -> Unit, modifier: Modifier = Modifier) {
    val d = state.data
    LazyColumn(modifier.fillMaxSize(), contentPadding = PaddingValues(start = 16.dp, end = 16.dp, top = 16.dp, bottom = 40.dp)) {
        when {
            state.failed && d == null -> item { LoadError(state::reload) }
            d == null -> item { LinearProgressIndicator(Modifier.fillMaxWidth()) }
            d.history.isEmpty() -> item { Text(stringResource(R.string.bookings_empty_history), color = Lq.colors.mutedForeground) }
            else -> {
                itemsIndexed(d.history, key = { _, b -> b.id }) { i, b ->
                    // near the end of what came, the next page
                    if (i >= d.history.size - 5) LaunchedEffect(d.history.size) { state.loadMoreHistory(actions) }
                    if (i > 0) HorizontalDivider(color = Lq.colors.border)
                    HistoryRow(b) { onRate(b) }
                }
                if (d.historyHasMore) item { LinearProgressIndicator(Modifier.fillMaxWidth().padding(vertical = 16.dp)) }
            }
        }
    }
}

@Composable
private fun LoadError(retry: () -> Unit) {
    Row(
        Modifier.fillMaxWidth().clip(RoundedCornerShape(6.dp)).background(Lq.colors.pillNegativeBg).padding(start = 16.dp, end = 4.dp, top = 6.dp, bottom = 6.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(stringResource(R.string.bookings_load_error), color = Lq.colors.pillNegativeInk, modifier = Modifier.weight(1f))
        GhostButton(stringResource(R.string.retry), retry)
    }
}

/** From when the entrance opens (15 minutes before) until it ends: the one to go to now. */
private fun Booking.inProgress(now: Instant = Instant.now()) = !cancelled && !now.isBefore(startsAt.minusSeconds(15 * 60)) && now.isBefore(endsAt)

/**
 * An upcoming booking in the list: the photo, where and when. Tapping it opens
 * everything about it. The one in progress stands out, on the pale brand blue.
 */
@Composable
private fun UpcomingRow(b: Booking, onOpen: () -> Unit) {
    val live = b.inProgress()
    Row(
        Modifier
            .fillMaxWidth()
            .then(if (live) Modifier.padding(vertical = 4.dp).clip(RoundedCornerShape(16.dp)).background(Lq.colors.accent) else Modifier)
            .clickable(onClick = onOpen)
            .padding(vertical = 12.dp, horizontal = if (live) 12.dp else 0.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        AsyncImage(
            b.imageUrl, contentDescription = null, contentScale = ContentScale.Crop,
            modifier = Modifier.size(64.dp).clip(RoundedCornerShape(12.dp)).background(Lq.colors.muted),
        )
        Spacer(Modifier.size(14.dp))
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
            // an event: its title first, then where
            if (b.isEvent) Text(b.title, style = MaterialTheme.typography.titleMedium, color = Lq.colors.foreground, maxLines = 2, overflow = TextOverflow.Ellipsis)
            Text(b.facilityName, style = if (b.isEvent) MaterialTheme.typography.bodyMedium else MaterialTheme.typography.titleMedium, color = Lq.colors.foreground, maxLines = 2, overflow = TextOverflow.Ellipsis)
            Text("${bookingDay(b.startsAt)} · ${time(b.startsAt)}–${time(b.endsAt)}", style = MaterialTheme.typography.bodyMedium, color = Lq.colors.foreground)
            Text(b.councilName, style = MaterialTheme.typography.bodySmall, color = Lq.colors.mutedForeground)
            Tags(b, Modifier.padding(top = 4.dp))
        }
        Icon(painterResource(R.drawable.ic_chevron_right), contentDescription = null, tint = Lq.colors.mutedForeground, modifier = Modifier.size(20.dp))
    }
}

/** «In progress», «Inside», «Invited by …», when they apply (public shows in its switch). */
@Composable
private fun Tags(b: Booking, modifier: Modifier = Modifier) {
    val live = b.inProgress() && !b.checkedIn
    if (!live && !b.checkedIn && b.owner) return
    Row(modifier, horizontalArrangement = Arrangement.spacedBy(6.dp)) {
        if (live) Pill(stringResource(R.string.in_progress), Lq.colors.primary, Lq.colors.primaryForeground)
        if (b.checkedIn) Pill(stringResource(R.string.status_checked_in), Lq.colors.pillPositiveBg, Lq.colors.pillPositiveInk)
        if (!b.owner) Pill(
            when {
                b.isEvent -> stringResource(R.string.role_event)
                b.ownerName.isNotBlank() -> stringResource(R.string.role_invited_by, b.ownerName)
                else -> stringResource(R.string.role_invited)
            },
            Lq.colors.pillNeutralBg, Lq.colors.pillNeutralInk,
        )
    }
}

private data class Entrance(@DrawableRes val icon: Int, val text: String, val ink: Color, val ground: Color)

/** A block of details: its Lucide icon at the side, the title, then the text. */
@Composable
private fun DetailBlock(@DrawableRes icon: Int, title: String, content: @Composable () -> Unit) {
    Row {
        Icon(painterResource(icon), contentDescription = null, tint = Lq.colors.foreground, modifier = Modifier.padding(top = 1.dp).size(22.dp))
        Spacer(Modifier.size(16.dp))
        Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
            Text(title, style = MaterialTheme.typography.titleSmall, color = Lq.colors.foreground)
            content()
        }
    }
}

/** The entrance line: a clock and when it opens, on the pale brand blue. */
@Composable
private fun IconText(@DrawableRes icon: Int, text: String, style: androidx.compose.ui.text.TextStyle, color: Color, textAlign: TextAlign, modifier: Modifier) {
    Row(modifier, verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.Center) {
        Icon(painterResource(icon), contentDescription = null, tint = color, modifier = Modifier.size(20.dp))
        Spacer(Modifier.size(8.dp))
        Text(text, style = style, color = color, textAlign = textAlign)
    }
}

@Composable
private fun Bullet(text: String) = Text("• $text", style = MaterialTheme.typography.bodyMedium, color = Lq.colors.mutedForeground)

private fun directions(context: android.content.Context, b: Booking) {
    val f = b.facility
    if (f.latitude == null || f.longitude == null) return
    // the visitor's own maps app (geo: works with any); the web's Google Maps link otherwise
    val geo = Intent(Intent.ACTION_VIEW, Uri.parse("geo:0,0?q=${f.latitude},${f.longitude}(${Uri.encode(b.facilityName)})"))
    val web = Intent(Intent.ACTION_VIEW, Uri.parse("https://www.google.com/maps/dir/?api=1&destination=${f.latitude},${f.longitude}"))
    runCatching { context.startActivity(geo) }.onFailure { runCatching { context.startActivity(web) } }
}

/** The booking page's content, top to bottom: what matters now first, what cannot be undone last. */
@Composable
private fun BookingContent(
    booking: Booking,
    error: String?,
    onInvite: () -> Unit,
    onPublic: (Boolean) -> Unit,
    onEditDescription: () -> Unit,
    onDirections: () -> Unit,
    onCancel: () -> Unit,
    onLeave: () -> Unit,
    onCheckin: () -> Unit,
    onChange: () -> Unit,
    onReport: () -> Unit,
) {
    val now = Instant.now()
    // the entrance opens 15 minutes before; their own booking, at a facility that takes some way in
    val canCheckIn = booking.owner && !booking.checkedIn && booking.checkinWays.isNotEmpty() &&
        !now.isBefore(booking.startsAt.minusSeconds(15 * 60)) && now.isBefore(booking.endsAt)
    val upcoming = booking.startsAt.isAfter(now)
    // as on the web: the owner moves it while it can still be cancelled; anyone in it reports a problem while it is under way
    val canChange = booking.owner && booking.canChange
    val underWay = !booking.cancelled && !now.isBefore(booking.startsAt) && now.isBefore(booking.endsAt)
    val takesPeople = booking.owner && booking.capacity > 1 && upcoming
    val f = booking.facility
    val hasMap = f.latitude != null && f.longitude != null
    val public = booking.isPublic // the API's word: turning it on goes through the sheet first
    // a booking's page always starts at its top
    // (coming from a sheet, the focus lands on a button down the page and scrolls to it)
    val listState = androidx.compose.foundation.lazy.rememberLazyListState()
    val focus = androidx.compose.ui.platform.LocalFocusManager.current
    androidx.compose.runtime.LaunchedEffect(booking.id) {
        focus.clearFocus()
        listState.scrollToItem(0)
        kotlinx.coroutines.delay(350) // after the page has slid in
        listState.scrollToItem(0)
    }
    run {
        LazyColumn(Modifier.fillMaxSize().navigationBarsPadding(), state = listState, contentPadding = PaddingValues(bottom = 32.dp)) {
            item {
                AsyncImage(
                    booking.imageUrl, contentDescription = null, contentScale = ContentScale.Crop,
                    modifier = Modifier.padding(horizontal = 16.dp).fillMaxWidth().height(190.dp).clip(RoundedCornerShape(16.dp)).background(Lq.colors.muted),
                )
                Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                    if (booking.isEvent) Text(booking.title, style = MaterialTheme.typography.headlineMedium, color = Lq.colors.foreground)
                    Text(booking.facilityName, style = if (booking.isEvent) MaterialTheme.typography.titleMedium else MaterialTheme.typography.headlineMedium, color = Lq.colors.foreground)
                    Text("${bookingDay(booking.startsAt)} · ${time(booking.startsAt)}–${time(booking.endsAt)}", style = MaterialTheme.typography.bodyLarge, color = Lq.colors.foreground)
                    Text(booking.councilName, style = MaterialTheme.typography.bodyMedium, color = Lq.colors.mutedForeground)
                }
            }
            if (error != null) item { Box(Modifier.padding(start = 16.dp, end = 16.dp, bottom = 12.dp)) { Notice(error, error = true) } }
            // what matters now: when the entrance opens (blue); once open and not done, «check in» (red); done (green)
            item {
                val opens = booking.startsAt.minusSeconds(15 * 60)
                val (icon, text, ink, ground) = when {
                    booking.checkedIn -> Entrance(R.drawable.ic_check, stringResource(R.string.checkin_done), Lq.colors.pillPositiveInk, Lq.colors.pillPositiveBg)
                    !now.isBefore(opens) -> Entrance(R.drawable.ic_clock, stringResource(R.string.checkin_now), Lq.colors.pillNegativeInk, Lq.colors.pillNegativeBg)
                    else -> Entrance(R.drawable.ic_clock, stringResource(R.string.checkin_from, time(opens)), Lq.colors.accentForeground, Lq.colors.accent)
                }
                // red to green (or blue to red) eases over, rather than jumping
                val inkNow by androidx.compose.animation.animateColorAsState(ink, tween(450), label = "entrance ink")
                val groundNow by androidx.compose.animation.animateColorAsState(ground, tween(450), label = "entrance ground")
                IconText(
                    icon, text,
                    style = MaterialTheme.typography.labelLarge,
                    color = inkNow,
                    textAlign = TextAlign.Center,
                    // red: it opens the ways in
                    modifier = Modifier.padding(horizontal = 16.dp).fillMaxWidth().clip(RoundedCornerShape(12.dp)).background(groundNow)
                        .then(if (canCheckIn) Modifier.clickable(onClick = onCheckin) else Modifier).padding(14.dp),
                )
            }
            // quick actions, round as the Home's
            if (takesPeople || hasMap || canCheckIn || canChange || underWay) item {
                // a row that scrolls sideways when they do not all fit
                Row(
                    Modifier.fillMaxWidth().horizontalScroll(rememberScrollState()).padding(horizontal = 16.dp, vertical = 16.dp),
                    horizontalArrangement = Arrangement.spacedBy(24.dp),
                ) {
                    if (canCheckIn) RoundAction(R.drawable.ic_log_in, stringResource(R.string.checkin_button)) { onCheckin() }
                    if (takesPeople) RoundAction(R.drawable.ic_user_plus, stringResource(R.string.invite)) { onInvite() }
                    if (canChange) RoundAction(R.drawable.ic_calendar_clock, stringResource(R.string.change)) { onChange() }
                    if (hasMap) RoundAction(R.drawable.ic_map_pin, stringResource(R.string.directions)) { onDirections() }
                    if (underWay) RoundAction(R.drawable.ic_wrench, stringResource(R.string.issue_report)) { onReport() }
                }
            }
            if (takesPeople) item {
                HorizontalDivider(Modifier.padding(horizontal = 16.dp), color = Lq.colors.border)
                Row(
                    Modifier.fillMaxWidth().toggleable(value = public, role = Role.Switch) { onPublic(it) }.padding(16.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Icon(painterResource(R.drawable.ic_globe), contentDescription = null, tint = Lq.colors.foreground, modifier = Modifier.size(22.dp))
                    Spacer(Modifier.size(16.dp))
                    Column(Modifier.weight(1f)) {
                        Text(stringResource(R.string.public_label), style = MaterialTheme.typography.bodyLarge, color = Lq.colors.foreground)
                        Text(stringResource(R.string.public_toggle_help), style = MaterialTheme.typography.bodySmall, color = Lq.colors.mutedForeground)
                    }
                    Spacer(Modifier.size(12.dp))
                    Switch(
                        checked = public, onCheckedChange = null,
                        colors = SwitchDefaults.colors(checkedTrackColor = Lq.colors.primary, checkedThumbColor = Lq.colors.primaryForeground),
                    )
                }
                // public: what they are doing, as people who join see it, and the way to change it
                if (public) {
                    Row(Modifier.fillMaxWidth().clickable(onClick = onEditDescription).padding(start = 54.dp, end = 16.dp, bottom = 16.dp), verticalAlignment = Alignment.Top) {
                        Text(
                            booking.publicDescription.ifBlank { stringResource(R.string.public_no_description) },
                            style = MaterialTheme.typography.bodyMedium,
                            color = if (booking.publicDescription.isBlank()) Lq.colors.mutedForeground else Lq.colors.foreground,
                            modifier = Modifier.weight(1f),
                        )
                        Spacer(Modifier.size(12.dp))
                        Text(stringResource(R.string.public_edit), style = MaterialTheme.typography.labelLarge, color = Lq.colors.primary)
                    }
                }
                Row(Modifier.padding(start = 16.dp, end = 16.dp, bottom = 16.dp), verticalAlignment = Alignment.CenterVertically) {
                    Icon(painterResource(R.drawable.ic_users), contentDescription = null, tint = Lq.colors.foreground, modifier = Modifier.size(22.dp))
                    Spacer(Modifier.size(16.dp))
                    Text(stringResource(R.string.people_count, booking.people, booking.capacity), style = MaterialTheme.typography.bodyLarge, color = Lq.colors.foreground)
                }
            }
            // an event they joined: what the council does, and how many are in (no owner among them)
            if (booking.isEvent) item {
                HorizontalDivider(Modifier.padding(horizontal = 16.dp), color = Lq.colors.border)
                Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
                    Text(
                        booking.publicDescription.ifBlank { stringResource(R.string.event_no_description) },
                        style = MaterialTheme.typography.bodyMedium,
                        color = if (booking.publicDescription.isBlank()) Lq.colors.mutedForeground else Lq.colors.foreground,
                    )
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Icon(painterResource(R.drawable.ic_users), contentDescription = null, tint = Lq.colors.foreground, modifier = Modifier.size(22.dp))
                        Spacer(Modifier.size(16.dp))
                        Text(stringResource(R.string.people_count, booking.people, booking.capacity), style = MaterialTheme.typography.bodyLarge, color = Lq.colors.foreground)
                    }
                }
            }
            // the details: drag the sheet up to read them
            item {
                HorizontalDivider(Modifier.padding(horizontal = 16.dp), color = Lq.colors.border)
                Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
                    DetailBlock(R.drawable.ic_map_pin, stringResource(R.string.directions)) {
                        Text(f.address.ifBlank { stringResource(R.string.no_location) }, style = MaterialTheme.typography.bodyMedium, color = Lq.colors.mutedForeground)
                    }
                    DetailBlock(R.drawable.ic_clipboard_list, stringResource(R.string.conditions)) {
                        if (f.slotMinutes > 0) Bullet(pluralStringResource(R.plurals.rule_slot, f.slotMinutes, f.slotMinutes))
                        Bullet(pluralStringResource(R.plurals.rule_cancel, f.minCancelHours, f.minCancelHours))
                        f.minAge?.let { Bullet(stringResource(R.string.min_age, it)) }
                        if (f.residentsOnly) Bullet(stringResource(R.string.residents_only_short))
                        booking.cancelUntil?.let { if (booking.owner) Bullet(stringResource(R.string.cancel_until, "${bookingDayInline(it)} ${time(it)}")) }
                    }
                    if (f.accessibilityInfo.isNotBlank()) DetailBlock(R.drawable.ic_accessibility, stringResource(R.string.accessibility)) {
                        Text(f.accessibilityInfo, style = MaterialTheme.typography.bodyMedium, color = Lq.colors.mutedForeground)
                    }
                    DetailBlock(R.drawable.ic_scroll_text, stringResource(R.string.usage_rules)) {
                        Text(f.usageRules.ifBlank { stringResource(R.string.no_rules) }, style = MaterialTheme.typography.bodyMedium, color = Lq.colors.mutedForeground)
                    }
                }
            }
            // last, apart, in red
            if ((booking.owner && booking.canCancel) || (!booking.owner && booking.canLeave)) item {
                HorizontalDivider(Modifier.padding(horizontal = 16.dp), color = Lq.colors.border)
                Box(Modifier.padding(16.dp)) {
                    if (booking.owner) DangerOutline(stringResource(R.string.cancel_confirm), onCancel)
                    else DangerOutline(stringResource(R.string.leave_confirm), onLeave)
                }
            }
        }
    }
}

@Composable
private fun RoundAction(@DrawableRes icon: Int, label: String, onClick: () -> Unit) {
    Column(
        Modifier.width(76.dp).clip(RoundedCornerShape(12.dp)).clickable(onClick = onClick).padding(vertical = 4.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Box(Modifier.size(56.dp).clip(CircleShape).background(Lq.colors.brandBlue), contentAlignment = Alignment.Center) {
            Icon(painterResource(icon), contentDescription = null, tint = Lq.colors.onBrandBlue, modifier = Modifier.size(24.dp))
        }
        Spacer(Modifier.height(6.dp))
        Text(label, style = MaterialTheme.typography.bodySmall, color = Lq.colors.foreground, textAlign = TextAlign.Center, maxLines = 2)
    }
}

/** A red outlined button: destructive, but not shouting. */
@Composable
private fun DangerOutline(text: String, onClick: () -> Unit) {
    androidx.compose.material3.OutlinedButton(
        onClick = onClick,
        modifier = Modifier.fillMaxWidth().height(48.dp),
        shape = RoundedCornerShape(6.dp),
        border = androidx.compose.foundation.BorderStroke(1.dp, Lq.colors.destructive),
    ) { Text(text, color = Lq.colors.destructive, style = MaterialTheme.typography.labelLarge) }
}


@Composable
private fun HistoryRow(b: Booking, onRate: () -> Unit) {
    Row(Modifier.fillMaxWidth().padding(vertical = 14.dp), verticalAlignment = Alignment.CenterVertically) {
        Column(Modifier.weight(1f)) {
            Text(if (b.isEvent) "${b.title} · ${b.facilityName}" else b.facilityName, style = MaterialTheme.typography.bodyLarge, color = Lq.colors.foreground, maxLines = 1, overflow = TextOverflow.Ellipsis)
            Text("${bookingDay(b.startsAt)} · ${time(b.startsAt)}", style = MaterialTheme.typography.bodySmall, color = Lq.colors.mutedForeground)
            // ended, not a no-show, within 30 days: the stars, in a sheet (as the web's «Valorar»)
            if (b.canRate) Row(
                Modifier.padding(top = 4.dp).clip(RoundedCornerShape(8.dp)).clickable(onClick = onRate)
                    .heightIn(min = 40.dp).padding(end = 8.dp),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(6.dp),
            ) {
                Icon(painterResource(R.drawable.ic_star), contentDescription = null, tint = Lq.colors.primary, modifier = Modifier.size(18.dp))
                Text(stringResource(R.string.rating_button), style = MaterialTheme.typography.labelLarge, color = Lq.colors.primary)
            }
        }
        val (label, positive) = when {
            b.cancelled -> R.string.status_cancelled to false
            b.attendance == "attended" -> R.string.status_attended to true
            b.attendance == "no_show" -> R.string.status_no_show to false
            else -> R.string.status_ended to false
        }
        Pill(
            stringResource(label),
            if (positive) Lq.colors.pillPositiveBg else Lq.colors.pillNeutralBg,
            if (positive) Lq.colors.pillPositiveInk else Lq.colors.pillNeutralInk,
        )
    }
}


@Composable
private fun locale(): Locale = LocalConfiguration.current.locales[0]

/** "Hoxe" is not in the web's strings: the date reads in full, in the app's language. */
@Composable
internal fun bookingDay(instant: Instant): String {
    val date = instant.atZone(ZONE).toLocalDate()
    val pattern = if (date.year == LocalDate.now(ZONE).year) "EEEE d MMMM" else null
    val formatter = pattern?.let { DateTimeFormatter.ofPattern(it, locale()) }
        ?: DateTimeFormatter.ofLocalizedDate(FormatStyle.MEDIUM).withLocale(locale())
    return date.format(formatter).replaceFirstChar { it.titlecase(locale()) }
}

/** The day inside a sentence: «martes 3 novembro», in lower case where the language writes it so. */
@Composable
internal fun bookingDayInline(instant: Instant): String {
    val date = instant.atZone(ZONE).toLocalDate()
    return date.format(DateTimeFormatter.ofPattern("EEEE d MMMM", locale()))
}

@Composable
internal fun time(instant: Instant): String = instant.atZone(ZONE).format(DateTimeFormatter.ofPattern("HH:mm", locale()))
