package gal.tribo.librequenda.ui.facility

import android.content.Intent
import android.net.Uri
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import gal.tribo.librequenda.R
import gal.tribo.librequenda.data.Api
import gal.tribo.librequenda.data.Me
import gal.tribo.librequenda.data.ResidentForm
import gal.tribo.librequenda.data.ResidentRequest
import gal.tribo.librequenda.data.fetchMe
import gal.tribo.librequenda.data.requestResidency
import gal.tribo.librequenda.data.withdrawResidencyRequest
import gal.tribo.librequenda.ui.components.ConfirmSheet
import gal.tribo.librequenda.ui.components.LqSheet
import gal.tribo.librequenda.ui.components.LqTextField
import gal.tribo.librequenda.ui.components.Notice
import gal.tribo.librequenda.ui.components.PrimaryButton
import gal.tribo.librequenda.ui.components.SecondaryButton
import gal.tribo.librequenda.ui.profile.ProfileActions
import gal.tribo.librequenda.ui.profile.Texts
import gal.tribo.librequenda.ui.profile.userMessage
import gal.tribo.librequenda.ui.theme.Lq
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.launch
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.time.format.FormatStyle

private val ZONE = ZoneId.of("Europe/Madrid")

/** The council the facility belongs to: who verifies the person, and how to call it. */
data class Council(val slug: String, val name: String, val phone: String)

/**
 * Signed in and not verified by the facility's council: a row to ask to be
 * verified (a sheet with the council's phone and the request form), or, once
 * asked, the request waiting for review, which can be withdrawn.
 */
@Composable
fun ResidentRequestRow(council: Council, me: Me, actions: ProfileActions) {
    var sheet by remember { mutableStateOf<Sheet?>(null) } // which, chosen on the tap: it stays while /me reloads
    var withdrawing by remember { mutableStateOf(false) }
    val pending = me.residentRequest(council.slug)
    Row(
        Modifier.fillMaxWidth().heightIn(min = 56.dp).clickable(role = Role.Button) { sheet = if (pending != null) Sheet.Pending else Sheet.Request }.padding(horizontal = 16.dp, vertical = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(painterResource(if (pending != null) R.drawable.ic_clock else R.drawable.ic_home), contentDescription = null, tint = Lq.colors.foreground, modifier = Modifier.size(22.dp))
        Spacer(Modifier.size(16.dp))
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
            Text(
                if (pending != null) stringResource(R.string.resident_request_pending, requestedOn(pending)) else stringResource(R.string.resident_request_row),
                style = MaterialTheme.typography.bodyLarge, color = Lq.colors.foreground,
            )
            Text(
                stringResource(if (pending != null) R.string.resident_request_pending_help else R.string.resident_request_row_help),
                style = MaterialTheme.typography.bodySmall, color = Lq.colors.mutedForeground,
            )
        }
        Icon(painterResource(R.drawable.ic_chevron_right), contentDescription = null, tint = Lq.colors.mutedForeground, modifier = Modifier.size(20.dp))
    }
    when (sheet) {
        Sheet.Pending -> if (pending != null) PendingSheet(council, pending, onWithdraw = { sheet = null; withdrawing = true }, onDismiss = { sheet = null })
        Sheet.Request -> RequestSheet(council, me, actions, onDismiss = { sheet = null })
        null -> {}
    }
    if (withdrawing) {
        val texts = Texts.current()
        ConfirmSheet(
            title = stringResource(R.string.resident_withdraw_title),
            text = stringResource(R.string.resident_withdraw_text, council.name),
            confirm = stringResource(R.string.resident_withdraw),
            confirming = stringResource(R.string.resident_withdrawing),
            keep = stringResource(R.string.resident_keep),
            onDismiss = { withdrawing = false },
            onConfirm = {
                actions.authed { withdrawResidencyRequest(it, council.slug) }
                refreshMe(actions)
                withdrawing = false
            },
            errorOf = { it.userMessage(texts) },
        )
    }
}

private enum class Sheet { Request, Pending }

/** When the request was sent, as a date inside a sentence. */
@Composable
private fun requestedOn(r: ResidentRequest): String =
    r.requestedAt?.atZone(ZONE)?.format(DateTimeFormatter.ofLocalizedDate(FormatStyle.LONG).withLocale(LocalConfiguration.current.locales[0])).orEmpty()

/** /me again: whether a request is pending (or the person verified) shows straight away. */
private suspend fun refreshMe(actions: ProfileActions) {
    runCatching { actions.authed { fetchMe(it) } }.onSuccess(actions::meUpdated)
}

