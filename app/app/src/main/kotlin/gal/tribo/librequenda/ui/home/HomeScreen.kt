package gal.tribo.librequenda.ui.home

import gal.tribo.librequenda.ui.search.FacilitySearch

import android.Manifest
import gal.tribo.librequenda.ui.components.Pill
import androidx.compose.material3.HorizontalDivider
import androidx.annotation.DrawableRes
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.Icon
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.style.TextAlign
import gal.tribo.librequenda.data.Booking
import gal.tribo.librequenda.data.DayFacility
import gal.tribo.librequenda.data.PublicGame
import gal.tribo.librequenda.ui.search.SearchFilters
import gal.tribo.librequenda.ui.search.TimeRange
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.pluralStringResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import coil.compose.AsyncImage
import gal.tribo.librequenda.HomeState
import gal.tribo.librequenda.R
import gal.tribo.librequenda.SessionState
import gal.tribo.librequenda.data.FacilityCard
import gal.tribo.librequenda.data.GeoPoint
import gal.tribo.librequenda.data.Recommendation
import gal.tribo.librequenda.data.currentLocation
import gal.tribo.librequenda.ui.components.PrimaryButton
import gal.tribo.librequenda.ui.theme.Lq
import gal.tribo.librequenda.ui.theme.onBrandBlue
import kotlinx.coroutines.launch
import java.time.Instant
import java.time.LocalDate
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.util.Locale


/**
 * Home: their next booking first, when there is one; what they still need to book
 * (date of birth, a confirmed email); a greeting with quick ways into the search;
 * then «For you» and the public bookings (only when there are some) as carousels.
 * Signed out, none of that: the facilities alone, as on the search page, to look
 * through (booking asks to sign in).
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun HomeScreen(
    home: HomeState,
    session: SessionState,
    nextBooking: Booking?,
    onRefresh: () -> Unit,
    onLocation: (GeoPoint) -> Unit,
    onSignIn: () -> Unit,
    onSearch: (SearchFilters) -> Unit,
    onProfile: () -> Unit,
    onBookings: () -> Unit,
    onFacility: (FacilityCard, Instant?) -> Unit,
    browse: SearchFilters,
    onBrowse: (SearchFilters) -> Unit,
    onBrowseFacility: (DayFacility, Instant?) -> Unit,
    modifier: Modifier = Modifier,
) {
    LocationRequest(onLocation)
    if (!session.signedIn) {
        FacilitySearch(
            filters = browse,
            onFilters = onBrowse,
            councils = home.facilities.map { it.councilSlug to it.councilName }.distinct(),
            location = home.location,
            onSlot = onBrowseFacility,
            modifier = modifier.padding(top = 16.dp),
        )
        return
    }
    val onOpen: (FacilityCard, Instant?) -> Unit = onFacility

    PullToRefreshBox(
        isRefreshing = home.loading && home.facilities.isNotEmpty(),
        onRefresh = onRefresh,
        modifier = modifier.fillMaxSize(),
    ) {
        LazyColumn(
            modifier = Modifier.fillMaxSize(),
            contentPadding = PaddingValues(top = 20.dp, bottom = 40.dp),
        ) {
            nextBooking?.let { b -> item { NextBooking(b, onBookings) } }
            session.me?.let { me ->
                // booking needs both, as on the web (ProfileReminder)
                if (!me.profileComplete) item {
                    Reminder(stringResource(R.string.incomplete_title), stringResource(R.string.incomplete_body), stringResource(R.string.incomplete_action), onProfile)
                }
                if (!me.emailVerified) item {
                    Reminder(stringResource(R.string.unverified_title), stringResource(R.string.unverified_body), stringResource(R.string.unverified_action), onProfile)
                }
            }
            item { Greeting(session, onSearch) }
            // the councils' events come first, whenever there are any: always shown, not a suggestion
            item { EventsSection(home, onOpen) }
            // «For you» is theirs to turn off (Preferences), as on the web
            if (session.me?.showRecommendations != false) item { ForYouSection(home, session, onOpen) }
            item { PublicSection(home, onOpen) }
        }
    }
}

/** Asks once for the rough location (distances, public bookings nearby); fine without it. */
@Composable
private fun LocationRequest(onLocation: (GeoPoint) -> Unit) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val launcher = rememberLauncherForActivityResult(ActivityResultContracts.RequestPermission()) { granted ->
        if (granted) scope.launch { currentLocation(context)?.let(onLocation) }
    }
    LaunchedEffect(Unit) {
        val known = currentLocation(context)
        if (known != null) onLocation(known) else launcher.launch(Manifest.permission.ACCESS_COARSE_LOCATION)
    }
}

