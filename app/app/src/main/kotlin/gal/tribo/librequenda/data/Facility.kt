package gal.tribo.librequenda.data

import org.json.JSONObject
import java.time.Instant
import java.time.LocalDate
import java.time.OffsetDateTime

/** A turn of a facility's day; only the free ones can be booked. */
data class Slot(val startsAt: Instant, val endsAt: Instant, val status: String) {
    val free get() = status == "free"
}

/** A facility's page for a day (GET /tenants/{council}/facilities/{facility}?date=). */
data class FacilityDay(
    val id: String,
    val slug: String,
    val name: String,
    val description: String,
    val sport: String,
    val imageUrl: String?,
    val councilSlug: String,
    val councilName: String,
    val capacity: Int,
    val maxAdvanceHours: Int,
    val details: FacilityDetails,
    val slots: List<Slot>,
    val publicTurns: List<PublicTurn> = emptyList(),
    val rating: RatingSummary? = null, // residents' stars after their bookings; null until someone rated
)

/** Someone's public booking at the facility that day: join it instead of booking. */
data class PublicTurn(val id: String, val startsAt: Instant, val endsAt: Instant, val description: String, val capacity: Int, val people: List<String>) {
    /** The owner is the first of people. */
    val spotsLeft: Int get() = capacity - people.size
}

suspend fun fetchFacilityDay(councilSlug: String, facilitySlug: String, date: LocalDate): FacilityDay {
    val json = Api.getJson(Api.baseUrl + "/tenants/$councilSlug/facilities/$facilitySlug?date=$date")
    val f = json.getJSONObject("facility")
    val slots = f.optJSONArray("slots")
    val publics = f.optJSONArray("public_bookings")
    return FacilityDay(
        id = f.getString("id"),
        slug = f.getString("slug"),
        name = f.getString("name"),
        description = f.optString("description"),
        sport = f.optString("sport"),
        imageUrl = f.optString("image_url").takeIf { it.isNotBlank() }?.let(Api::absolute),
        councilSlug = councilSlug,
        councilName = json.getJSONObject("tenant").optString("name"),
        capacity = f.optInt("capacity"),
        maxAdvanceHours = f.optInt("max_advance_hours"),
        details = FacilityDetails(
            address = f.optString("address"),
            latitude = if (f.isNull("latitude")) null else f.optDouble("latitude"),
            longitude = if (f.isNull("longitude")) null else f.optDouble("longitude"),
            accessibilityInfo = f.optString("accessibility_info"),
            slotMinutes = f.optInt("slot_minutes"),
            minCancelHours = f.optInt("min_cancel_hours"),
            minAge = if (f.isNull("min_age")) null else f.optInt("min_age"),
            usageRules = f.optString("usage_rules"),
        ),
        slots = List(slots?.length() ?: 0) { slots!!.getJSONObject(it) }.map {
            Slot(OffsetDateTime.parse(it.getString("starts_at")).toInstant(), OffsetDateTime.parse(it.getString("ends_at")).toInstant(), it.optString("status"))
        },
        publicTurns = List(publics?.length() ?: 0) { publics!!.getJSONObject(it) }.map { p ->
            val people = p.optJSONArray("people")
            PublicTurn(
                id = p.getString("id"),
                startsAt = OffsetDateTime.parse(p.getString("starts_at")).toInstant(),
                endsAt = OffsetDateTime.parse(p.getString("ends_at")).toInstant(),
                description = p.optString("description"),
                capacity = p.optInt("capacity"),
                people = List(people?.length() ?: 0) { people!!.optString(it) },
            )
        },
        rating = json.ratingSummary("rating"),
    )
}

/**
 * POST /me/bookings: the turn, with the rules accepted (the API refuses without);
 * public lets anyone signed in find and join it. Returns the new booking's id.
 */
suspend fun book(token: String, facilityId: String, startsAt: Instant, public: Boolean, description: String): String =
    Api.postJson(
        Api.baseUrl + "/me/bookings",
        JSONObject()
            .put("facility_id", facilityId)
            .put("starts_at", startsAt.toString())
            .put("accept_rules", true)
            .put("public", public)
            .put("public_description", description),
        token,
    ).getString("id")
