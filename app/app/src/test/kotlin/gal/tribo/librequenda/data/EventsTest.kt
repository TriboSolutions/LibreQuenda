package gal.tribo.librequenda.data

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import java.time.Instant

/** The council's events have no owner: nobody else takes a place in them. */
class EventsTest {
    private val at = Instant.parse("2026-11-13T09:00:00Z")

    private fun booking(kind: String, joined: Int) = Booking(
        id = "b", startsAt = at, endsAt = at.plusSeconds(4 * 3600), cancelled = false, facilityName = "Pavillón", councilName = "Lalín",
        imageUrl = null, canCancel = false, checkedIn = false, attendance = "", owner = false, ownerName = "", capacity = 30,
        participantCount = joined, kind = kind,
    )

    @Test fun aJoinedBookingCountsItsOwner() {
        assertEquals(4, booking("public", 3).people)
    }

    @Test fun anEventCountsOnlyWhoJoined() {
        val event = booking("event", 3)
        assertTrue(event.isEvent)
        assertEquals(3, event.people)
    }

    @Test fun spotsLeftOnTheFacilityDay() {
        // a public booking lists its owner first; an event, only who joined
        val public = PublicTurn("p", at, at, "", capacity = 4, people = listOf("Uxía", "Brais"))
        val event = PublicTurn("e", at, at, "", capacity = 30, people = listOf("Brais"), kind = "event", title = "Torneo 5x5")
        assertFalse(public.isEvent)
        assertEquals(2, public.spotsLeft)
        assertTrue(event.isEvent)
        assertEquals(29, event.spotsLeft)
    }
}
