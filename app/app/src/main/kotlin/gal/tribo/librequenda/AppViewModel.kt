package gal.tribo.librequenda

import android.app.Application
import android.content.Context
import android.content.Intent
import androidx.core.content.edit
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import gal.tribo.librequenda.auth.AuthManager
import gal.tribo.librequenda.notify.NoticesWorker
import gal.tribo.librequenda.notify.Reminders
import gal.tribo.librequenda.ui.profile.ProfileActions
import gal.tribo.librequenda.data.Branding
import gal.tribo.librequenda.data.FacilityCard
import gal.tribo.librequenda.data.GeoPoint
import gal.tribo.librequenda.data.Recommendation
import gal.tribo.librequenda.data.fetchFacilities
import gal.tribo.librequenda.data.fetchRecommendations
import gal.tribo.librequenda.data.Me
import gal.tribo.librequenda.data.Notifications
import gal.tribo.librequenda.data.fetchBranding
import gal.tribo.librequenda.data.fetchMe
import gal.tribo.librequenda.data.fetchNotifications
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

/** Home: the councils' events, «For you» and public bookings (signed in), and every facility for the rest. */
data class HomeState(
    val facilities: List<FacilityCard> = emptyList(),
    val events: List<gal.tribo.librequenda.data.UpcomingEvent> = emptyList(), // not started, soonest first
    val recommendations: List<Recommendation> = emptyList(),
    val location: GeoPoint? = null, // shared by the person, for distances; never stored
    val loading: Boolean = true,
    val today: List<gal.tribo.librequenda.data.DayFacility> = emptyList(), // every facility's day: the public bookings to join
)

enum class AuthError { Failed, Cancelled }

data class SessionState(
    val signedIn: Boolean = false,
    val me: Me? = null,
    val notifications: Notifications? = null,
    val busy: Boolean = false, // opening the sign-in page or exchanging the code
    val error: AuthError? = null,
    val welcomeSeen: Boolean = false, // "Explore without an account" was chosen once
    val tourSeen: Boolean = false, // the first-launch tour was finished or skipped
    val accountDeleted: Boolean = false, // just now, from the profile
)

class AppViewModel(application: Application) : AndroidViewModel(application), ProfileActions {
    private val auth = AuthManager(application)
    private val prefs = application.getSharedPreferences("app", Context.MODE_PRIVATE)

    private val sessionState = MutableStateFlow(
        SessionState(signedIn = auth.isSignedIn, welcomeSeen = prefs.getBoolean(KEY_WELCOME_SEEN, false),
            tourSeen = prefs.getBoolean(KEY_TOUR_SEEN, false))
    )
    val session = sessionState.asStateFlow()

    private val brandingState = MutableStateFlow(Branding.Default)
    val branding = brandingState.asStateFlow()

    private val homeState = MutableStateFlow(HomeState())
    val home = homeState.asStateFlow()
    private var facilitiesRequest: Job? = null
    private val residentNoticesSeen = java.util.Collections.synchronizedSet(mutableSetOf<String>()) // resident_* notices already acted on

    init {
        refreshFacilities()
        viewModelScope.launch { runCatching { fetchBranding() }.onSuccess { brandingState.value = it } }
        if (auth.isSignedIn) loadMe()
    }

    /** Pull to refresh on Home: everything it shows. */
    fun refreshHome() {
        refreshFacilities()
        refreshRecommendations()
    }

    fun setLocation(location: GeoPoint) {
        if (homeState.value.location == location) return
        homeState.update { it.copy(location = location) }
        refreshFacilities() // the API sorts them by distance now
        refreshRecommendations() // public bookings near the person
    }

    private fun refreshFacilities() {
        if (facilitiesRequest?.isActive == true) return
        facilitiesRequest = viewModelScope.launch {
            homeState.update { it.copy(loading = true) }
            val near = homeState.value.location
            val list = runCatching { fetchFacilities(near) }.getOrNull() // the first page, nearest first
            homeState.update { it.copy(loading = false, facilities = list ?: it.facilities) }
            // the councils' events, whenever there are any: on top of the home
            runCatching { gal.tribo.librequenda.data.fetchUpcomingEvents(near) }
                .onSuccess { events -> homeState.update { it.copy(events = events) } }
            // today at every council, nearest first: all the public bookings with spots, not only the few «For you» picks
            // the API keeps only the facilities with a public booking with spots left, nearest first
            runCatching {
                gal.tribo.librequenda.data.fetchFacilitiesDay(
                    gal.tribo.librequenda.data.DayQuery(java.time.LocalDate.now(java.time.ZoneId.of("Europe/Madrid")), near, public = true)
                )
            }.onSuccess { page -> homeState.update { it.copy(today = page.items) } }
        }
    }

    private fun refreshRecommendations() {
        if (!sessionState.value.signedIn) return
        viewModelScope.launch {
            try {
                val token = auth.accessToken() ?: return@launch
                val list = fetchRecommendations(token, homeState.value.location)
                homeState.update { it.copy(recommendations = list) }
            } catch (cancelled: CancellationException) {
                throw cancelled
            } catch (_: Exception) {
                // keeps what it had
            }
        }
    }

