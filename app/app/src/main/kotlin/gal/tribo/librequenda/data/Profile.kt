package gal.tribo.librequenda.data

import org.json.JSONObject

// What /perfil edits, as the web's profile page does (web/src/lib/api.ts).

/** PUT /me/profile: the personal data (date of birth, phone, photo by its storage reference). */
suspend fun updateProfile(token: String, birthDate: String, phone: String, avatarRef: String): Me =
    Api.send("PUT", Api.baseUrl + "/me/profile", token, JSONObject().apply {
        put("birth_date", birthDate); put("phone", phone); put("avatar_ref", avatarRef)
    })!!.toMe()

/** PUT /me/preferences: how the app speaks to them and what it shows or sends. */
suspend fun updatePreferences(token: String, language: String, emails: Boolean, reminders: Boolean, recommendations: Boolean): Me =
    Api.send("PUT", Api.baseUrl + "/me/preferences", token, JSONObject().apply {
        put("preferred_language", language); put("email_notifications", emails)
        put("email_reminders", reminders); put("show_recommendations", recommendations)
    })!!.toMe()

/** An uploaded image: forms save the ref; screens only show the url. */
data class StoredImage(val ref: String, val url: String)

suspend fun uploadImage(token: String, folder: String, bytes: ByteArray, mimeType: String, fields: Map<String, String> = emptyMap()): StoredImage {
    val json = Api.upload(folder, bytes, mimeType, token, fields)
    return StoredImage(json.getString("ref"), Api.absolute(json.getString("url")))
}

/** The sign-in account, kept by Zitadel and edited through the API. */
data class PasswordRules(val minLength: Int, val uppercase: Boolean, val lowercase: Boolean, val number: Boolean, val symbol: Boolean)
data class Account(val givenName: String, val familyName: String, val email: String, val emailVerified: Boolean, val rules: PasswordRules)

object AccountApi {
    private val base get() = Api.baseUrl + "/me/account"

    suspend fun get(token: String): Account {
        val j = Api.getJson(base, token)
        val r = j.optJSONObject("password_rules") ?: JSONObject()
        return Account(
            j.optString("given_name"), j.optString("family_name"), j.optString("email"), j.optBoolean("email_verified"),
            PasswordRules(r.optInt("min_length", 8), r.optBoolean("uppercase"), r.optBoolean("lowercase"), r.optBoolean("number"), r.optBoolean("symbol")),
        )
    }

    suspend fun setName(token: String, given: String, family: String) {
        Api.send("PUT", "$base/name", token, JSONObject().put("given_name", given).put("family_name", family))
    }

    suspend fun setPassword(token: String, current: String, next: String) {
        Api.send("PUT", "$base/password", token, JSONObject().put("current_password", current).put("new_password", next))
    }

    /** The new address gets a code to type here. */
    suspend fun setEmail(token: String, email: String) {
        Api.send("PUT", "$base/email", token, JSONObject().put("email", email))
    }

    suspend fun resendEmailCode(token: String) {
        Api.send("POST", "$base/email/resend", token)
    }

    suspend fun verifyEmail(token: String, code: String) {
        Api.send("POST", "$base/email/verify", token, JSONObject().put("code", code))
    }
}

/** The bookings as a calendar subscription (iCalendar); "" when it is off. */
object CalendarApi {
    private val base get() = Api.baseUrl + "/me/calendar"
    suspend fun get(token: String): String = Api.getJson(base, token).optString("url")
    /** Turns it on, or gives it a new address: the old one stops working. */
    suspend fun reset(token: String): String = Api.send("POST", base, token)?.optString("url").orEmpty()
    suspend fun turnOff(token: String) { Api.send("DELETE", base, token) }
}

/** GET /me/export: everything LibreQuenda keeps about them, as JSON or CSV. */
suspend fun exportMyData(token: String, format: String): Pair<ByteArray, String?> =
    Api.download(Api.baseUrl + "/me/export?format=$format", token)

/** DELETE /me: the account, for good. Upcoming bookings are cancelled. */
suspend fun deleteAccount(token: String) {
    Api.send("DELETE", Api.baseUrl + "/me", token)
}