@Composable
private fun SectionHeader(title: String, help: String) {
    Column(Modifier.padding(horizontal = 16.dp).padding(bottom = 12.dp)) {
        Text(title, style = MaterialTheme.typography.headlineSmall, color = Lq.colors.foreground, modifier = Modifier.semantics { heading() })
        Spacer(Modifier.height(2.dp))
        Text(help, style = MaterialTheme.typography.bodyMedium, color = Lq.colors.mutedForeground)
    }
}

@Composable
private fun ForYouSection(home: HomeState, session: SessionState, onOpen: (FacilityCard, Instant?) -> Unit) {
    val suggestions = home.recommendations.filterNot { it.isPublicBooking }
    // signed out, or nothing of theirs yet: the facilities as the API sorts them (nearest first)
    val fallback = home.facilities
    if (suggestions.isEmpty() && fallback.isEmpty()) return

    SectionHeader(
        stringResource(R.string.for_you_title),
        stringResource(if (suggestions.isNotEmpty()) R.string.for_you_help else R.string.for_you_help_signed_out),
    )
    Carousel {
        if (suggestions.isNotEmpty()) {
            items(suggestions, key = { "r-" + it.facility.id + it.kind }) { r ->
                FacilityCarouselCard(r.facility, home.location, badge = r.startsAt?.let { stringResource(R.string.free_at, format(it)) }, onClick = { onOpen(r.facility, r.startsAt) })
            }
        } else {
            items(fallback.take(10), key = { it.id }) { f ->
                FacilityCarouselCard(f, home.location, badge = pluralStringResource(R.plurals.free_today, f.freeToday, f.freeToday), onClick = { onOpen(f, null) })
            }
        }
    }
    Spacer(Modifier.height(28.dp))
}

/** Every public booking with spots left today, the nearest first, then the soonest; nothing when there are none. */
@Composable
private fun PublicSection(home: HomeState, onOpen: (FacilityCard, Instant?) -> Unit) {
    val here = home.location
    val now = Instant.now()
    // the facilities come nearest first from the API, each one's games by time; events have their own section
    val games = home.today.flatMap { f -> f.publicGames.filter { it.startsAt.isAfter(now) && !it.isEvent }.map { f to it } }
    if (games.isEmpty()) return
    SectionHeader(stringResource(R.string.public_title), stringResource(R.string.public_help))
    Carousel {
        items(games, key = { it.second.id }) { (f, game) ->
            val card = home.facilities.firstOrNull { it.id == f.id } ?: FacilityCard(
                f.id, f.slug, f.name, f.imageUrl, f.latitude, f.longitude, f.freeSlots.size, f.councilSlug, f.councilName, f.distanceKm,
            )
            val spots = pluralStringResource(R.plurals.spots_left, game.spotsLeft, game.spotsLeft)
            // the council's event says what it is
            val badge = listOfNotNull(game.title.takeIf { game.isEvent }, format(game.startsAt), spots).joinToString(" · ")
            FacilityCarouselCard(card, here, badge = badge, onClick = { onOpen(card, game.startsAt) })
        }
    }
    Spacer(Modifier.height(28.dp))
}

/**
 * The councils' events that have not started, soonest first, as the API sends them:
 * the event's title on the facility's photo, when, and the places left. Tapping one
 * opens the facility at that time, where joining is one step; full ones still show.
 */
@Composable
private fun EventsSection(home: HomeState, onOpen: (FacilityCard, Instant?) -> Unit) {
    if (home.events.isEmpty()) return
    SectionHeader(stringResource(R.string.events_title), stringResource(R.string.events_help))
    Carousel {
        items(home.events, key = { it.id }) { event ->
            val spots = if (event.spotsLeft > 0) pluralStringResource(R.plurals.spots_left, event.spotsLeft, event.spotsLeft) else stringResource(R.string.event_full)
            FacilityCarouselCard(
                event.facility, home.location,
                badge = "${formatDay(event.startsAt)} · $spots",
                title = event.title,
                onClick = { onOpen(event.facility, event.startsAt) },
            )
        }
    }
    Spacer(Modifier.height(28.dp))
}

