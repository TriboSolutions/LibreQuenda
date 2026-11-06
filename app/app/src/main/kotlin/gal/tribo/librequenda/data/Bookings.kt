package gal.tribo.librequenda.data

import org.json.JSONObject
import java.time.Instant

/** The longest description of a public booking, in characters, as the API takes it. */
const val MAX_PUBLIC_DESCRIPTION = 1000

/** One of their bookings, made or joined (GET /me/bookings), as the web's /reservas shows it. */
data class Booking(
    val id: String,
    val startsAt: Instant,
    val endsAt: Instant,
    val cancelled: Boolean,
    val facilityName: String,
    val councilName: String,
    val imageUrl: String?,
    val canCancel: Boolean,
    val checkedIn: Boolean,
    val attendance: String, // past ones: attended, no_show, unknown or ""
    val owner: Boolean, // false: joined someone else's
    val ownerName: String,
    val capacity: Int, // people including the owner; 0 = no invites
    val participantCount: Int,
    val canChange: Boolean = false,
    val canLeave: Boolean = false,
    val isPublic: Boolean = false,
    val publicDescription: String = "", // what they are doing, for people who join
    val kind: String = "private", // private, public, or event: the council's, joined by the person
    val title: String = "", // events only
    val cancelUntil: Instant? = null,
    val facility: FacilityDetails = FacilityDetails(),
    // checking in (owner only): how the facility takes it, and the booking's QR for the reader
    val councilSlug: String = "",
    val facilitySlug: String = "",
    val facilityId: String = "", // for a problem report
    val checkinWays: List<String> = emptyList(), // reader, entry_code, printed_qr, staff
    val checkinCode: String = "", // the QR's content; "" unless upcoming and confirmed
    val checkedInAt: Instant? = null,
    val checkinRadiusM: Int = 0,
    val staffContacts: List<StaffContact> = emptyList(),
    val canRate: Boolean = false, // ended, not a no-show, within 30 days, not rated yet: the rating sheet
) {
    val isEvent: Boolean get() = kind == "event"

    /** People coming, the owner included; an event has no owner, and its capacity is all for who joins. */
    val people: Int get() = participantCount + if (isEvent) 0 else 1
}

/** Who to ask on site to mark the entry. */
data class StaffContact(val name: String, val phone: String, val place: String)

/** Where to go and what to know before going (the booking's facility). */
data class FacilityDetails(
    val address: String = "",
    val latitude: Double? = null,
    val longitude: Double? = null,
    val accessibilityInfo: String = "",
    val slotMinutes: Int = 0,
    val minCancelHours: Int = 0,
    val minAge: Int? = null,
    val residentsOnly: Boolean = false, // only the council's verified residents book or join
    val usageRules: String = "",
)

data class MyBookings(val upcoming: List<Booking>, val history: List<Booking>, val historyPage: Int = 1, val historyHasMore: Boolean = false)

private fun JSONObject.toBooking() = Booking(
    id = getString("id"),
    startsAt = Instant.parse(getString("starts_at")),
    endsAt = Instant.parse(getString("ends_at")),
    cancelled = optString("status") == "cancelled",
    facilityName = optString("facility_name"),
    councilName = optString("tenant_name"),
    imageUrl = optJSONObject("facility")?.optString("image_url")?.takeIf { it.isNotBlank() }?.let(Api::absolute),
    canCancel = optBoolean("can_cancel"),
    checkedIn = !isNull("checked_in_at") && optString("checked_in_at").isNotBlank(),
    checkedInAt = if (isNull("checked_in_at")) null else optString("checked_in_at").takeIf { it.isNotBlank() }?.let { runCatching { java.time.OffsetDateTime.parse(it).toInstant() }.getOrNull() },
    councilSlug = optString("tenant_slug"),
    facilitySlug = optString("facility_slug"),
    facilityId = optString("facility_id"),
    checkinWays = optJSONArray("checkin_ways")?.let { a -> List(a.length()) { a.getString(it) } }.orEmpty(),
    checkinCode = optString("checkin_code"),
    checkinRadiusM = optInt("checkin_radius_m"),
    staffContacts = optJSONArray("staff_contacts")?.let { a ->
        List(a.length()) { a.getJSONObject(it) }.map { StaffContact(it.optString("name"), it.optString("phone"), it.optString("place")) }
    }.orEmpty(),
    attendance = optString("attendance"),
    canRate = optBoolean("can_rate"),
    owner = optString("role", "owner") == "owner",
    ownerName = optString("owner_name"),
    capacity = optInt("capacity"),
    participantCount = optInt("participant_count"),
    canChange = optBoolean("can_change"),
    canLeave = optBoolean("can_leave"),
    isPublic = optBoolean("is_public"),
    publicDescription = optString("public_description"),
    kind = optString("kind", "private"),
    title = optString("title"),
    cancelUntil = optString("cancel_until").takeIf { it.isNotBlank() }?.let { runCatching { java.time.OffsetDateTime.parse(it).toInstant() }.getOrNull() },
    facility = optJSONObject("facility")?.let { f ->
        FacilityDetails(
            address = f.optString("address"),
            latitude = if (f.isNull("latitude")) null else f.optDouble("latitude"),
            longitude = if (f.isNull("longitude")) null else f.optDouble("longitude"),
            accessibilityInfo = f.optString("accessibility_info"),
            slotMinutes = f.optInt("slot_minutes"),
            minCancelHours = f.optInt("min_cancel_hours"),
            minAge = if (f.isNull("min_age")) null else f.optInt("min_age"),
            residentsOnly = f.optBoolean("residents_only"),
            usageRules = f.optString("usage_rules"),
        )
    } ?: FacilityDetails(),
)

