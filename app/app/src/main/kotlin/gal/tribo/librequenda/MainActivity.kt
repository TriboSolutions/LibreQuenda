package gal.tribo.librequenda

import android.content.Intent
import android.os.Bundle
import androidx.compose.runtime.mutableStateOf
import gal.tribo.librequenda.notify.NoticesWorker
import gal.tribo.librequenda.notify.Notifier
import androidx.activity.ComponentActivity
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.compose.setContent
import androidx.lifecycle.repeatOnLifecycle
import androidx.compose.runtime.LaunchedEffect
import androidx.activity.enableEdgeToEdge
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.runtime.getValue
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.platform.LocalConfiguration
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import gal.tribo.librequenda.data.Api
import androidx.lifecycle.viewmodel.compose.viewModel
import gal.tribo.librequenda.ui.main.MainScreen
import gal.tribo.librequenda.ui.theme.LibreQuendaTheme
import gal.tribo.librequenda.ui.welcome.TourScreen
import gal.tribo.librequenda.ui.welcome.WelcomeScreen
import kotlinx.coroutines.launch

class MainActivity : ComponentActivity() {
    /** Where a tapped notification asked to open (bookings or notices); consumed once shown. */
    private val opening = mutableStateOf<String?>(null)

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        opening.value = intent.getStringExtra(Notifier.EXTRA_OPEN)
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        // notifications without push: the channels, and the background check every 15 minutes
        Notifier.createChannels(this)
        NoticesWorker.start(this)
        if (savedInstanceState == null) opening.value = intent.getStringExtra(Notifier.EXTRA_OPEN)
        setContent {
            LibreQuendaTheme {
                // the API answers (errors, emails) in the app's language
                Api.language = LocalConfiguration.current.locales[0].language
                val model: AppViewModel = viewModel()
                val session by model.session.collectAsStateWithLifecycle()
                val branding by model.branding.collectAsStateWithLifecycle()
                val home by model.home.collectAsStateWithLifecycle()
                val scope = rememberCoroutineScope()
                // the bell's count while the app is on screen: at once on coming back, then every minute
                // (the background check posts phone notifications, not this count)
                val lifecycle = androidx.lifecycle.compose.LocalLifecycleOwner.current.lifecycle
                LaunchedEffect(session.signedIn) {
                    if (session.signedIn) lifecycle.repeatOnLifecycle(androidx.lifecycle.Lifecycle.State.STARTED) {
                        while (true) {
                            model.refreshNotifications()
                            kotlinx.coroutines.delay(60_000)
                        }
                    }
                }

                // Zitadel's sign-in page in a Custom Tab; back here with the code (or nothing when closed)
                val signInLauncher = rememberLauncherForActivityResult(ActivityResultContracts.StartActivityForResult()) {
                    model.onSignInResult(it.data)
                }
                val endSessionLauncher = rememberLauncherForActivityResult(ActivityResultContracts.StartActivityForResult()) {}

                fun signIn(register: Boolean) = scope.launch {
                    model.signInIntent(register)?.let(signInLauncher::launch)
                }

                // the tour again, from the menu; it survives the language change (the activity is recreated)
                var tourAgain by rememberSaveable { mutableStateOf(false) }

                if (!session.signedIn && !session.welcomeSeen) {
                    WelcomeScreen(
                        busy = session.busy,
                        error = session.error,
                        onSignIn = { signIn(register = false) },
                        onRegister = { signIn(register = true) },
                        onExplore = model::skipWelcome,
                    )
                } else if (!session.tourSeen) {
                    // once in, signed in or exploring without an account: the tour, the first time.
                    // Instead of the app, not over it: Home asks for the location only after it.
                    TourScreen(onDone = model::finishTour)
                } else {
                    MainScreen(
                        session = session,
                        home = home,
                        branding = branding,
                        onRefreshHome = model::refreshHome,
                        onLocation = model::setLocation,
                        onOpenNotifications = model::refreshNotifications,
                        onSignIn = { signIn(register = false) },
                        onRegister = { signIn(register = true) },
                        onSignOut = { model.signOut()?.let(endSessionLauncher::launch) },
                        actions = model,
                        onMarkRead = model::markNotificationsRead,
                        opening = opening.value,
                        onOpened = { opening.value = null },
                        onTour = { tourAgain = true },
                    )
                    if (tourAgain) TourScreen(onDone = { tourAgain = false })
                }
            }
        }
    }
}
