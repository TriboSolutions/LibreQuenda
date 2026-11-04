package gal.tribo.librequenda.ui.main

import android.Manifest
import android.os.Build
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.annotation.DrawableRes
import androidx.compose.runtime.LaunchedEffect
import gal.tribo.librequenda.notify.Notifier
import androidx.compose.animation.animateColorAsState
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.spring
import androidx.compose.animation.core.tween
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.interaction.collectIsPressedAsState
import androidx.compose.foundation.layout.RowScope
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.draw.drawBehind
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.activity.compose.BackHandler
import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.slideInHorizontally
import androidx.compose.animation.slideOutHorizontally
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.semantics.heading
import gal.tribo.librequenda.ui.components.InitialsAvatar
import gal.tribo.librequenda.ui.components.Pill
import gal.tribo.librequenda.ui.components.PrimaryButton
import gal.tribo.librequenda.ui.components.SecondaryButton
import kotlinx.coroutines.launch
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.selection.selectable
import androidx.compose.foundation.selection.selectableGroup
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Badge
import androidx.compose.material3.BadgedBox
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.pluralStringResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import coil.compose.AsyncImage
import gal.tribo.librequenda.R
import gal.tribo.librequenda.SessionState
import gal.tribo.librequenda.data.AppNotification
import gal.tribo.librequenda.data.Branding
import gal.tribo.librequenda.ui.components.BrandLockup
import gal.tribo.librequenda.ui.flash.FlashScreen
import gal.tribo.librequenda.ui.flash.FlashSuccessOverlay
import gal.tribo.librequenda.ui.legal.LegalContent
import gal.tribo.librequenda.ui.legal.LegalText
import gal.tribo.librequenda.ui.legal.SupportContent
import gal.tribo.librequenda.HomeState
import gal.tribo.librequenda.data.GeoPoint
import gal.tribo.librequenda.ui.home.HomeScreen
import androidx.compose.animation.slideInVertically
import androidx.compose.animation.slideOutVertically
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import gal.tribo.librequenda.ui.profile.ProfileActions
import gal.tribo.librequenda.ui.profile.ProfileScreen
import gal.tribo.librequenda.ui.bookings.BookingsScreen
import gal.tribo.librequenda.ui.search.SearchFilters
import gal.tribo.librequenda.ui.search.SearchPage
import gal.tribo.librequenda.ui.bookings.BookingHistory
import gal.tribo.librequenda.ui.bookings.BookingPage
import gal.tribo.librequenda.ui.bookings.RateSheet
import gal.tribo.librequenda.ui.components.SuccessOverlay
import gal.tribo.librequenda.ui.facility.Checkout
import gal.tribo.librequenda.ui.facility.ConfirmBookingPage
import gal.tribo.librequenda.ui.facility.FacilityPage
import gal.tribo.librequenda.ui.facility.FacilityTarget
import gal.tribo.librequenda.ui.bookings.rememberBookingsState
import gal.tribo.librequenda.ui.preferences.PreferencesContent
import androidx.compose.foundation.layout.ColumnScope
import gal.tribo.librequenda.ui.theme.Lq
import gal.tribo.librequenda.ui.theme.onBrandBlue

enum class Tab { Home, Flash, Bookings }

/**
 * The app's frame: the installation's logo top left, notifications and the menu
 * top right, and Home · Flash booking · Profile at the bottom, the flash booking
 * raised in the middle.
 */