/** GET /me/bookings?page=: the upcoming ones whole, and a page of the history (100), most recent first. */
suspend fun fetchMyBookings(token: String, historyPage: Int = 1): MyBookings {
    val json = Api.getJson(Api.baseUrl + "/me/bookings?page=$historyPage", token)
    fun list(name: String) = json.optJSONArray(name)?.let { a -> List(a.length()) { a.getJSONObject(it).toBooking() } }.orEmpty()
    val p = json.optJSONObject("history_pagination")
    return MyBookings(list("upcoming"), list("history"), p?.optInt("page", historyPage) ?: historyPage, p?.optBoolean("has_more") ?: false)
}

/** DELETE /me/bookings/{id}: the turn is free again for someone else. */
suspend fun cancelBooking(token: String, id: String) {
    Api.send("DELETE", Api.baseUrl + "/me/bookings/$id", token)
}

/** POST /me/bookings/{id}/invite: the invite link's token (made, or the current one). */
suspend fun shareInvite(token: String, id: String): String =
    Api.send("POST", Api.baseUrl + "/me/bookings/$id/invite", token)?.optString("token").orEmpty()

/** The link people open to join: the web's invite page, with or without an account. */
fun inviteUrl(inviteToken: String) = Api.origin + "/invitacion/" + inviteToken

/** PUT /me/bookings/{id}/public: anyone signed in can find it in «For you» and join. */
suspend fun setPublic(token: String, id: String, public: Boolean, description: String? = null) {
    val body = JSONObject().put("public", public)
    if (description != null) body.put("public_description", description)
    Api.send("PUT", Api.baseUrl + "/me/bookings/$id/public", token, body)
}

/** DELETE /me/bookings/{id}/participation: leave someone else's booking. */
suspend fun leaveBooking(token: String, id: String) {
    Api.send("DELETE", Api.baseUrl + "/me/bookings/$id/participation", token)
}

/** PATCH /me/bookings/{id}: another free turn of the same facility; who joined, the QR and the invite stay with it. */
suspend fun changeBooking(token: String, id: String, startsAt: Instant) {
    Api.send("PATCH", Api.baseUrl + "/me/bookings/$id", token, JSONObject().put("starts_at", startsAt.toString()))
}

/** The kinds of problem a resident can report, as the web lists them. */
val ISSUE_CATEGORIES = listOf("equipment", "lighting", "cleaning", "changing_rooms", "accessibility", "safety", "other")

/** POST /me/issues: a problem at the facility, from a booking under way; it lands in the council's maintenance list. */
suspend fun reportIssue(
    token: String, facilityId: String, bookingId: String, category: String, description: String,
    photoRef: String?, urgent: Boolean, shareContact: Boolean, position: Position,
) {
    // only from the facility: where the phone is goes with it, checked and never stored
    val body = JSONObject()
        .put("facility_id", facilityId).put("booking_id", bookingId).put("category", category)
        .put("description", description).put("urgent", urgent).put("share_contact", shareContact)
        .put("latitude", position.latitude).put("longitude", position.longitude).put("accuracy", position.accuracy.toDouble())
    photoRef?.let { body.put("photo_ref", it) }
    Api.send("POST", Api.baseUrl + "/me/issues", token, body)
}
