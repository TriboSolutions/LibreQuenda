package gal.tribo.librequenda.data

import org.json.JSONObject
import java.time.Instant
import java.time.LocalDate
import java.time.OffsetDateTime

/** A facility tag from the platform's catalogue (sport, type, services…). Names come from the API. */
data class Tag(val slug: String, val name: String, val facilityCount: Int)
data class TagCategory(val name: String, val tags: List<Tag>)

/** GET /api/tags: the catalogue, by category; only tags some facility has. */
suspend fun fetchTags(): List<TagCategory> {
    val categories = Api.getJson(Api.baseUrl + "/tags").getJSONArray("categories")
    return List(categories.length()) { i ->
        val c = categories.getJSONObject(i)
        val tags = c.getJSONArray("tags")
        TagCategory(
            c.getString("name"),
            List(tags.length()) { j ->
                val t = tags.getJSONObject(j)
                Tag(t.getString("slug"), t.getString("name"), t.optInt("facility_count"))
            }.filter { it.facilityCount > 0 },
        )
    }.filter { it.tags.isNotEmpty() }
}

/** A facility on a given day: what the search lists, with its free turns. */
data class DayFacility(
    val id: String,
    val slug: String,
    val name: String,
    val sport: String,
    val imageUrl: String?,
    val latitude: Double?,
    val longitude: Double?,
    val tags: Set<String>,
    val councilSlug: String,
    val councilName: String,
    val freeSlots: List<Instant>,
    val publicGames: List<PublicGame>, // with spots left
    val distanceKm: Double? = null, // from where the phone is, worked out by the API
    val residentsOnly: Boolean = false, // only the council's verified residents book or join
    val rating: RatingSummary? = null, // residents' stars after their bookings; null until someone rated
)

/** Someone's public booking at a facility that day: join it to fill a spot. */
/** A public booking with spots left, or the council's event (a title, no owner). */
data class PublicGame(val id: String, val startsAt: Instant, val endsAt: Instant, val spotsLeft: Int, val ownerFirstName: String, val title: String = "") {
    val isEvent: Boolean get() = title.isNotBlank()
}

/** A page of a list the API pages (?page=, 100 by default): its items and whether more follow. */
data class Page<T>(val items: List<T>, val page: Int, val hasMore: Boolean, val total: Int)

internal fun <T> JSONObject.page(key: String, item: (JSONObject) -> T): Page<T> {
    val list = getJSONArray(key)
    val p = optJSONObject("pagination") ?: optJSONObject("${key}_pagination")
    return Page(List(list.length()) { item(list.getJSONObject(it)) }, p?.optInt("page", 1) ?: 1, p?.optBoolean("has_more") ?: false, p?.optInt("total", list.length()) ?: list.length())
}

/** What the facilities search asks the API for; the API filters, sorts (nearest first) and pages. */
data class DayQuery(
    val date: LocalDate,
    val near: GeoPoint? = null,
    val words: String = "",
    val council: String? = null,
    val sport: String? = null,
    val tags: Set<String> = emptySet(),
    val from: String? = null, // HH:MM, the council's time
    val until: String? = null,
    val public: Boolean = false,
    val minRating: Double? = null, // residents' average at least this; the API leaves out who nobody rated
)

/**
 * GET /api/facilities/day: every council's facilities and their turns that day,
 * as [query] narrows them, nearest first when the position is known. The API
 * filters, sorts and pages; the app shows the pages in order.
 */
suspend fun fetchFacilitiesDay(query: DayQuery, page: Int = 1): Page<DayFacility> {
    val params = buildList {
        add("date=${query.date}")
        add("page=$page")
        query.near?.let { add("lat=${it.latitude}"); add("lon=${it.longitude}") }
        query.words.trim().takeIf { it.isNotEmpty() }?.let { add("q=" + java.net.URLEncoder.encode(it.take(100), "UTF-8")) }
        query.council?.let { add("council=$it") }
        query.sport?.let { add("sport=$it") }
        if (query.tags.isNotEmpty()) add("tags=" + query.tags.joinToString(","))
        query.from?.let { add("from=$it") }
        query.until?.let { add("until=$it") }
        if (query.public) add("public=true")
        query.minRating?.let { add("min_rating=$it") }
    }
    val json = Api.getJson(Api.baseUrl + "/facilities/day?" + params.joinToString("&"))
    return json.page("facilities") { f -> f.toDayFacility() }
}

private fun JSONObject.toDayFacility(): DayFacility {
    val f = this
    return run {
        val council = f.getJSONObject("council")
        val tags = f.optJSONArray("tags")
        val slots = f.optJSONArray("slots")
        val games = f.optJSONArray("public_bookings")
        DayFacility(
            id = f.getString("id"),
            slug = f.getString("slug"),
            name = f.getString("name"),
            sport = f.optString("sport"),
            imageUrl = f.optString("image_url").takeIf { it.isNotBlank() }?.let(Api::absolute),
            latitude = if (f.isNull("latitude")) null else f.optDouble("latitude"),
            longitude = if (f.isNull("longitude")) null else f.optDouble("longitude"),
            tags = List(tags?.length() ?: 0) { tags!!.getString(it) }.toSet(),
            councilSlug = council.getString("slug"),
            councilName = council.optString("name"),
            residentsOnly = f.optBoolean("residents_only"),
            rating = f.ratingSummary("rating"),
            // the turns still free: what the search offers to book
            freeSlots = List(slots?.length() ?: 0) { slots!!.getJSONObject(it) }
                .filter { it.optString("status") == "free" }
                .map { OffsetDateTime.parse(it.getString("starts_at")).toInstant() },
            // a public booking's capacity counts the owner, and people lists the owner first;
            // an event has no owner: people are only who joined
            publicGames = List(games?.length() ?: 0) { games!!.getJSONObject(it) }.map { g ->
                val people = g.optJSONArray("people")
                val event = g.optString("kind") == "event"
                PublicGame(
                    id = g.getString("id"),
                    startsAt = OffsetDateTime.parse(g.getString("starts_at")).toInstant(),
                    endsAt = OffsetDateTime.parse(g.getString("ends_at")).toInstant(),
                    spotsLeft = g.optInt("capacity") - (people?.length() ?: if (event) 0 else 1),
                    ownerFirstName = if (event) "" else people?.optString(0).orEmpty(),
                    title = if (event) g.optString("title") else "",
                )
            }.filter { it.spotsLeft > 0 },
            distanceKm = if (f.isNull("distance_km")) null else f.optDouble("distance_km"),
        )
    }
}

/** The councils (slug, name), for the search's council filter. */
suspend fun fetchCouncils(): List<Pair<String, String>> {
    val list = Api.getJson(Api.baseUrl + "/tenants").getJSONArray("tenants")
    return List(list.length()) { list.getJSONObject(it).let { t -> t.getString("slug") to t.getString("name") } }
}

/** POST /me/bookings/{id}/join: a spot in someone's public booking. */
suspend fun joinPublicGame(token: String, id: String) {
    Api.send("POST", Api.baseUrl + "/me/bookings/$id/join", token)
}
