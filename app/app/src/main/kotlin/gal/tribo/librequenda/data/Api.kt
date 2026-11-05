package gal.tribo.librequenda.data

import gal.tribo.librequenda.BuildConfig
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import org.json.JSONObject
import java.net.HttpURLConnection
import java.net.URI

/** The LibreQuenda server: the core API under /api and the web origin that serves /config.json. */
object Api {
    val baseUrl: String = BuildConfig.API_BASE_URL
    val origin: String = baseUrl.removeSuffix("/api")

    /** The app's language: the API answers its errors (and writes its emails) in it. */
    @Volatile var language: String = "gl"

    /** A media path from the API (/api/media/…) as a full URL. */
    fun absolute(path: String): String = if (path.startsWith("/")) origin + path else path

    /** An error from the API; [message] is its own, already in the app's language, when it sent one. */
    class HttpException(val status: Int, message: String? = null) : Exception(message ?: "HTTP $status")

    suspend fun getJson(url: String, accessToken: String? = null): JSONObject = json(request("GET", url, accessToken, body = null))

    suspend fun postJson(url: String, body: JSONObject, accessToken: String? = null): JSONObject = json(request("POST", url, accessToken, body))

    /** PUT, POST or DELETE where only success matters (204 or a body that is not needed). */
    suspend fun send(method: String, url: String, accessToken: String, body: JSONObject? = null): JSONObject? =
        request(method, url, accessToken, body).takeIf { it.isNotEmpty() }?.let(::json)

    /** A file the API sends (the data export): its bytes and the name it suggests. */
    suspend fun download(url: String, accessToken: String): Pair<ByteArray, String?> = withContext(Dispatchers.IO) {
        open("GET", url, accessToken).use { c ->
            val name = c.getHeaderField("Content-Disposition")?.let { Regex("filename=\"?([^\";]+)").find(it)?.groupValues?.get(1) }
            c.inputStream.use { it.readBytes() } to name
        }
    }

    /**
     * POST /api/media: an image as multipart form data; the API answers its storage reference and a presigned URL.
     * [fields] say whose file it is: "facility" (its id) for a problem's photo.
     */
    suspend fun upload(folder: String, bytes: ByteArray, mimeType: String, accessToken: String, fields: Map<String, String> = emptyMap()): JSONObject = withContext(Dispatchers.IO) {
        val boundary = "lq" + System.nanoTime()
        val body = java.io.ByteArrayOutputStream().apply {
            fun line(text: String) = write((text + "\r\n").toByteArray())
            line("--$boundary"); line("Content-Disposition: form-data; name=\"folder\""); line(""); line(folder)
            for ((name, value) in fields) { line("--$boundary"); line("Content-Disposition: form-data; name=\"$name\""); line(""); line(value) }
            line("--$boundary"); line("Content-Disposition: form-data; name=\"file\"; filename=\"image\""); line("Content-Type: $mimeType"); line("")
            write(bytes); line(""); line("--$boundary--")
        }.toByteArray()
        open("POST", baseUrl + "/media", accessToken, "multipart/form-data; boundary=$boundary", body).use { c ->
            json(c.inputStream.bufferedReader().use { it.readText() })
        }
    }

    private fun json(text: String) = JSONObject(text.ifBlank { "{}" })

    private suspend fun request(method: String, url: String, accessToken: String?, body: JSONObject?): String = withContext(Dispatchers.IO) {
        open(method, url, accessToken, body?.let { "application/json" }, body?.toString()?.toByteArray()).use { c ->
            c.inputStream.bufferedReader().use { it.readText() }
        }
    }