/** The council's phone, a tap away from the dialler. */
@Composable
private fun CallRow(phone: String) {
    val context = LocalContext.current
    Row(
        Modifier.fillMaxWidth().heightIn(min = 48.dp).clickable(role = Role.Button) {
            runCatching { context.startActivity(Intent(Intent.ACTION_DIAL, Uri.parse("tel:" + phone.filterNot(Char::isWhitespace)))) }
        }.padding(vertical = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(painterResource(R.drawable.ic_phone), contentDescription = null, tint = Lq.colors.primary, modifier = Modifier.size(24.dp))
        Spacer(Modifier.size(18.dp))
        Column(Modifier.weight(1f)) {
            Text(phone, style = MaterialTheme.typography.bodyLarge, color = Lq.colors.foreground)
            Text(stringResource(R.string.resident_call), style = MaterialTheme.typography.bodySmall, color = Lq.colors.mutedForeground)
        }
        Icon(painterResource(R.drawable.ic_chevron_right), contentDescription = null, tint = Lq.colors.mutedForeground, modifier = Modifier.size(20.dp))
    }
}

/** How to be verified: call the council, or send it the data it checks against the census. */
@Composable
private fun RequestSheet(council: Council, me: Me, actions: ProfileActions, onDismiss: () -> Unit) {
    val scope = rememberCoroutineScope()
    val texts = Texts.current()
    var dni by remember { mutableStateOf("") }
    var fullName by remember { mutableStateOf(me.fullName) }
    var postalCode by remember { mutableStateOf("") }
    var street by remember { mutableStateOf("") }
    var busy by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    val form = ResidentForm(dni, fullName, postalCode, street)
    LqSheet(stringResource(R.string.resident_sheet_title, council.name), onDismiss = { if (!busy) onDismiss() }) {
        Text(stringResource(R.string.resident_sheet_text, council.name), style = MaterialTheme.typography.bodyLarge, color = Lq.colors.foreground)
        if (council.phone.isNotBlank()) {
            Spacer(Modifier.height(8.dp))
            CallRow(council.phone)
        }
        HorizontalDivider(Modifier.padding(vertical = 12.dp), color = Lq.colors.border)
        Column(verticalArrangement = Arrangement.spacedBy(16.dp)) {
            Text(stringResource(R.string.resident_form_title), style = MaterialTheme.typography.titleMedium, color = Lq.colors.foreground)
            LqTextField(stringResource(R.string.resident_dni), dni, { if (it.length <= 12) dni = it }, enabled = !busy)
            LqTextField(stringResource(R.string.resident_full_name), fullName, { if (it.length <= 200) fullName = it }, enabled = !busy)
            LqTextField(
                stringResource(R.string.resident_postal_code), postalCode, { v -> if (v.length <= 5 && v.all(Char::isDigit)) postalCode = v },
                keyboardType = KeyboardType.Number, enabled = !busy,
                error = if (postalCode.length == 5 && !form.postalCodeValid) stringResource(R.string.resident_postal_code_invalid) else null,
            )
            LqTextField(stringResource(R.string.resident_street), street, { if (it.length <= 200) street = it }, help = stringResource(R.string.resident_street_help), enabled = !busy)
            Text(stringResource(R.string.resident_privacy), style = MaterialTheme.typography.bodySmall, color = Lq.colors.mutedForeground)
            Notice(error, error = true)
            PrimaryButton(
                stringResource(if (busy) R.string.resident_sending else R.string.resident_send),
                {
                    scope.launch {
                        busy = true; error = null
                        try {
                            actions.authed { requestResidency(it, council.slug, form) }
                            refreshMe(actions)
                            onDismiss()
                        } catch (c: CancellationException) {
                            throw c
                        } catch (e: Exception) {
                            error = e.userMessage(texts)
                            // already pending or verified elsewhere (the web, the council): /me shows how things are
                            if (e is Api.HttpException && e.status == 409) refreshMe(actions)
                        }
                        busy = false
                    }
                },
                Modifier.fillMaxWidth(),
                enabled = form.complete && !busy,
            )
        }
    }
}

/** The request waiting for the council: when it went, and the way to take it back. */
@Composable
private fun PendingSheet(council: Council, pending: ResidentRequest, onWithdraw: () -> Unit, onDismiss: () -> Unit) {
    LqSheet(stringResource(R.string.resident_pending_title), onDismiss) {
        Text(stringResource(R.string.resident_request_pending, requestedOn(pending)) + ".", style = MaterialTheme.typography.bodyLarge, color = Lq.colors.foreground)
        Spacer(Modifier.height(6.dp))
        Text(stringResource(R.string.resident_pending_text, council.name), style = MaterialTheme.typography.bodyMedium, color = Lq.colors.mutedForeground)
        if (council.phone.isNotBlank()) {
            Spacer(Modifier.height(8.dp))
            CallRow(council.phone)
        }
        Spacer(Modifier.height(20.dp))
        SecondaryButton(stringResource(R.string.resident_withdraw), onWithdraw, Modifier.fillMaxWidth())
    }
}
