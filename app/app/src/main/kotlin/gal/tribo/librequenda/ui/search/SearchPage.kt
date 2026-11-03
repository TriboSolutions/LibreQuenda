package gal.tribo.librequenda.ui.search

import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.rememberCoroutineScope
import gal.tribo.librequenda.data.PublicGame
import gal.tribo.librequenda.data.joinPublicGame
import gal.tribo.librequenda.ui.components.GhostButton
import gal.tribo.librequenda.ui.components.ConfirmSheet
import gal.tribo.librequenda.ui.components.MultiChoiceSheet
import gal.tribo.librequenda.ui.components.DateSheet
import gal.tribo.librequenda.ui.components.SingleChoiceSheet
import gal.tribo.librequenda.ui.profile.ProfileActions
import gal.tribo.librequenda.ui.profile.Texts
import gal.tribo.librequenda.ui.profile.userMessage
import kotlinx.coroutines.launch
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
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Text
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.pluralStringResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import coil.compose.AsyncImage
import gal.tribo.librequenda.R
import gal.tribo.librequenda.data.DayFacility
import gal.tribo.librequenda.data.GeoPoint
import gal.tribo.librequenda.data.TagCategory
import gal.tribo.librequenda.data.fetchFacilitiesDay
import gal.tribo.librequenda.data.DayQuery
import gal.tribo.librequenda.data.fetchCouncils
import androidx.compose.foundation.lazy.itemsIndexed
import gal.tribo.librequenda.data.fetchTags
import gal.tribo.librequenda.ui.components.PrimaryButton
import gal.tribo.librequenda.ui.components.SecondaryButton
import gal.tribo.librequenda.ui.theme.Lq
import kotlinx.coroutines.CancellationException
import java.time.Instant
import java.time.LocalDate
import java.time.LocalTime
import java.time.ZoneId
import java.time.format.DateTimeFormatter

// The councils are all in Galicia today, as on the web
private val ZONE = ZoneId.of("Europe/Madrid")

enum class TimeRange(val from: LocalTime?, val until: LocalTime?) {
    Any(null, null),
    Now(null, null), // from this moment, today
    Morning(null, LocalTime.of(14, 0)),
    Afternoon(LocalTime.of(14, 0), LocalTime.of(20, 0)),
    Evening(LocalTime.of(20, 0), null),
}

/** What the search filters by; the Home's buttons open it with some already set. */
data class SearchFilters(
    val query: String = "",
    val sport: String? = null, // a tag of the "sports" category
    val tags: Set<String> = emptySet(), // type of facility, services…
    val council: String? = null,
    val date: String = LocalDate.now(ZONE).toString(),
    val time: TimeRange = TimeRange.Any,
    val publicOnly: Boolean = false, // the public bookings to join, not the free turns
) {
    val day: LocalDate get() = LocalDate.parse(date)
}

/**
 * The facility search, a page of its own: the search box, the filters as chips
 * (sport, day, time; the rest behind the filters button) and every facility that
 * matches, each with its free turns that day.
 */
@OptIn(ExperimentalLayoutApi::class)
@Composable
fun SearchPage(
    open: Boolean,
    filters: SearchFilters,
    onFilters: (SearchFilters) -> Unit,
    councils: List<Pair<String, String>>, // slug to name
    location: GeoPoint?,
    onBack: () -> Unit,
    onSignIn: () -> Unit,
    onSlot: (DayFacility, Instant?) -> Unit, // a turn, or null for the facility itself
) {
    BackHandler(enabled = open, onBack = onBack)
    AnimatedVisibility(
        visible = open,
        enter = slideInHorizontally(tween(260)) { it },
        exit = slideOutHorizontally(tween(220)) { it },
    ) {
        Column(
            Modifier
                .fillMaxSize()
                .background(Lq.colors.card)
                .clickable(interactionSource = null, indication = null) {} // taps stay on the page
                .statusBarsPadding()
        ) {
            // bar: back, the title in the middle
            Box(Modifier.fillMaxWidth().height(56.dp)) {
                IconButton(onClick = onBack, modifier = Modifier.align(Alignment.CenterStart).padding(start = 4.dp)) {
                    Icon(painterResource(R.drawable.ic_arrow_left), contentDescription = stringResource(R.string.back), tint = Lq.colors.foreground)
                }
                Text(
                    stringResource(R.string.facilities_title),
                    style = MaterialTheme.typography.titleMedium,
                    color = Lq.colors.foreground,
                    modifier = Modifier.align(Alignment.Center).semantics { heading() },
                )
            }
            FacilitySearch(filters, onFilters, councils, location, onSlot)
        }
    }
}