    /** The intent that opens Zitadel's sign-in; null (with the error shown) when the server can't be reached. */
    suspend fun signInIntent(register: Boolean): Intent? {
        sessionState.update { it.copy(busy = true, error = null) }
        return try {
            auth.signInIntent(register)
        } catch (cancelled: CancellationException) {
            throw cancelled
        } catch (e: Exception) {
            android.util.Log.w(TAG, "sign-in could not start", e)
            sessionState.update { it.copy(busy = false, error = AuthError.Failed) }
            null
        }
    }

    /** Back from the sign-in page: [data] is null when the person closed it. */
    fun onSignInResult(data: Intent?) {
        if (data == null) {
            android.util.Log.i(TAG, "sign-in: no result (closed)")
            sessionState.update { it.copy(busy = false, error = AuthError.Cancelled) }
            return
        }
        viewModelScope.launch {
            try {
                auth.completeSignIn(data)
                sessionState.update { it.copy(signedIn = true, busy = false, error = null, accountDeleted = false) }
                NoticesWorker.runNow(getApplication()) // their reminders and notices, now
                loadMe()
            } catch (cancelled: CancellationException) {
                throw cancelled
            } catch (e: Exception) {
                android.util.Log.w(TAG, "sign-in failed", e) // never the tokens: AppAuth's errors carry none
                sessionState.update { it.copy(signedIn = false, busy = false, error = AuthError.Failed) }
            }
        }
    }

    fun dismissError() = sessionState.update { it.copy(error = null) }

    fun skipWelcome() {
        prefs.edit { putBoolean(KEY_WELCOME_SEEN, true) }
        sessionState.update { it.copy(welcomeSeen = true, error = null) }
    }

    fun finishTour() {
        prefs.edit { putBoolean(KEY_TOUR_SEEN, true) }
        sessionState.update { it.copy(tourSeen = true) }
    }

    /** Signs out here; the returned intent, if any, ends the session at Zitadel. */
    fun signOut(): Intent? {
        val endSession = auth.signOut()
        forgetNotifications()
        sessionState.update { it.copy(signedIn = false, me = null, notifications = null) }
        homeState.update { it.copy(recommendations = emptyList()) }
        return endSession
    }

    private fun loadMe() {
        viewModelScope.launch {
            try {
                val token = auth.accessToken()
                if (token == null) {
                    sessionState.update { it.copy(signedIn = false, me = null) }
                    return@launch
                }
                val me = fetchMe(token)
                sessionState.update { it.copy(me = me) }
                refreshNotifications()
                refreshRecommendations()
            } catch (cancelled: CancellationException) {
                throw cancelled
            } catch (_: Exception) {
                // offline: keep the session, the name shows up next time
            }
        }
    }

    fun refreshNotifications() {
        viewModelScope.launch {
            try {
                val token = auth.accessToken() ?: return@launch
                val notifications = fetchNotifications(token)
                sessionState.update { it.copy(notifications = notifications) }
                // a council decided on their request to be verified as a resident: /me says so now
                val decided = notifications.items.filter { !it.read && it.type in RESIDENT_DECISIONS && residentNoticesSeen.add(it.id) }
                if (decided.isNotEmpty()) runCatching { fetchMe(token) }.onSuccess { me -> sessionState.update { it.copy(me = me) } }
            } catch (cancelled: CancellationException) {
                throw cancelled
            } catch (_: Exception) {
                // the bell keeps what it had
            }
        }
    }

    /** Runs [block] with a fresh access token; a missing session is a 401, like the API's. */
    override suspend fun <T> authed(block: suspend (token: String) -> T): T {
        val token = auth.accessToken() ?: run {
            sessionState.update { it.copy(signedIn = false, me = null) }
            throw gal.tribo.librequenda.data.Api.HttpException(401)
        }
        return block(token)
    }

    /** The profile or the preferences were saved: the API answered the person as it is now. */
    override fun meUpdated(me: Me) {
        val recommendationsChanged = sessionState.value.me?.showRecommendations != me.showRecommendations
        sessionState.update { it.copy(me = me) }
        if (recommendationsChanged) refreshRecommendations()
    }

    /** The name or email changed at the sign-in service: read /me again. */
    override fun reloadMe() = loadMe()

    /** The account was deleted: forget the session here; the screen says it is gone. */
    override fun accountDeleted() {
        auth.clear()
        forgetNotifications()
        sessionState.update { it.copy(signedIn = false, me = null, notifications = null, accountDeleted = true) }
        homeState.update { it.copy(recommendations = emptyList()) }
    }

    /** No reminders or notices of this account left on the phone. */
    private fun forgetNotifications() {
        Reminders.clear(getApplication())
        NoticesWorker.reset(getApplication())
    }

    /** Every notice read: the count goes at once, the API follows. */
    fun markNotificationsRead() {
        sessionState.update { st -> st.copy(notifications = st.notifications?.let { n -> n.copy(unreadCount = 0, items = n.items.map { it.copy(read = true) }) }) }
        viewModelScope.launch {
            runCatching { auth.accessToken()?.let { gal.tribo.librequenda.data.markNotificationsRead(it) } }
        }
    }

    override fun onCleared() = auth.dispose()

    private companion object {
        const val TAG = "LibreQuenda"
        const val KEY_WELCOME_SEEN = "welcome_seen"
        const val KEY_TOUR_SEEN = "tour_seen"
        val RESIDENT_DECISIONS = setOf("resident_verified", "resident_rejected")
    }
}
