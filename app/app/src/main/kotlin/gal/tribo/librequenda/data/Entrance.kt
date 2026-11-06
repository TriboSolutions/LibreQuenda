package gal.tribo.librequenda.data

import android.Manifest
import android.content.Context
import android.content.pm.PackageManager
import android.location.Location
import android.location.LocationManager
import android.net.Uri
import android.os.Build
import androidx.core.content.ContextCompat
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlinx.coroutines.withTimeoutOrNull
import org.json.JSONObject
import kotlin.coroutines.resume

/** Where the phone is, as precisely as it can tell now: the entrance checks it is at the facility. Never stored. */
data class Position(val latitude: Double, val longitude: Double, val accuracy: Float)

class LocationException(val denied: Boolean) : Exception()

/** A fresh, precise fix (GPS when it can); throws LocationException without permission or a fix. */
suspend fun preciseLocation(context: Context): Position {
    if (ContextCompat.checkSelfPermission(context, Manifest.permission.ACCESS_FINE_LOCATION) != PackageManager.PERMISSION_GRANTED) {
        throw LocationException(denied = true)
    }
    val manager = context.getSystemService(LocationManager::class.java) ?: throw LocationException(false)
    val providers = manager.getProviders(true)
    val preferred = buildList {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) add(LocationManager.FUSED_PROVIDER)
        add(LocationManager.GPS_PROVIDER); add(LocationManager.NETWORK_PROVIDER)
    }.filter { it in providers }
    var fix: Location? = null
    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
        for (provider in preferred) {
            fix = withTimeoutOrNull(20_000) {
                suspendCancellableCoroutine { cont -> manager.getCurrentLocation(provider, null, context.mainExecutor) { cont.resume(it) } }
            }
            if (fix != null) break
        }
    }
    // a recent one is good enough when no fresh fix comes (older Android, or indoors)
    if (fix == null) fix = preferred.mapNotNull { runCatching { manager.getLastKnownLocation(it) }.getOrNull() }
        .filter { System.currentTimeMillis() - it.time < 2 * 60_000 }.maxByOrNull { it.time }
    val f = fix ?: throw LocationException(false)
    return Position(f.latitude, f.longitude, f.accuracy)
}

/** What the entrance answered: «checked_in», «booked», or why not («too_far»…); for «not_booked», [reason] says why no turn was booked («slot_booked»…). */
data class EntranceResult(val result: String, val canBook: Boolean = false, val bookingId: String? = null, val reason: String? = null) {
    val granted get() = result in setOf("checked_in", "already_checked_in", "booked")
}

/**
 * POST /tenants/{council}/facilities/{facility}/entrance: with the entry code, the door QR's token or the NFC tag's,
 * and where the phone is. With [book], a free turn is booked on the spot (the flash booking).
 */
suspend fun checkInAtEntrance(
    token: String, councilSlug: String, facilitySlug: String, position: Position,
    code: String? = null, qrToken: String? = null, nfcToken: String? = null, book: Boolean = false,
): EntranceResult {
    val body = JSONObject()
        .put("latitude", position.latitude).put("longitude", position.longitude).put("accuracy", position.accuracy.toDouble())
        .put("book", book)
    code?.let { body.put("code", it) }
    qrToken?.let { body.put("qr_token", it) }
    nfcToken?.let { body.put("nfc_token", it) }
    val json = Api.postJson(Api.baseUrl + "/tenants/$councilSlug/facilities/$facilitySlug/entrance", body, token)
    return EntranceResult(
        json.optString("result"), json.optBoolean("can_book"),
        json.optString("booking_id").takeIf { it.isNotBlank() }, json.optString("reason").takeIf { it.isNotBlank() },
    )
}

/** A facility's door, as its printed QR or NFC tag gives it: /<council>/<facility>/entrada?c=<token> (QR) or ?n=<token> (NFC). */
data class Door(val councilSlug: String, val facilitySlug: String, val token: String)

private fun parseDoor(text: String, param: String): Door? {
    val uri = runCatching { Uri.parse(text.trim()) }.getOrNull() ?: return null
    val parts = uri.pathSegments ?: return null
    val token = runCatching { uri.getQueryParameter(param) }.getOrNull()?.takeIf { it.isNotBlank() } ?: return null
    if (parts.size != 3 || parts[2] != "entrada") return null
    return Door(parts[0], parts[1], token)
}

/** The printed QR's door; null for any other QR. */
fun parseDoorQr(text: String): Door? = parseDoor(text, "c")

/** The NFC tag's door; null for any other tag. */
fun parseNfcTag(text: String): Door? = parseDoor(text, "n")

/** The personal QR (GET /me/qr): it works once, until it expires; a new one every refreshSeconds. */
data class PersonalQr(val code: String, val refreshSeconds: Int)

suspend fun fetchPersonalQr(token: String): PersonalQr {
    val json = Api.getJson(Api.baseUrl + "/me/qr", token)
    return PersonalQr(json.optString("code"), json.optInt("refresh_seconds", 60).coerceAtLeast(10))
}

/** What became of a personal QR: used at a reader, and the flash booking it made (null when none). */
data class PersonalQrStatus(val used: Boolean, val bookingId: String?)

suspend fun fetchPersonalQrStatus(token: String, code: String): PersonalQrStatus {
    val json = Api.getJson(Api.baseUrl + "/me/qr/status?code=" + Uri.encode(code), token)
    return PersonalQrStatus(json.optBoolean("used"), json.optString("booking_id").takeIf { it.isNotBlank() })
}

/** A facility the phone is at (POST /entrance/nearby), nearest first. */
data class NearbyFacility(
    val name: String, val slug: String, val councilName: String, val councilSlug: String,
    val entryCode: Boolean, val printedQr: Boolean, val nfc: Boolean, val radiusM: Int, val staffContacts: List<StaffContact>,
)

/** Where the phone is answers the facilities there, or a location problem (location_required, location_inaccurate). */
data class Nearby(val problem: String, val facilities: List<NearbyFacility>)

suspend fun fetchNearby(token: String, position: Position): Nearby {
    val body = JSONObject().put("latitude", position.latitude).put("longitude", position.longitude).put("accuracy", position.accuracy.toDouble())
    val json = Api.postJson(Api.baseUrl + "/entrance/nearby", body, token)
    val list = json.optJSONArray("facilities")
    val facilities = List(list?.length() ?: 0) { list!!.getJSONObject(it) }.map { f ->
        val contacts = f.optJSONArray("staff_contacts")
        NearbyFacility(
            f.optString("facility_name"), f.optString("facility_slug"), f.optString("tenant_name"), f.optString("tenant_slug"),
            f.optBoolean("entry_code"), f.optBoolean("printed_qr"), f.optBoolean("nfc"), f.optInt("radius_m"),
            List(contacts?.length() ?: 0) { contacts!!.getJSONObject(it) }.map { StaffContact(it.optString("name"), it.optString("phone"), it.optString("place")) },
        )
    }
    return Nearby(json.optString("result"), facilities)
}