/**
 * The search itself: the box, the chips and every facility that matches, a page
 * at a time. The search page shows it under its bar; signed out, it is the home.
 */
@Composable
fun FacilitySearch(
    filters: SearchFilters,
    onFilters: (SearchFilters) -> Unit,
    councils: List<Pair<String, String>>, // slug to name
    location: GeoPoint?,
    onSlot: (DayFacility, Instant?) -> Unit, // a turn, or null for the facility itself
    modifier: Modifier = Modifier,
) {
    var results by remember { mutableStateOf<List<DayFacility>?>(null) }
    var total by remember { mutableIntStateOf(0) }
    var nextPage by remember { mutableStateOf<Int?>(null) } // the API has more: the next page, loaded at the list's end
    var loadingMore by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()
    var failed by remember { mutableStateOf(false) }
    var catalogue by remember { mutableStateOf<List<TagCategory>>(emptyList()) }
    var sheet by remember { mutableStateOf(false) }

    var councilList by remember { mutableStateOf(councils) } // every council (the facilities come a page at a time)
    LaunchedEffect(Unit) { runCatching { fetchTags() }.onSuccess { catalogue = it } }
    LaunchedEffect(Unit) { runCatching { fetchCouncils() }.onSuccess { councilList = it } }
    // the API filters, sorts (nearest first) and pages; typing waits a moment before asking
    val query = filters.toQuery(location)
    LaunchedEffect(query) {
        if (query.words.isNotEmpty()) kotlinx.coroutines.delay(300)
        failed = false; results = null; nextPage = null
        try {
            val page = fetchFacilitiesDay(query)
            results = page.items; total = page.total; nextPage = if (page.hasMore) page.page + 1 else null
        } catch (c: CancellationException) { throw c } catch (_: Exception) { failed = true }
    }
    fun loadMore() {
        val next = nextPage ?: return
        if (loadingMore) return
        loadingMore = true
        scope.launch {
            try {
                val page = fetchFacilitiesDay(query, next)
                results = results.orEmpty() + page.items; nextPage = if (page.hasMore) page.page + 1 else null
            } catch (c: CancellationException) { throw c } catch (_: Exception) { /* the next scroll tries again */ }
            loadingMore = false
        }
    }
    val shown = results

    Column(modifier.fillMaxSize()) {
        SearchBox(filters.query) { onFilters(filters.copy(query = it)) }
        Spacer(Modifier.height(12.dp))
        ChipsRow(filters, catalogue, councilList, onFilters, onMore = { sheet = true })
        Spacer(Modifier.height(8.dp))

        when {
            failed -> Text(stringResource(R.string.search_load_error), color = Lq.colors.destructive, modifier = Modifier.padding(16.dp))
            shown == null -> LinearProgressIndicator(Modifier.fillMaxWidth().padding(16.dp))
            shown.isEmpty() -> Text(stringResource(R.string.search_no_results), color = Lq.colors.mutedForeground, modifier = Modifier.padding(16.dp))
            else -> LazyColumn(Modifier.fillMaxSize(), contentPadding = PaddingValues(bottom = 40.dp)) {
                item {
                    Text(
                        pluralStringResource(R.plurals.results_count, total, total),
                        style = MaterialTheme.typography.bodySmall,
                        color = Lq.colors.mutedForeground,
                        modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp),
                    )
                }
                itemsIndexed(shown, key = { _, f -> f.id }) { i, f ->
                    // near the end of what came, the next page
                    if (i >= shown.size - 5) LaunchedEffect(shown.size) { loadMore() }
                    // a public booking opens the facility with it chosen, to join it there
                    ResultRow(f, filters, onOpen = { onSlot(f, null) }, onSlot = { onSlot(f, it) }, onGame = { game -> onSlot(f, game.startsAt) })
                }
                if (nextPage != null) item { LinearProgressIndicator(Modifier.fillMaxWidth().padding(16.dp)) }
            }
        }
    }

    if (sheet) FiltersSheet(filters, catalogue, councilList, onApply = { onFilters(it); sheet = false }, onDismiss = { sheet = false })
}