/** Their next booking, at the top while there is one: where, when, and on to their bookings. */
@Composable
private fun NextBooking(b: Booking, onClick: () -> Unit) {
    val locale = locale()
    val at = b.startsAt.atZone(ZoneId.systemDefault())
    val today = LocalDate.now()
    val day = when (at.toLocalDate()) {
        today -> stringResource(R.string.today)
        today.plusDays(1) -> stringResource(R.string.tomorrow)
        else -> at.format(DateTimeFormatter.ofPattern("EEEE d MMMM", locale)).replaceFirstChar { it.titlecase(locale) }
    }
    val time = DateTimeFormatter.ofPattern("HH:mm", locale)
    Row(
        Modifier
            .padding(horizontal = 16.dp)
            .padding(bottom = 20.dp)
            .fillMaxWidth()
            .clip(RoundedCornerShape(16.dp))
            .background(Lq.colors.brandBlue) // the logo's blue
            .clickable(onClick = onClick)
            .padding(12.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        AsyncImage(b.imageUrl, contentDescription = null, contentScale = ContentScale.Crop, modifier = Modifier.size(64.dp).clip(RoundedCornerShape(12.dp)).background(Lq.colors.accent))
        Spacer(Modifier.width(14.dp))
        Column(Modifier.weight(1f)) {
            Text(stringResource(R.string.next_booking), color = Lq.colors.onBrandBlue.copy(alpha = 0.9f), style = MaterialTheme.typography.labelMedium)
            Text(b.facilityName, color = Lq.colors.onBrandBlue, style = MaterialTheme.typography.titleMedium, maxLines = 1, overflow = TextOverflow.Ellipsis)
            Text("$day · ${at.format(time)}–${b.endsAt.atZone(ZoneId.systemDefault()).format(time)}", color = Lq.colors.onBrandBlue, style = MaterialTheme.typography.bodyMedium.copy(fontWeight = androidx.compose.ui.text.font.FontWeight.Medium))
        }
        Icon(painterResource(R.drawable.ic_chevron_right), contentDescription = null, tint = Lq.colors.onBrandBlue, modifier = Modifier.size(20.dp))
    }
}

/** What they still need before booking, as the web's banner, with the way to fix it. */
@Composable
internal fun Reminder(title: String, body: String, action: String, onClick: () -> Unit) {
    Column(
        Modifier
            .padding(horizontal = 16.dp)
            .padding(bottom = 16.dp)
            .fillMaxWidth()
            .clip(RoundedCornerShape(12.dp))
            .background(Lq.colors.pillNegativeBg)
            .clickable(onClick = onClick)
            .padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(4.dp),
    ) {
        Text(title, color = Lq.colors.pillNegativeInk, style = MaterialTheme.typography.titleSmall)
        Text(body, color = Lq.colors.pillNegativeInk, style = MaterialTheme.typography.bodyMedium)
        Text("$action ›", color = Lq.colors.pillNegativeInk, style = MaterialTheme.typography.labelLarge, modifier = Modifier.padding(top = 4.dp))
    }
}

/** «Hi …, ready to play?» and round buttons that open the search with filters set. */
@Composable
private fun Greeting(session: SessionState, onSearch: (SearchFilters) -> Unit) {
    val first = session.me?.fullName?.split(" ")?.firstOrNull()?.takeIf { it.isNotBlank() }
    Column(Modifier.padding(horizontal = 16.dp).padding(bottom = 28.dp)) {
        Text(
            if (first != null) stringResource(R.string.greeting_named, first) else stringResource(R.string.greeting),
            style = MaterialTheme.typography.headlineSmall,
            color = Lq.colors.foreground,
        )
        Spacer(Modifier.height(16.dp))
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
            QuickAction(R.drawable.ic_calendar_plus, stringResource(R.string.quick_book)) { onSearch(SearchFilters()) }
            QuickAction(R.drawable.ic_navigation, stringResource(R.string.quick_near)) { onSearch(SearchFilters()) } // sorted by distance when the phone shares it
            QuickAction(R.drawable.ic_clock, stringResource(R.string.quick_now)) { onSearch(SearchFilters(time = TimeRange.Now)) }
            QuickAction(R.drawable.ic_users, stringResource(R.string.quick_join)) { onSearch(SearchFilters(publicOnly = true)) }
        }
    }
}

