package gal.tribo.librequenda.data

import org.json.JSONObject
import java.time.Instant

/** A request to be verified as a council's resident, not decided yet (GET /me, resident_requests). */
data class ResidentRequest(val tenantSlug: String, val tenantName: String, val requestedAt: Instant?)

/** What the request form asks, as the person typed it. */
data class ResidentForm(val dni: String, val fullName: String, val postalCode: String, val street: String) {
    /** A light check before sending; the API checks the DNI/NIE and the rest for real. */
    val complete: Boolean
        get() = dni.isNotBlank() && fullName.isNotBlank() && street.isNotBlank() && postalCodeValid

    val postalCodeValid: Boolean get() = POSTAL_CODE.matches(postalCode.trim())

    private companion object {
        val POSTAL_CODE = Regex("\\d{5}")
    }
}

/**
 * POST /tenants/{council}/resident-request: asks the council to verify the person
 * as its resident. The council uses the data only to check the census, and deletes it on deciding.
 */
suspend fun requestResidency(token: String, councilSlug: String, form: ResidentForm) {
    Api.postJson(
        Api.baseUrl + "/tenants/$councilSlug/resident-request",
        JSONObject()
            .put("dni", form.dni.trim().uppercase())
            .put("full_name", form.fullName.trim())
            .put("postal_code", form.postalCode.trim())
            .put("street", form.street.trim()),
        token,
    )
}

/** DELETE /tenants/{council}/resident-request: withdraws the person's own pending request. */
suspend fun withdrawResidencyRequest(token: String, councilSlug: String) {
    Api.send("DELETE", Api.baseUrl + "/tenants/$councilSlug/resident-request", token)
}