@Composable
fun MainScreen(
    session: SessionState,
    home: HomeState,
    branding: Branding,
    onRefreshHome: () -> Unit,
    onLocation: (GeoPoint) -> Unit,
    onOpenNotifications: () -> Unit,
    onSignIn: () -> Unit,
    onRegister: () -> Unit,
    onSignOut: () -> Unit,
    actions: ProfileActions,
    onMarkRead: () -> Unit = {},
    opening: String? = null, // from a tapped notification
    onOpened: () -> Unit = {},
) {
    var tab by rememberSaveable { mutableStateOf(Tab.Home) }
    var notificationsOpen by rememberSaveable { mutableStateOf(false) } // survives the language change (the activity is recreated)
    var menuOpen by rememberSaveable { mutableStateOf(false) } // survives the language change (the activity is recreated)
    var preferencesOpen by rememberSaveable { mutableStateOf(false) } // survives the language change (the activity is recreated)
    var legalOpen by rememberSaveable { mutableStateOf<String?>(null) } // "terms", "privacy" or "support", over the menu
    var profileOpen by rememberSaveable { mutableStateOf(false) } // survives the language change (the activity is recreated)
    var bookingOpen by rememberSaveable { mutableStateOf<String?>(null) } // a booking's page
    var ratingOpen by rememberSaveable { mutableStateOf<String?>(null) } // a booking that ended, to rate in a sheet
    var rated by remember { mutableStateOf(false) } // just rated: the celebration, as after checking in
    var facilityOpen by remember { mutableStateOf<FacilityTarget?>(null) } // a facility, maybe with a turn chosen
    var confirming by remember { mutableStateOf<Checkout?>(null) } // the booking (or joining) to confirm
    var historyOpen by rememberSaveable { mutableStateOf(false) } // survives the language change (the activity is recreated)
    var searchOpen by rememberSaveable { mutableStateOf(false) }
    var searchFilters by remember { mutableStateOf(SearchFilters()) }
    var browseFilters by remember { mutableStateOf(SearchFilters()) } // signed out, the home's own search
    val bookings = rememberBookingsState(session, actions)
    var flashDone by remember { mutableStateOf<Pair<String, String?>?>(null) } // booked on the spot and in (words, booking): the celebration, then the booking

    // a tapped notification: a reminder opens the bookings, any other notice the notices
    LaunchedEffect(opening) {
        when {
            opening == null -> return@LaunchedEffect
            // one booking: its page, over the bookings
            opening.startsWith("booking:") -> { tab = Tab.Bookings; menuOpen = false; bookings.reload(); bookingOpen = opening.removePrefix("booking:") }
            // a booking that ended: the stars, in a sheet over the bookings
            opening.startsWith(Notifier.OPEN_RATE) -> { tab = Tab.Bookings; menuOpen = false; notificationsOpen = false; ratingOpen = opening.removePrefix(Notifier.OPEN_RATE) }
            opening == Notifier.OPEN_BOOKINGS -> { tab = Tab.Bookings; menuOpen = false }
            opening == Notifier.OPEN_NOTICES -> { notificationsOpen = true; onOpenNotifications() }
            else -> return@LaunchedEffect
        }
        onOpened()
    }
    // signed in, Android 13+ asks once whether the app may notify (reminders, notices)
    val notifyPermission = rememberLauncherForActivityResult(ActivityResultContracts.RequestPermission()) {}
    LaunchedEffect(session.signedIn) {
        if (session.signedIn && Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) notifyPermission.launch(Manifest.permission.POST_NOTIFICATIONS)
    }

    Box(Modifier.fillMaxSize()) {
        Scaffold(
            containerColor = Lq.colors.background,
            contentWindowInsets = WindowInsets(0),
            topBar = {
                TopBar(
                    session = session,
                    branding = branding,
                    onNotifications = { notificationsOpen = true; onOpenNotifications() },
                    onMenu = { menuOpen = true },
                )
            },
            bottomBar = { BottomBar(tab) { tab = it } },
        ) { padding ->
            val modifier = Modifier.padding(padding)
            when (tab) {
                Tab.Home -> HomeScreen(
                    home = home,
                    session = session,
                    nextBooking = bookings.data?.upcoming?.firstOrNull { !it.cancelled },
                    onRefresh = { onRefreshHome(); bookings.reload() },
                    onLocation = onLocation,
                    onSignIn = onSignIn,
                    onSearch = { searchFilters = it; searchOpen = true },
                    onProfile = { profileOpen = true },
                    // the next booking opens its own page, straight away
                    onBookings = { bookingOpen = bookings.data?.upcoming?.firstOrNull { !it.cancelled }?.id },
                    onFacility = { f, at ->
                        facilityOpen = FacilityTarget(f.councilSlug, f.slug, at?.atZone(java.time.ZoneId.of("Europe/Madrid"))?.toLocalDate() ?: java.time.LocalDate.now(), at)
                    },
                    browse = browseFilters,
                    onBrowse = { browseFilters = it },
                    // signed out, the facility opens to look at; booking there asks to sign in
                    onBrowseFacility = { f, slot -> facilityOpen = FacilityTarget(f.councilSlug, f.slug, browseFilters.day, slot) },
                    modifier = modifier,
                )
                Tab.Flash -> FlashScreen(session, actions, onSignIn, onProfile = { profileOpen = true }, onBooked = { message, id -> flashDone = message to id }, modifier = modifier)
                Tab.Bookings -> BookingsScreen(session, bookings, actions, onSignIn, onBookNow = { tab = Tab.Home }, onHistory = { historyOpen = true }, onOpen = { bookingOpen = it.id }, modifier = modifier)
            }
        }

        // pages over everything, the bars included
        NotificationsPage(
            notificationsOpen, session, onSignIn = { notificationsOpen = false; onSignIn() }, onMarkRead = onMarkRead,
            onRate = { ratingOpen = it },
        ) { notificationsOpen = false }
        MenuPage(
            open = menuOpen,
            session = session,
            onClose = { menuOpen = false },
            onProfile = { profileOpen = true },
            onPreferences = { preferencesOpen = true },
            branding = branding,
            onLegal = { legalOpen = it },
            onSignIn = onSignIn,
            onRegister = onRegister,
            onSignOut = onSignOut,
        )
        SearchPage(
            open = searchOpen,
            filters = searchFilters,
            onFilters = { searchFilters = it },
            councils = home.facilities.map { it.councilSlug to it.councilName }.distinct(),
            location = home.location,
            onBack = { searchOpen = false },
            onSignIn = onSignIn,
            // the facility, with that turn chosen; signed out, booking there asks to sign in
            onSlot = { f, slot -> facilityOpen = FacilityTarget(f.councilSlug, f.slug, searchFilters.day, slot) },
        )
        FacilityPage(facilityOpen, signedIn = session.signedIn, onSignIn = onSignIn, onBack = { facilityOpen = null }, onCheckout = { confirming = it })
        ConfirmBookingPage(confirming, actions, onBack = { confirming = null }, onBooked = { id ->
            // booked: on to it, the way back to the bookings
            confirming = null; facilityOpen = null; searchOpen = false
            tab = Tab.Bookings
            bookings.reload()
            bookingOpen = id
        })
        BookingPage(bookingOpen, bookings, actions, onBack = { bookingOpen = null })
        SettingsPage(historyOpen, stringResource(R.string.bookings_history), onBack = { historyOpen = false }) {
            BookingHistory(bookings, actions, onRate = { ratingOpen = it.id })
        }
        // over the menu, as its next pages
        SettingsPage(profileOpen, stringResource(R.string.profile_title), onBack = { profileOpen = false }) {
            ProfileScreen(session, actions, onSignIn, onRegister, onSignOut = { profileOpen = false; onSignOut() })
        }
        SettingsPage(preferencesOpen, stringResource(R.string.preferences), onBack = { preferencesOpen = false }) {
            PreferencesContent(session, actions)
        }
        SettingsPage(legalOpen == "terms", stringResource(R.string.terms_title), onBack = { legalOpen = null }) {
            LegalContent(LegalText.Terms, branding)
        }
        SettingsPage(legalOpen == "privacy", stringResource(R.string.privacy_title), onBack = { legalOpen = null }) {
            LegalContent(LegalText.Privacy, branding)
        }
        SettingsPage(legalOpen == "support", stringResource(R.string.support_title), onBack = { legalOpen = null }) {
            SupportContent(branding)
        }
        // from a notification the app may be starting: the sheet waits for the session to be back
        ratingOpen?.takeIf { session.signedIn && session.me != null }?.let { id ->
            RateSheet(id, actions, onDismiss = { ratingOpen = null }, onSaved = { ratingOpen = null; rated = true; bookings.reload() })
        }
        if (rated) SuccessOverlay(stringResource(R.string.rating_thanks_title), stringResource(R.string.rating_thanks)) { rated = false }
        flashDone?.let { (message, id) ->
            // on to the booking just made, over the bookings (the way back); the bookings alone when it is not known
            FlashSuccessOverlay(message) { flashDone = null; tab = Tab.Bookings; bookings.reload(); bookingOpen = id }
        }
    }
}