@Composable
private fun QuickAction(@DrawableRes icon: Int, label: String, onClick: () -> Unit) {
    Column(
        Modifier.width(80.dp).clip(RoundedCornerShape(12.dp)).clickable(onClick = onClick).padding(vertical = 4.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        // the logo's blue, solid, the icon white
        Box(Modifier.size(60.dp).clip(CircleShape).background(Lq.colors.brandBlue), contentAlignment = Alignment.Center) {
            Icon(painterResource(icon), contentDescription = null, tint = Lq.colors.onBrandBlue, modifier = Modifier.size(26.dp))
        }
        Spacer(Modifier.height(8.dp))
        Text(label, style = MaterialTheme.typography.bodySmall, color = Lq.colors.foreground, textAlign = TextAlign.Center, maxLines = 2)
    }
}

@Composable
private fun Carousel(content: androidx.compose.foundation.lazy.LazyListScope.() -> Unit) {
    LazyRow(
        contentPadding = PaddingValues(horizontal = 16.dp),
        horizontalArrangement = Arrangement.spacedBy(12.dp),
        content = content,
    )
}

/** The facility's photo with its name, distance and council on it; an event's title over them. */
@Composable
private fun FacilityCarouselCard(f: FacilityCard, here: GeoPoint?, badge: String?, title: String? = null, onClick: () -> Unit) {
    // how far, as the API worked it out
    val distance = f.distanceKm?.let { stringResource(R.string.distance_km, formatKm(it)) }
    Box(
        Modifier
            .width(264.dp)
            .height(172.dp)
            .clip(RoundedCornerShape(12.dp))
            .background(Lq.colors.muted)
            .clickable(onClick = onClick),
    ) {
        AsyncImage(
            model = f.imageUrl,
            contentDescription = null,
            contentScale = ContentScale.Crop,
            modifier = Modifier.fillMaxSize(),
        )
        // the text stays readable on any photo
        Box(
            Modifier
                .fillMaxSize()
                .background(Brush.verticalGradient(0.4f to Color.Transparent, 1f to Color.Black.copy(alpha = 0.78f)))
        )
        if (badge != null) {
            Text(
                badge,
                color = Lq.colors.pillPositiveInk,
                fontSize = 12.sp,
                fontWeight = FontWeight.SemiBold,
                modifier = Modifier
                    .align(Alignment.TopStart)
                    .padding(10.dp)
                    .clip(RoundedCornerShape(50))
                    .background(Lq.colors.pillPositiveBg)
                    .padding(horizontal = 10.dp, vertical = 4.dp),
            )
        }
        Column(
            Modifier.align(Alignment.BottomStart).padding(horizontal = 14.dp, vertical = 12.dp),
            verticalArrangement = Arrangement.spacedBy(2.dp),
        ) {
            Text(
                title ?: f.name,
                color = Color.White,
                style = MaterialTheme.typography.titleMedium,
                maxLines = 2,
                overflow = TextOverflow.Ellipsis,
            )
            Text(
                listOfNotNull(f.name.takeIf { title != null }, distance, f.councilName).joinToString(" · "),
                color = Color.White.copy(alpha = 0.85f),
                style = MaterialTheme.typography.bodySmall,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
        }
    }
}

/** The app's language right now (it can differ from the phone's). */
@Composable
private fun locale(): Locale = LocalConfiguration.current.locales[0]

@Composable
private fun formatKm(km: Double): String =
    if (km < 10) String.format(locale(), "%.1f", km) else km.toInt().toString()

/** "18:00" today, "mar 18:00" this week, "21 nov 18:00" further ahead: events are planned weeks before. */
@Composable
private fun formatDay(instant: Instant): String {
    val at = instant.atZone(ZoneId.systemDefault())
    val days = java.time.temporal.ChronoUnit.DAYS.between(LocalDate.now(), at.toLocalDate())
    val pattern = when {
        days == 0L -> "HH:mm"
        days < 7 -> "EEE HH:mm"
        else -> "d MMM HH:mm"
    }
    return at.format(DateTimeFormatter.ofPattern(pattern, locale()))
}

/** "18:00" today, "mar 18:00" another day, in the phone's time zone and the app's language. */
@Composable
private fun format(instant: Instant): String {
    val at = instant.atZone(ZoneId.systemDefault())
    val pattern = if (at.toLocalDate() == LocalDate.now()) "HH:mm" else "EEE HH:mm"
    return at.format(DateTimeFormatter.ofPattern(pattern, locale()))
}
