package gal.tribo.librequenda.ui.entrance

import gal.tribo.librequenda.data.EntranceResult
import org.junit.Assert.assertEquals
import org.junit.Test

/** When no turn could be booked at the door, the answer says why, as the web does. */
class EntranceWordsTest {
    private val messages = mapOf("booked" to "Booked", "not_booked" to "No slot could be booked", "too_far" to "Too far", "refused" to "Refused")
    private val reasons = mapOf("slot_booked" to "This slot is already booked.", "other" to "Book in the app or ask the staff.")

    @Test fun notBookedSaysWhy() {
        assertEquals("No slot could be booked. This slot is already booked.", entranceWords(EntranceResult("not_booked", reason = "slot_booked"), messages, reasons))
    }

    @Test fun anUnknownReasonFallsBackToOther() {
        assertEquals("No slot could be booked. Book in the app or ask the staff.", entranceWords(EntranceResult("not_booked", reason = "something_new"), messages, reasons))
    }

    @Test fun withoutAReasonOnlyTheResult() {
        assertEquals("No slot could be booked", entranceWords(EntranceResult("not_booked"), messages, reasons))
    }

    @Test fun otherResultsIgnoreTheReason() {
        assertEquals("Too far", entranceWords(EntranceResult("too_far", reason = "slot_booked"), messages, reasons))
        assertEquals("Refused", entranceWords(EntranceResult("unknown"), messages, reasons))
    }
}