    /** Opens the request and checks the status; the caller reads the body inside use {}. */
    private fun open(method: String, url: String, accessToken: String?, contentType: String? = null, body: ByteArray? = null): Connection {
        val connection = URI(url).toURL().openConnection() as HttpURLConnection
        try {
            connection.requestMethod = method
            connection.connectTimeout = 10_000
            connection.readTimeout = 30_000
            connection.setRequestProperty("Accept", "application/json")
            connection.setRequestProperty("Accept-Language", language)
            accessToken?.let { connection.setRequestProperty("Authorization", "Bearer $it") }
            if (body != null) {
                connection.doOutput = true
                connection.setRequestProperty("Content-Type", contentType)
                connection.outputStream.use { it.write(body) }
            }
            val status = connection.responseCode
            if (status !in 200..299) {
                // the API explains in {"error": "..."}, translated; the rate limiter does not
                val message = runCatching {
                    connection.errorStream?.bufferedReader()?.use { it.readText() }?.let { JSONObject(it).optString("error") }
                }.getOrNull()?.takeIf { it.isNotBlank() }
                throw HttpException(status, message)
            }
            return Connection(connection)
        } catch (e: Exception) {
            connection.disconnect()
            throw e
        }
    }

    private class Connection(private val c: HttpURLConnection) : AutoCloseable {
        val inputStream get() = c.inputStream
        fun getHeaderField(name: String): String? = c.getHeaderField(name)
        override fun close() = c.disconnect()
    }
}

/** What zitadel-setup writes to /config.json: where to sign in and with which client. */
data class RuntimeConfig(val authority: String, val clientId: String, val scope: String) {
    companion object {
        suspend fun load(): RuntimeConfig {
            val json = Api.getJson(Api.origin + "/config.json")
            return RuntimeConfig(
                authority = json.getString("authority"),
                clientId = json.getString("androidClientId"),
                scope = json.getString("scope"),
            )
        }
    }
}

/** The signed-in person, from GET /api/me. */
data class Me(
    val id: String,
    val fullName: String,
    val email: String,
    val emailVerified: Boolean,
    val avatarRef: String,
    val avatarUrl: String?,
    val isSuperadmin: Boolean,
    val profileComplete: Boolean,
    val birthDate: String, // YYYY-MM-DD, "" until filled in
    val phone: String,
    val preferredLanguage: String, // "" = the same as the app
    val emailNotifications: Boolean,
    val emailReminders: Boolean,
    val showRecommendations: Boolean,
    val rating: RatingSummary? = null, // what others gave them after public bookings; null until someone did
    val residentOf: Set<String> = emptySet(), // slugs of the councils that verified them as their resident
    val residentRequests: List<ResidentRequest> = emptyList(), // asked to be verified, not decided yet
) {
    /** Their pending request to [councilSlug], if they sent one. */
    fun residentRequest(councilSlug: String): ResidentRequest? = residentRequests.firstOrNull { it.tenantSlug == councilSlug }

    val initials: String
        get() = (fullName.ifBlank { email }).split(Regex("\\s+")).filter { it.isNotBlank() }
            .take(2).joinToString("") { it.first().uppercase() }
}

suspend fun fetchMe(accessToken: String): Me = Api.getJson(Api.baseUrl + "/me", accessToken).toMe()

internal fun JSONObject.toMe() = Me(
    id = optString("id"),
    fullName = optString("full_name"),
    email = optString("email"),
    emailVerified = optBoolean("email_verified"),
    avatarRef = optString("avatar_ref"),
    avatarUrl = optString("avatar_url").takeIf { it.isNotBlank() }?.let(Api::absolute),
    isSuperadmin = optBoolean("is_superadmin"),
    profileComplete = optBoolean("profile_complete"),
    birthDate = optString("birth_date"),
    phone = optString("phone"),
    preferredLanguage = optString("preferred_language"),
    emailNotifications = optBoolean("email_notifications"),
    emailReminders = optBoolean("email_reminders"),
    showRecommendations = optBoolean("show_recommendations"),
    rating = ratingSummary("rating"),
    residentOf = optJSONArray("resident_of")?.let { a -> List(a.length()) { a.getJSONObject(it).optString("tenant_slug") }.toSet() } ?: emptySet(),
    residentRequests = optJSONArray("resident_requests")?.let { a ->
        List(a.length()) { a.getJSONObject(it) }.map { r ->
            ResidentRequest(
                tenantSlug = r.optString("tenant_slug"),
                tenantName = r.optString("tenant_name"),
                requestedAt = r.optString("requested_at").takeIf { it.isNotBlank() }?.let { runCatching { java.time.OffsetDateTime.parse(it).toInstant() }.getOrNull() },
            )
        }
    }.orEmpty(),
)