/** The filters as the API takes them: «from now» is the time now, in the councils' zone. */
private fun SearchFilters.toQuery(near: GeoPoint?): DayQuery {
    val clock = DateTimeFormatter.ofPattern("HH:mm")
    val (from, until) = when (time) {
        TimeRange.Any -> null to null
        TimeRange.Now -> java.time.ZonedDateTime.now(ZONE).format(clock).takeIf { day == LocalDate.now(ZONE) } to null
        else -> time.from?.format(clock) to time.until?.format(clock)
    }
    return DayQuery(day, near, query, council, sport, tags, from, until, publicOnly)
}

@Composable
private fun SearchBox(value: String, onChange: (String) -> Unit) {
    Row(
        Modifier.padding(horizontal = 16.dp).fillMaxWidth().height(52.dp).clip(RoundedCornerShape(26.dp)).background(Lq.colors.muted).padding(horizontal = 16.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(painterResource(R.drawable.ic_search), contentDescription = null, tint = Lq.colors.foreground, modifier = Modifier.size(22.dp))
        Spacer(Modifier.width(12.dp))
        Box(Modifier.weight(1f)) {
            if (value.isEmpty()) Text(stringResource(R.string.search_placeholder), color = Lq.colors.mutedForeground, style = MaterialTheme.typography.bodyLarge, maxLines = 1, overflow = TextOverflow.Ellipsis)
            BasicTextField(
                value = value,
                onValueChange = onChange,
                singleLine = true,
                textStyle = MaterialTheme.typography.bodyLarge.copy(color = Lq.colors.foreground),
                cursorBrush = SolidColor(Lq.colors.primary),
                modifier = Modifier.fillMaxWidth(),
            )
        }
        if (value.isNotEmpty()) {
            Icon(painterResource(R.drawable.ic_x), contentDescription = stringResource(R.string.clear_filters), tint = Lq.colors.mutedForeground,
                modifier = Modifier.size(20.dp).clickable { onChange("") })
        }
    }
}

/** Which chip's sheet is open. */
private sealed interface ChipSheet {
    data object Day : ChipSheet
    data object Time : ChipSheet
    data object Mode : ChipSheet
    data object Sport : ChipSheet
    data object Council : ChipSheet
    data class Category(val category: TagCategory) : ChipSheet
}

/**
 * The filters button, then the chips: day and time always; every other filter only
 * once it is set (from the filters sheet, or by whoever opened the search). A chip
 * shows its value and opens a sheet to change or remove just that filter.
 */
@Composable
private fun ChipsRow(
    f: SearchFilters,
    catalogue: List<TagCategory>,
    councils: List<Pair<String, String>>,
    onFilters: (SearchFilters) -> Unit,
    onMore: () -> Unit,
) {
    var open by remember { mutableStateOf<ChipSheet?>(null) }
    val sports = catalogue.firstOrNull()?.tags.orEmpty() // the catalogue's first category is the sports
    val others = catalogue.drop(1)
    val times = timeOptions()

    LazyRow(contentPadding = PaddingValues(start = 8.dp, end = 16.dp), horizontalArrangement = Arrangement.spacedBy(6.dp), verticalAlignment = Alignment.CenterVertically) {
        // the active filters are the chips themselves: no count here
        item {
            IconButton(onClick = onMore) {
                Icon(painterResource(R.drawable.ic_sliders), contentDescription = stringResource(R.string.filters), tint = Lq.colors.foreground)
            }
        }
        item { Chip(dayLabel(f.day)) { open = ChipSheet.Day } }
        item { Chip(times.first { it.first == f.time }.second) { open = ChipSheet.Time } }
        if (f.publicOnly) item { Chip(stringResource(R.string.public_title), R.drawable.ic_users) { open = ChipSheet.Mode } }
        f.sport?.let { slug -> item { Chip(sports.firstOrNull { it.slug == slug }?.name ?: slug) { open = ChipSheet.Sport } } }
        f.council?.let { slug -> item { Chip(councils.firstOrNull { it.first == slug }?.second ?: slug) { open = ChipSheet.Council } } }
        others.forEach { category ->
            val chosen = category.tags.filter { it.slug in f.tags }
            if (chosen.isNotEmpty()) item(key = category.name) { Chip(chosen.joinToString(", ") { it.name }) { open = ChipSheet.Category(category) } }
        }
    }

    val apply = stringResource(R.string.apply)
    when (val sheet = open) {
        null -> Unit
        ChipSheet.Day -> {
            val today = LocalDate.now(ZONE)
            DateSheet(stringResource(R.string.day), f.day, from = today, until = today.plusDays(90), { onFilters(f.copy(date = it.toString())) }, { open = null })
        }
        ChipSheet.Time -> SingleChoiceSheet(stringResource(R.string.time_of_day), times, f.time, { onFilters(f.copy(time = it)) }, { open = null })
        ChipSheet.Mode -> SingleChoiceSheet(
            stringResource(R.string.show),
            listOf(false to stringResource(R.string.free_turns), true to stringResource(R.string.public_title)),
            f.publicOnly, { onFilters(f.copy(publicOnly = it)) }, { open = null },
        )
        ChipSheet.Sport -> SingleChoiceSheet(
            stringResource(R.string.sport),
            listOf<Pair<String?, String>>(null to stringResource(R.string.any_sport)) + sports.map { it.slug to it.name },
            f.sport, { onFilters(f.copy(sport = it)) }, { open = null },
        )
        ChipSheet.Council -> SingleChoiceSheet(
            stringResource(R.string.council),
            listOf<Pair<String?, String>>(null to stringResource(R.string.all_councils)) + councils,
            f.council, { onFilters(f.copy(council = it)) }, { open = null },
        )
        is ChipSheet.Category -> {
            val slugs = sheet.category.tags.map { it.slug }.toSet()
            MultiChoiceSheet(
                sheet.category.name, sheet.category.tags.map { it.slug to it.name }, f.tags intersect slugs,
                onApply = { onFilters(f.copy(tags = (f.tags - slugs) + it)) }, onDismiss = { open = null }, applyLabel = apply,
            )
        }
    }
}

/** «Today», «Tomorrow», or the date, in the app's language. */
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
private fun timeOptions() = listOf(
    TimeRange.Any to stringResource(R.string.time_any),
    TimeRange.Now to stringResource(R.string.time_now),
    TimeRange.Morning to stringResource(R.string.time_morning),
    TimeRange.Afternoon to stringResource(R.string.time_afternoon),
    TimeRange.Evening to stringResource(R.string.time_evening),
)

/** A chosen filter: its value on the brand blue, a chevron, a sheet to change it. */
@Composable
private fun Chip(label: String, icon: Int? = null, onClick: () -> Unit) {
    Row(
        Modifier
            .height(32.dp)
            .clip(RoundedCornerShape(16.dp))
            .background(Lq.colors.primary)
            .clickable(role = Role.Button, onClick = onClick)
            .padding(start = 12.dp, end = 10.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        val ink = Lq.colors.primaryForeground
        if (icon != null) {
            Icon(painterResource(icon), contentDescription = null, tint = ink, modifier = Modifier.size(14.dp))
            Spacer(Modifier.width(5.dp))
        }
        Text(label, color = ink, style = MaterialTheme.typography.labelMedium.copy(fontSize = 13.sp), maxLines = 1)
        Spacer(Modifier.width(4.dp))
        Icon(painterResource(R.drawable.ic_chevron_down), contentDescription = null, tint = ink, modifier = Modifier.size(14.dp))
    }
}

/** A result: the photo edge to edge with the name on it, where it is, then its free turns as chips. */
@Composable
private fun ResultRow(f: DayFacility, filters: SearchFilters, onOpen: () -> Unit, onSlot: (Instant) -> Unit, onGame: (PublicGame) -> Unit) {
    val locale = LocalConfiguration.current.locales[0]
    // the API already left only the turns (or games) the filters ask for
    val slots = f.freeSlots
    val games = if (filters.publicOnly) f.publicGames else emptyList()
    Column(Modifier.fillMaxWidth()) {
        Box(Modifier.fillMaxWidth().height(200.dp).background(Lq.colors.muted).clickable(onClick = onOpen)) {
            AsyncImage(f.imageUrl, contentDescription = null, contentScale = ContentScale.Crop, modifier = Modifier.fillMaxSize())
            Box(Modifier.fillMaxSize().background(Brush.verticalGradient(0.5f to Color.Transparent, 1f to Color.Black.copy(alpha = 0.7f))))
            Text(
                f.name,
                color = Color.White,
                style = MaterialTheme.typography.titleLarge.copy(fontSize = 20.sp),
                maxLines = 2,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier.align(Alignment.BottomStart).padding(16.dp),
            )
        }
        // how far, as the API worked it out
        val distance = f.distanceKm?.let { km -> stringResource(R.string.distance_km, if (km < 10) String.format(locale, "%.1f", km) else km.toInt().toString()) }
        Text(
            listOfNotNull(distance, f.councilName).joinToString(" · "),
            style = MaterialTheme.typography.bodyMedium,
            color = Lq.colors.mutedForeground,
            modifier = Modifier.padding(start = 16.dp, end = 16.dp, top = 12.dp, bottom = 10.dp),
        )
        if (filters.publicOnly) {
            LazyRow(contentPadding = PaddingValues(start = 16.dp, end = 16.dp, bottom = 16.dp), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                items(games, key = { it.id }) { game ->
                    Column(
                        Modifier
                            .clip(RoundedCornerShape(8.dp))
                            .background(Lq.colors.pillPositiveBg)
                            .clickable { onGame(game) }
                            .padding(horizontal = 14.dp, vertical = 10.dp),
                    ) {
                        Text(game.startsAt.atZone(ZONE).format(DateTimeFormatter.ofPattern("HH:mm", locale)), style = MaterialTheme.typography.labelLarge, color = Lq.colors.pillPositiveInk)
                        Text(pluralStringResource(R.plurals.public_spots, game.spotsLeft, game.spotsLeft), style = MaterialTheme.typography.bodySmall, color = Lq.colors.pillPositiveInk)
                    }
                }
            }
        } else if (slots.isEmpty()) {
            Text(stringResource(R.string.slots_full), style = MaterialTheme.typography.bodyMedium, color = Lq.colors.mutedForeground,
                modifier = Modifier.padding(start = 16.dp, bottom = 16.dp))
        } else {
            LazyRow(contentPadding = PaddingValues(start = 16.dp, end = 16.dp, bottom = 16.dp), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                items(slots) { slot ->
                    Text(
                        slot.atZone(ZONE).format(DateTimeFormatter.ofPattern("HH:mm", locale)),
                        style = MaterialTheme.typography.labelLarge,
                        color = Lq.colors.foreground,
                        modifier = Modifier
                            .clip(RoundedCornerShape(8.dp))
                            .background(Lq.colors.muted)
                            .clickable { onSlot(slot) }
                            .padding(horizontal = 16.dp, vertical = 12.dp),
                    )
                }
            }
        }
        HorizontalDivider(color = Lq.colors.border)
    }
}

/** Everything else to filter by (council, type of facility, services…), in a sheet from the bottom. */
@OptIn(ExperimentalMaterial3Api::class, ExperimentalLayoutApi::class)
@Composable
private fun FiltersSheet(
    initial: SearchFilters,
    catalogue: List<TagCategory>,
    councils: List<Pair<String, String>>,
    onApply: (SearchFilters) -> Unit,
    onDismiss: () -> Unit,
) {
    var f by remember { mutableStateOf(initial) }
    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true), containerColor = Lq.colors.card) {
        Column(Modifier.fillMaxWidth().verticalScroll(rememberScrollState()).padding(horizontal = 16.dp).navigationBarsPadding()) {
            Text(stringResource(R.string.filters), style = MaterialTheme.typography.headlineSmall, color = Lq.colors.foreground)
            Spacer(Modifier.height(20.dp))
            Group(stringResource(R.string.show)) {
                Toggle(stringResource(R.string.free_turns), !f.publicOnly) { f = f.copy(publicOnly = false) }
                Toggle(stringResource(R.string.public_title), f.publicOnly) { f = f.copy(publicOnly = true) }
            }
            catalogue.firstOrNull()?.let { sports ->
                Group(stringResource(R.string.sport)) {
                    Toggle(stringResource(R.string.any_sport), f.sport == null) { f = f.copy(sport = null) }
                    sports.tags.forEach { tag -> Toggle(tag.name, f.sport == tag.slug) { f = f.copy(sport = if (f.sport == tag.slug) null else tag.slug) } }
                }
            }
            Group(stringResource(R.string.council)) {
                Toggle(stringResource(R.string.all_councils), f.council == null) { f = f.copy(council = null) }
                councils.forEach { (slug, name) -> Toggle(name, f.council == slug) { f = f.copy(council = if (f.council == slug) null else slug) } }
            }
            catalogue.drop(1).forEach { category ->
                Group(category.name) {
                    category.tags.forEach { tag ->
                        Toggle(tag.name, tag.slug in f.tags) { f = f.copy(tags = if (tag.slug in f.tags) f.tags - tag.slug else f.tags + tag.slug) }
                    }
                }
            }
            Row(Modifier.fillMaxWidth().padding(vertical = 16.dp), horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                SecondaryButton(stringResource(R.string.clear_filters), { f = f.copy(tags = emptySet(), council = null, sport = null, publicOnly = false) }, Modifier.weight(1f))
                PrimaryButton(stringResource(R.string.apply), { onApply(f) }, Modifier.weight(1f))
            }
        }
    }
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun Group(title: String, content: @Composable () -> Unit) {
    Text(title, style = MaterialTheme.typography.titleMedium, color = Lq.colors.foreground)
    Spacer(Modifier.height(10.dp))
    FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) { content() }
    Spacer(Modifier.height(24.dp))
}

@Composable
private fun Toggle(label: String, selected: Boolean, onClick: () -> Unit) {
    val ink = if (selected) Lq.colors.primaryForeground else Lq.colors.foreground
    Text(
        label,
        color = ink,
        style = MaterialTheme.typography.labelLarge,
        modifier = Modifier
            .clip(RoundedCornerShape(19.dp))
            .background(if (selected) Lq.colors.primary else Lq.colors.muted)
            .clickable(role = Role.Checkbox, onClick = onClick)
            .padding(horizontal = 16.dp, vertical = 10.dp),
    )
}

