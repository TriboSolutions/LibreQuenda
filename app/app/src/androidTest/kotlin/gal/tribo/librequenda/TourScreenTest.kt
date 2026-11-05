package gal.tribo.librequenda

import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.test.platform.app.InstrumentationRegistry
import gal.tribo.librequenda.ui.theme.LibreQuendaTheme
import gal.tribo.librequenda.ui.welcome.TourScreen
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test

class TourScreenTest {
    @get:Rule val compose = createComposeRule()

    // in whatever language the app is in on the device
    private fun text(id: Int) = InstrumentationRegistry.getInstrumentation().targetContext.getString(id)

    @Test fun skipEndsTheTour() {
        var done = 0
        compose.setContent { LibreQuendaTheme { TourScreen(onDone = { done++ }) } }
        compose.onNodeWithText(text(R.string.tour_find_title)).assertIsDisplayed()
        compose.onNodeWithText(text(R.string.tour_skip)).performClick()
        assertEquals(1, done)
    }

    @Test fun nextWalksEveryStepThenStarts() {
        var done = 0
        compose.setContent { LibreQuendaTheme { TourScreen(onDone = { done++ }) } }
        listOf(R.string.tour_book_title, R.string.tour_games_title, R.string.tour_flash_title, R.string.tour_enter_title, R.string.tour_bookings_title).forEach {
            compose.onNodeWithText(text(R.string.tour_next)).performClick()
            compose.onNodeWithText(text(it)).assertIsDisplayed()
        }
        assertEquals(0, done)
        compose.onNodeWithText(text(R.string.tour_start)).performClick()
        assertEquals(1, done)
    }
}