/** The installation's own name and logo (GET /api/branding), set by a superadministrator; empty = LibreQuenda's. */
/** How the installation presents itself, and its support contact (SUPPORT_EMAIL, SUPPORT_PHONE; "" = not set). */
data class Branding(val customName: String, val logoUrl: String?, val supportEmail: String = "", val supportPhone: String = "") {
    /** What to call it: its own name, or LibreQuenda. */
    val name get() = customName.ifBlank { "LibreQuenda" }
    val hasSupport get() = supportEmail.isNotBlank() || supportPhone.isNotBlank()

    companion object {
        val Default = Branding("", null)
    }
}

suspend fun fetchBranding(): Branding {
    val json = Api.getJson(Api.baseUrl + "/branding")
    return Branding(
        customName = json.optString("custom_name"),
        logoUrl = json.optString("logo_url").takeIf { it.isNotBlank() }?.let(Api::absolute),
        supportEmail = json.optString("support_email").trim(),
        supportPhone = json.optString("support_phone").trim(),
    )
}

data class AppNotification(
    val id: String,
    val type: String,
    val facilityName: String,
    val tenantName: String,
    val read: Boolean,
    val startsAt: java.time.Instant? = null,
    val name: String = "", // booking_joined: who joined
    val closedFrom: java.time.Instant? = null, // a council cancellation caused by a closure
    val closedUntil: java.time.Instant? = null,
    val byStaff: Boolean = false, // a council cancellation of one booking by staff
    val createdAt: java.time.Instant? = null, // when the notice came
    val bookingId: String = "", // "" for notices about no booking
    val isPublic: Boolean = false, // rating_requested: the others can be rated too
    val tenantSlug: String = "", // resident_*: which council
)

private fun JSONObject.instant(key: String): java.time.Instant? =
    if (isNull(key)) null else optString(key).takeIf { it.isNotBlank() }?.let { java.time.OffsetDateTime.parse(it).toInstant() }
data class Notifications(val unreadCount: Int, val items: List<AppNotification>)

suspend fun fetchNotifications(accessToken: String): Notifications {
    val json = Api.getJson(Api.baseUrl + "/me/notifications", accessToken)
    val list = json.optJSONArray("notifications")
    return Notifications(
        unreadCount = json.optInt("unread_count"),
        items = List(list?.length() ?: 0) { index ->
            val n = list!!.getJSONObject(index)
            AppNotification(
                id = n.getString("id"),
                type = n.optString("type"),
                facilityName = n.optString("facility_name"),
                tenantName = n.optString("tenant_name"),
                read = n.optBoolean("read"),
                startsAt = n.instant("starts_at"),
                name = n.optString("name"),
                closedFrom = n.instant("closed_from"),
                closedUntil = n.instant("closed_until"),
                byStaff = n.optBoolean("by_staff"),
                createdAt = n.instant("created_at"),
                bookingId = n.optString("booking_id"),
                isPublic = n.optBoolean("is_public"),
                tenantSlug = n.optString("tenant_slug"),
            )
        },
    )
}

/** A facility as the home page shows it, from any council (GET /api/facilities). */
data class FacilityCard(
    val id: String,
    val slug: String,
    val name: String,
    val imageUrl: String?,
    val latitude: Double?,
    val longitude: Double?,
    val freeToday: Int,
    val councilSlug: String,
    val councilName: String,
    val distanceKm: Double? = null, // from where the phone is, worked out by the API
    val residentsOnly: Boolean = false, // only the council's verified residents book or join (its events and public games too)
)

private fun org.json.JSONObject.toFacilityCard(): FacilityCard {
    val council = getJSONObject("council")
    return FacilityCard(
        id = getString("id"),
        slug = getString("slug"),
        name = getString("name"),
        imageUrl = optString("image_url").takeIf { it.isNotBlank() }?.let(Api::absolute),
        latitude = if (isNull("latitude")) null else optDouble("latitude"),
        longitude = if (isNull("longitude")) null else optDouble("longitude"),
        freeToday = optInt("free_today"),
        councilSlug = council.getString("slug"),
        councilName = council.getString("name"),
        distanceKm = if (isNull("distance_km") || !has("distance_km")) null else optDouble("distance_km"),
        residentsOnly = optBoolean("residents_only"),
    )
}

