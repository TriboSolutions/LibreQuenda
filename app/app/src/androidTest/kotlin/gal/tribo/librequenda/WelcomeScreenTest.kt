package gal.tribo.librequenda

import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import gal.tribo.librequenda.ui.theme.LibreQuendaTheme
import gal.tribo.librequenda.ui.welcome.WelcomeScreen
import androidx.test.platform.app.InstrumentationRegistry
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test

class WelcomeScreenTest {
    @get:Rule val compose = createComposeRule()

    // in whatever language the app is in on the device
    private fun text(id: Int) = InstrumentationRegistry.getInstrumentation().targetContext.getString(id)

    @Test fun offersSignInRegisterAndExplore() {
        val tapped = mutableListOf<String>()
        compose.setContent {
            LibreQuendaTheme {
                WelcomeScreen(busy = false, error = null, onSignIn = { tapped += "in" }, onRegister = { tapped += "up" }, onExplore = { tapped += "explore" })
            }
        }
        compose.onNodeWithText(text(R.string.sign_in)).performClick()
        compose.onNodeWithText(text(R.string.create_account)).performClick()
        compose.onNodeWithText(text(R.string.welcome_explore)).performClick()
        assertTrue(tapped == listOf("in", "up", "explore"))
    }

    @Test fun cancelledSignInIsExplained() {
        compose.setContent {
            LibreQuendaTheme { WelcomeScreen(busy = false, error = AuthError.Cancelled, onSignIn = {}, onRegister = {}, onExplore = {}) }
        }
        compose.onNodeWithText(text(R.string.sign_in_cancelled)).assertIsDisplayed()
    }
}