/**
 * A settings page (profile, preferences), pushed in from the right over everything.
 * No bar: the back arrow alone at the top, the title large under it, and the page
 * all on one ground, so it reads as one piece.
 */
@Composable
private fun SettingsPage(open: Boolean, title: String, onBack: () -> Unit, content: @Composable () -> Unit) {
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
            IconButton(onClick = onBack, modifier = Modifier.padding(start = 4.dp, top = 4.dp)) {
                Icon(painterResource(R.drawable.ic_arrow_left), contentDescription = stringResource(R.string.back), tint = Lq.colors.foreground)
            }
            Text(
                title,
                style = MaterialTheme.typography.headlineLarge,
                color = Lq.colors.foreground,
                modifier = Modifier.padding(start = 16.dp, end = 16.dp, top = 4.dp).semantics { heading() },
            )
            content()
        }
    }
}

@Composable
private fun TopBar(
    session: SessionState,
    branding: Branding,
    onNotifications: () -> Unit,
    onMenu: () -> Unit,
) {
    Column(Modifier.background(Lq.colors.card).statusBarsPadding()) {
        Row(
            Modifier.fillMaxWidth().height(60.dp).padding(start = 16.dp, end = 4.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            val home = stringResource(R.string.home)
            Box(Modifier.weight(1f).semantics { contentDescription = home }) {
                InstallationLogo(branding)
            }
            NotificationsButton(session, onNotifications)
            IconButton(onClick = onMenu) {
                Icon(painterResource(R.drawable.ic_menu), contentDescription = stringResource(R.string.menu), tint = Lq.colors.foreground)
            }
        }
        HorizontalDivider(color = Lq.colors.border)
    }
}

/** The logo the installation chose at /admin, or LibreQuenda's lockup when it has none. */
@Composable
private fun InstallationLogo(branding: Branding) {
    when {
        branding.logoUrl != null -> Row(verticalAlignment = Alignment.CenterVertically) {
            AsyncImage(
                model = branding.logoUrl,
                contentDescription = null,
                contentScale = ContentScale.Fit,
                modifier = Modifier.size(36.dp).clip(RoundedCornerShape(6.dp)),
            )
            if (branding.customName.isNotBlank()) {
                Spacer(Modifier.size(10.dp))
                Text(branding.customName, style = MaterialTheme.typography.titleMedium, color = Lq.colors.brandInk, maxLines = 1, overflow = TextOverflow.Ellipsis)
            }
        }
        branding.customName.isNotBlank() ->
            Text(branding.customName, style = MaterialTheme.typography.titleLarge, color = Lq.colors.brandInk, maxLines = 1, overflow = TextOverflow.Ellipsis)
        else -> BrandLockup(height = 34.dp)
    }
}

@Composable
private fun NotificationsButton(session: SessionState, onClick: () -> Unit) {
    val unread = session.notifications?.unreadCount ?: 0
    val label = if (unread > 0) pluralStringResource(R.plurals.notifications_unread, unread, unread)
    else stringResource(R.string.notifications)
    IconButton(onClick = onClick, modifier = Modifier.semantics { contentDescription = label }) {
        BadgedBox(badge = { if (unread > 0) Badge { Text(if (unread > 9) "9+" else "$unread") } }) {
            Icon(painterResource(R.drawable.ic_bell), contentDescription = null, tint = Lq.colors.foreground)
        }
    }
}

/**
 * The hamburger menu: a page that comes up from the bottom. Who they are at the
 * top, then «Your account» (profile, preferences) as rows that open their page.
 */
@Composable
private fun MenuPage(
    open: Boolean,
    session: SessionState,
    onClose: () -> Unit,
    onProfile: () -> Unit,
    onPreferences: () -> Unit,
    branding: Branding,
    onLegal: (String) -> Unit,
    onSignIn: () -> Unit,
    onRegister: () -> Unit,
    onSignOut: () -> Unit,
) {
    BackHandler(enabled = open, onBack = onClose)
    fun pick(action: () -> Unit) { onClose(); action() }
    AnimatedVisibility(
        visible = open,
        enter = slideInVertically(tween(280)) { it },
        exit = slideOutVertically(tween(220)) { it },
    ) {
        Column(
            Modifier
                .fillMaxSize()
                .background(Lq.colors.background)
                .clickable(interactionSource = null, indication = null) {} // taps stay on the page
                .statusBarsPadding()
        ) {
            IconButton(onClick = onClose, modifier = Modifier.padding(start = 4.dp, top = 4.dp)) {
                Icon(painterResource(R.drawable.ic_x), contentDescription = stringResource(R.string.close), tint = Lq.colors.foreground)
            }
            Column(
                Modifier.fillMaxWidth().verticalScroll(rememberScrollState()).navigationBarsPadding()
                    .padding(start = 16.dp, end = 16.dp, bottom = 24.dp),
            ) {
                val me = session.me
                if (session.signedIn) {
                    Row(Modifier.fillMaxWidth().padding(vertical = 8.dp), verticalAlignment = Alignment.CenterVertically) {
                        Column(Modifier.weight(1f)) {
                            Text(me?.fullName.orEmpty(), style = MaterialTheme.typography.headlineSmall, color = Lq.colors.foreground, maxLines = 2, overflow = TextOverflow.Ellipsis)
                            Text(me?.email.orEmpty(), style = MaterialTheme.typography.bodyMedium, color = Lq.colors.mutedForeground, maxLines = 1, overflow = TextOverflow.Ellipsis)
                            if (me != null && !me.emailVerified) {
                                Spacer(Modifier.height(8.dp))
                                Pill(stringResource(R.string.email_not_verified), Lq.colors.pillNegativeBg, Lq.colors.pillNegativeInk)
                            }
                        }
                        Spacer(Modifier.size(12.dp))
                        if (me?.avatarUrl != null) {
                            AsyncImage(me.avatarUrl, contentDescription = null, contentScale = ContentScale.Crop, modifier = Modifier.size(64.dp).clip(CircleShape))
                        } else {
                            // the logo's blue, as the app's round buttons
                            Box(Modifier.size(64.dp).clip(CircleShape).background(Lq.colors.brandBlue), contentAlignment = Alignment.Center) {
                                Text(me?.initials.orEmpty(), color = Lq.colors.onBrandBlue, style = MaterialTheme.typography.titleLarge)
                            }
                        }
                    }
                } else {
                    Text(stringResource(R.string.welcome_title), style = MaterialTheme.typography.headlineSmall, color = Lq.colors.foreground)
                    Spacer(Modifier.height(16.dp))
                    PrimaryButton(stringResource(R.string.sign_in), { pick(onSignIn) }, Modifier.fillMaxWidth(), enabled = !session.busy)
                    Spacer(Modifier.height(12.dp))
                    SecondaryButton(stringResource(R.string.create_account), { pick(onRegister) }, Modifier.fillMaxWidth(), enabled = !session.busy)
                }

                MenuGroup(stringResource(R.string.menu_account)) {
                    if (session.signedIn) {
                        MenuRow(R.drawable.ic_user, stringResource(R.string.profile_title), stringResource(R.string.menu_profile_sub), onClick = onProfile)
                        MenuDivider()
                    }
                    // the menu stays under the preferences: back returns to it
                    MenuRow(R.drawable.ic_sliders, stringResource(R.string.preferences), stringResource(R.string.menu_preferences_sub), onClick = onPreferences)
                }
                // each installation is its own (self-hosted): its contact comes from SUPPORT_EMAIL and SUPPORT_PHONE
                if (branding.hasSupport) {
                    MenuGroup(stringResource(R.string.support_group)) {
                        val contact = listOf(branding.supportEmail, branding.supportPhone).filter { it.isNotBlank() }.joinToString(" · ")
                        MenuRow(R.drawable.ic_life_buoy, stringResource(R.string.support_title), contact) { onLegal("support") }
                    }
                }
                MenuGroup(stringResource(R.string.legal_group)) {
                    MenuRow(R.drawable.ic_scroll_text, stringResource(R.string.terms_title), stringResource(R.string.terms_sub)) { onLegal("terms") }
                    MenuDivider()
                    MenuRow(R.drawable.ic_shield_check, stringResource(R.string.privacy_title), stringResource(R.string.privacy_sub)) { onLegal("privacy") }
                }
                if (session.signedIn) {
                    MenuGroup(null) {
                        MenuRow(R.drawable.ic_log_out, stringResource(R.string.sign_out), null, chevron = false) { pick(onSignOut) }
                    }
                }
            }
        }
    }
}

@Composable
private fun MenuGroup(title: String?, content: @Composable ColumnScope.() -> Unit) {
    Spacer(Modifier.height(28.dp))
    if (title != null) {
        Text(title, style = MaterialTheme.typography.titleLarge, color = Lq.colors.foreground, modifier = Modifier.semantics { heading() })
        Spacer(Modifier.height(12.dp))
    }
    Column(Modifier.fillMaxWidth().clip(RoundedCornerShape(12.dp)).background(Lq.colors.card), content = content)
}

@Composable
private fun MenuDivider() = HorizontalDivider(Modifier.padding(start = 58.dp), color = Lq.colors.border)

@Composable
private fun MenuRow(@DrawableRes icon: Int, label: String, subtitle: String?, chevron: Boolean = true, onClick: () -> Unit) {
    Row(
        Modifier.fillMaxWidth().clickable(onClick = onClick).padding(horizontal = 16.dp, vertical = 16.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(painterResource(icon), contentDescription = null, tint = Lq.colors.foreground, modifier = Modifier.size(24.dp))
        Spacer(Modifier.size(18.dp))
        Column(Modifier.weight(1f)) {
            Text(label, style = MaterialTheme.typography.bodyLarge, color = Lq.colors.foreground)
            if (subtitle != null) {
                Text(subtitle, style = MaterialTheme.typography.bodySmall, color = Lq.colors.mutedForeground, maxLines = 1, overflow = TextOverflow.Ellipsis)
            }
        }
        if (chevron) Icon(painterResource(R.drawable.ic_chevron_right), contentDescription = null, tint = Lq.colors.mutedForeground, modifier = Modifier.size(20.dp))
    }
}

private val BarHeight = 52.dp
private val Rise = 24.dp // how far the flash button stands above the bar
private val IconTop = 9.dp // from the bar's line to the side icons
private val CircleSize = 48.dp

/**
 * Home and Profile at the sides; the flash booking raised in the middle, in the
 * brand blue. The raised part is inside the bar's bounds (transparent above the
 * line), so the whole button takes taps.
 */
@Composable
private fun BottomBar(selected: Tab, onSelect: (Tab) -> Unit) {
    val card = Lq.colors.card
    val line = Lq.colors.border
    Row(
        Modifier
            .fillMaxWidth()
            .drawBehind {
                val top = Rise.toPx()
                drawRect(card, topLeft = Offset(0f, top), size = Size(size.width, size.height - top))
                drawLine(line, Offset(0f, top), Offset(size.width, top), strokeWidth = 0.5.dp.toPx())
            }
            .navigationBarsPadding()
            .height(Rise + BarHeight)
            .selectableGroup(),
        horizontalArrangement = Arrangement.SpaceEvenly,
        verticalAlignment = Alignment.Bottom,
    ) {
        SideTab(Tab.Home, R.drawable.ic_home, stringResource(R.string.tab_home), selected, onSelect)
        FlashTab(selected == Tab.Flash) { onSelect(Tab.Flash) }
        SideTab(Tab.Bookings, R.drawable.ic_calendar, stringResource(R.string.tab_bookings), selected, onSelect)
    }
}

/** Shrinks a touch while pressed: the only feedback, no ripple. */
@Composable
private fun pressScale(interaction: MutableInteractionSource, pressed: Float): Float {
    val isPressed by interaction.collectIsPressedAsState()
    val scale by animateFloatAsState(
        if (isPressed) pressed else 1f,
        animationSpec = spring(dampingRatio = 0.55f, stiffness = 600f),
        label = "press",
    )
    return scale
}

@Composable
private fun RowScope.SideTab(
    tab: Tab,
    @DrawableRes icon: Int,
    label: String,
    selected: Tab,
    onSelect: (Tab) -> Unit,
) {
    val isSelected = tab == selected
    val color by animateColorAsState(
        if (isSelected) Lq.colors.primary else Lq.colors.mutedForeground.copy(alpha = 0.75f),
        animationSpec = tween(220),
        label = "tab",
    )
    val interaction = remember { MutableInteractionSource() }
    val scale = pressScale(interaction, 0.9f)
    Column(
        Modifier
            .weight(1f)
            .height(BarHeight)
            .selectable(
                selected = isSelected,
                role = Role.Tab,
                interactionSource = interaction,
                indication = null,
                onClick = { onSelect(tab) },
            )
            .graphicsLayer { scaleX = scale; scaleY = scale }
            .padding(top = IconTop),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Icon(painterResource(icon), contentDescription = null, tint = color, modifier = Modifier.size(22.dp))
        Spacer(Modifier.height(2.dp))
        Text(label, color = color, fontSize = 11.sp, fontWeight = if (isSelected) FontWeight.SemiBold else FontWeight.Medium)
    }
}

@Composable
private fun RowScope.FlashTab(selected: Boolean, onClick: () -> Unit) {
    val interaction = remember { MutableInteractionSource() }
    val scale = pressScale(interaction, 0.9f)
    val labelColor by animateColorAsState(
        if (selected) Lq.colors.primary else Lq.colors.mutedForeground.copy(alpha = 0.75f),
        animationSpec = tween(220),
        label = "flash",
    )
    // selected: the deep brand blue, a pale halo around it, a filled bolt, a little bigger
    val fill by animateColorAsState(if (selected) Lq.colors.primaryDeep else Lq.colors.brandBlue, tween(220), label = "fill") // the logo's blue; deep when selected
    val halo by animateColorAsState(if (selected) Lq.colors.ring.copy(alpha = 0.18f) else Color.Transparent, tween(220), label = "halo")
    val grow by animateFloatAsState(if (selected) 1.08f else 1f, spring(dampingRatio = 0.5f, stiffness = 500f), label = "grow")
    Column(
        Modifier
            .weight(1f)
            .fillMaxHeight()
            .selectable(
                selected = selected,
                role = Role.Tab,
                interactionSource = interaction,
                indication = null,
                onClick = onClick,
            )
            // the circle ends where the side icons end, so the three labels share a line
            .padding(top = Rise + IconTop + 22.dp - CircleSize),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Box(
            Modifier
                .graphicsLayer { scaleX = scale * grow; scaleY = scale * grow }
                .size(CircleSize)
                .drawBehind { drawCircle(halo, radius = size.minDimension / 2 + 6.dp.toPx()) }
                .shadow(3.dp, CircleShape)
                .clip(CircleShape)
                .background(fill),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                painterResource(if (selected) R.drawable.ic_zap_filled else R.drawable.ic_zap),
                contentDescription = null,
                tint = if (selected) Lq.colors.primaryForeground else Lq.colors.onBrandBlue,
                modifier = Modifier.size(22.dp),
            )
        }
        Spacer(Modifier.height(2.dp))
        Text(
            stringResource(R.string.tab_flash),
            color = labelColor,
            fontSize = 11.sp,
            fontWeight = if (selected) FontWeight.SemiBold else FontWeight.Medium,
        )
    }
}

/** Notifications: a page of its own, pushed in from the right like a new screen. */
@Composable
private fun NotificationsPage(open: Boolean, session: SessionState, onSignIn: () -> Unit, onMarkRead: () -> Unit, onRate: (String) -> Unit, onBack: () -> Unit) {
    BackHandler(enabled = open, onBack = onBack)
    AnimatedVisibility(
        visible = open,
        enter = slideInHorizontally(tween(260)) { it },
        exit = slideOutHorizontally(tween(220)) { it },
    ) {
        Column(
            Modifier
                .fillMaxSize()
                .background(Lq.colors.background)
                .clickable(interactionSource = null, indication = null) {} // taps stay on the page
        ) {
            Column(Modifier.background(Lq.colors.card).statusBarsPadding()) {
                Row(
                    Modifier.fillMaxWidth().height(60.dp).padding(start = 4.dp, end = 16.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    IconButton(onClick = onBack) {
                        Icon(painterResource(R.drawable.ic_arrow_left), contentDescription = stringResource(R.string.back), tint = Lq.colors.foreground)
                    }
                    Text(
                        stringResource(R.string.notifications),
                        style = MaterialTheme.typography.titleLarge,
                        color = Lq.colors.foreground,
                        modifier = Modifier.weight(1f).semantics { heading() },
                    )
                }
                HorizontalDivider(color = Lq.colors.border)
            }
            val items = session.notifications?.items.orEmpty()
            // going in reads them all, as the web's bell: the count goes at once, and what was new
            // keeps its dot while the page stays open, so it still stands out
            val fresh = remember { mutableStateOf(emptySet<String>()) }
            LaunchedEffect(items) { // again when they load, or a new one comes while open
                val unread = items.filterNot { it.read }.map { it.id }
                if (unread.isNotEmpty()) { fresh.value += unread; onMarkRead() }
            }
            when {
                !session.signedIn -> Column(Modifier.padding(20.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
                    Text(stringResource(R.string.notifications_signed_out), color = Lq.colors.mutedForeground)
                    PrimaryButton(stringResource(R.string.sign_in), onSignIn, Modifier.fillMaxWidth(), enabled = !session.busy)
                }
                items.isEmpty() -> Text(stringResource(R.string.notifications_empty), color = Lq.colors.mutedForeground, modifier = Modifier.padding(20.dp))
                else -> LazyColumn(Modifier.fillMaxSize().navigationBarsPadding().padding(horizontal = 16.dp)) {
                    items(items, key = { it.id }) { n ->
                        // the request to rate opens the stars, as the web's bell does
                        val rate = n.type == "rating_requested" && n.bookingId.isNotBlank()
                        NotificationRow(n, new = n.id in fresh.value || !n.read, onClick = if (rate) ({ onRate(n.bookingId) }) else null)
                    }
                }
            }
        }
    }
}

/** When a notice came: «14:45» today, «onte» or the date before. */
private fun received(context: android.content.Context, at: java.time.Instant): String {
    val zone = java.time.ZoneId.systemDefault()
    val date = at.atZone(zone).toLocalDate()
    val today = java.time.LocalDate.now(zone)
    val locale = context.resources.configuration.locales[0]
    return when (date) {
        today -> at.atZone(zone).format(java.time.format.DateTimeFormatter.ofPattern("HH:mm"))
        else -> date.format(java.time.format.DateTimeFormatter.ofPattern("d MMM", locale))
    }
}

@Composable
private fun NotificationRow(n: AppNotification, new: Boolean, onClick: (() -> Unit)? = null) {
    // the same words as the Android notification and the web: what, where, and the day and time
    val context = androidx.compose.ui.platform.LocalContext.current
    androidx.compose.ui.platform.LocalConfiguration.current // read again when the language changes
    val (title, text) = Notifier.describe(context, n)
    Row(
        Modifier.fillMaxWidth().then(if (onClick != null) Modifier.clickable(onClick = onClick) else Modifier).padding(vertical = 12.dp),
        verticalAlignment = Alignment.Top,
    ) {
        Box(
            Modifier.padding(top = 7.dp).size(8.dp).clip(CircleShape)
                .background(if (new) Lq.colors.brandBlue else Color.Transparent)
        )
        Spacer(Modifier.size(12.dp))
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(title, style = MaterialTheme.typography.titleSmall, color = Lq.colors.foreground, modifier = Modifier.weight(1f))
                n.createdAt?.let {
                    Spacer(Modifier.size(8.dp))
                    Text(received(context, it), style = MaterialTheme.typography.bodySmall, color = Lq.colors.mutedForeground)
                }
            }
            Text(text, style = MaterialTheme.typography.bodyMedium, color = Lq.colors.mutedForeground)
        }
    }
    HorizontalDivider(color = Lq.colors.border)
}
