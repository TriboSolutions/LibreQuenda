package gal.tribo.librequenda.data

import org.json.JSONArray
import org.json.JSONObject
import java.time.Instant
import java.time.OffsetDateTime

// Stars after a booking, as the web's RateDialog: the facility's and, after a public one, the others'.

/** An average of 1 to 5 stars and how many gave them; never who. */
data class RatingSummary(val average: Double, val count: Int)

/** «rating»: {average, count}, or null until someone rated. */
internal fun JSONObject.ratingSummary(key: String): RatingSummary? =
    optJSONObject(key)?.let { RatingSummary(it.optDouble("average"), it.optInt("count")) }?.takeIf { it.count > 0 }

/** Someone else in a public booking with an account, and the stars given to them (0 = not yet). */
data class PersonToRate(val userId: String, val name: String, val isOwner: Boolean, val stars: Int)

/** GET /me/bookings/{id}/rating: what can be rated, and what was given already (0 = not yet). */
data class BookingRating(
    val bookingId: String,
    val facilityName: String,
    val startsAt: Instant,
    val isPublic: Boolean,
    val canRate: Boolean, // ended, not a no-show, within the 30 days, not rated yet
    val rated: Boolean, // rated already: the stars are final, to look at only
    val rateUntil: Instant,
    val facilityStars: Int,
    val people: List<PersonToRate>,
)

private fun JSONObject.toBookingRating(): BookingRating {
    val people = optJSONArray("people") ?: JSONArray()
    return BookingRating(
        bookingId = getString("booking_id"),
        facilityName = optString("facility_name"),
        startsAt = OffsetDateTime.parse(getString("starts_at")).toInstant(),
        isPublic = optBoolean("is_public"),
        canRate = optBoolean("can_rate"),
        rated = optBoolean("rated"),
        rateUntil = OffsetDateTime.parse(getString("rate_until")).toInstant(),
        facilityStars = optInt("facility_stars"),
        people = List(people.length()) { people.getJSONObject(it) }.map {
            PersonToRate(it.getString("user_id"), it.optString("name"), it.optBoolean("is_owner"), it.optInt("stars"))
        },
    )
}

suspend fun fetchRating(token: String, bookingId: String): BookingRating =
    Api.getJson(Api.baseUrl + "/me/bookings/$bookingId/rating", token).toBookingRating()

/** PUT /me/bookings/{id}/rating, once and for good: the facility's stars (required) and the others' (0: left unrated). */
suspend fun saveRating(token: String, bookingId: String, facilityStars: Int, people: Map<String, Int>) {
    val body = JSONObject()
        .put("facility_stars", facilityStars)
        .put("people", JSONArray(people.map { (id, stars) -> JSONObject().put("user_id", id).put("stars", stars) }))
    Api.send("PUT", Api.baseUrl + "/me/bookings/$bookingId/rating", token, body)
}