/** «?lat=&lon=» when the phone shares where it is: the API sorts nearest first and says how far. */
internal fun near(location: GeoPoint?, first: Boolean = true): String =
    location?.let { (if (first) "?" else "&") + "lat=${it.latitude}&lon=${it.longitude}" }.orEmpty()

/** GET /api/facilities: every facility, nearest first when [near] is known (the API sorts). */
suspend fun fetchFacilities(near: GeoPoint? = null): List<FacilityCard> {
    val list = Api.getJson(Api.baseUrl + "/facilities" + near(near)).getJSONArray("facilities") // the first page (100)
    return List(list.length()) { list.getJSONObject(it).toFacilityCard() }
}

data class GeoPoint(val latitude: Double, val longitude: Double)

/** One of the councils' events (GET /api/events): what it is, when, the places left, and where. */
data class UpcomingEvent(
    val id: String,
    val title: String,
    val description: String,
    val startsAt: java.time.Instant,
    val endsAt: java.time.Instant,
    val capacity: Int,
    val spotsLeft: Int, // 0: full
    val facility: FacilityCard,
)

/** GET /api/events: every council's events that have not started, soonest first (the API sorts). */
suspend fun fetchUpcomingEvents(near: GeoPoint? = null): List<UpcomingEvent> {
    val list = Api.getJson(Api.baseUrl + "/events" + near(near)).getJSONArray("events")
    return List(list.length()) { index ->
        val e = list.getJSONObject(index)
        UpcomingEvent(
            id = e.getString("id"),
            title = e.optString("title"),
            description = e.optString("description"),
            startsAt = java.time.OffsetDateTime.parse(e.getString("starts_at")).toInstant(),
            endsAt = java.time.OffsetDateTime.parse(e.getString("ends_at")).toInstant(),
            capacity = e.optInt("capacity"),
            spotsLeft = e.optInt("spots_left"),
            facility = e.getJSONObject("facility").toFacilityCard(),
        )
    }
}

/** Someone's public booking with spots left: what «For you» shows of it. */
data class PublicBooking(val id: String, val startsAt: java.time.Instant, val endsAt: java.time.Instant, val spotsLeft: Int, val ownerFirstName: String, val title: String = "") // title: the council's event

/** One «For you» item (POST /api/me/recommendations): a facility to book, or a public booking to join. */
data class Recommendation(
    val kind: String, // usual_today, favourite, similar, public_booking
    val facility: FacilityCard,
    val startsAt: java.time.Instant?,
    val booking: PublicBooking?,
) {
    val isPublicBooking get() = kind == "public_booking" && booking != null
}

suspend fun fetchRecommendations(accessToken: String, near: GeoPoint?): List<Recommendation> {
    val body = org.json.JSONObject().apply {
        near?.let { put("latitude", it.latitude); put("longitude", it.longitude) }
    }
    val list = Api.postJson(Api.baseUrl + "/me/recommendations", body, accessToken).getJSONArray("recommendations")
    return List(list.length()) { index ->
        val r = list.getJSONObject(index)
        val b = r.optJSONObject("booking")
        Recommendation(
            kind = r.getString("kind"),
            facility = r.getJSONObject("facility").toFacilityCard(),
            startsAt = r.optString("starts_at").takeIf { it.isNotBlank() }?.let(java.time.Instant::parse),
            booking = b?.let {
                PublicBooking(
                    id = it.getString("id"),
                    startsAt = java.time.Instant.parse(it.getString("starts_at")),
                    endsAt = java.time.Instant.parse(it.getString("ends_at")),
                    spotsLeft = it.optInt("spots_left"),
                    ownerFirstName = it.optString("owner_first_name"),
                    title = if (it.optString("kind") == "event") it.optString("title") else "",
                )
            },
        )
    }
}


/** POST /me/notifications/read: every notice read (the bell's count goes to 0). */
suspend fun markNotificationsRead(accessToken: String) {
    Api.send("POST", Api.baseUrl + "/me/notifications/read", accessToken)
}
