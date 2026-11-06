package gal.tribo.librequenda.ui.profile

import android.content.ClipData
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.graphics.Color
import androidx.compose.material3.Icon
import androidx.compose.foundation.clickable
import android.content.ClipboardManager
import android.content.Context
import android.content.Intent
import android.net.Uri
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.PickVisualMediaRequest
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.pluralStringResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import coil.compose.AsyncImage
import gal.tribo.librequenda.R
import gal.tribo.librequenda.SessionState
import gal.tribo.librequenda.data.Account
import gal.tribo.librequenda.data.AccountApi
import gal.tribo.librequenda.data.Api
import gal.tribo.librequenda.data.CalendarApi
import gal.tribo.librequenda.data.Me
import gal.tribo.librequenda.data.StoredImage
import gal.tribo.librequenda.data.deleteAccount
import gal.tribo.librequenda.data.exportMyData
import gal.tribo.librequenda.data.updateProfile
import gal.tribo.librequenda.data.uploadImage
import gal.tribo.librequenda.ui.components.DangerButton
import gal.tribo.librequenda.ui.components.ConfirmSheet
import gal.tribo.librequenda.ui.components.DateSheet
import gal.tribo.librequenda.ui.components.GhostButton
import gal.tribo.librequenda.ui.components.InitialsAvatar
import gal.tribo.librequenda.ui.components.LqPickerField
import gal.tribo.librequenda.ui.components.LqTextField
import gal.tribo.librequenda.ui.components.Notice
import gal.tribo.librequenda.ui.components.PrimaryButton
import gal.tribo.librequenda.ui.components.SecondaryButton
import gal.tribo.librequenda.ui.components.SettingsCard
import gal.tribo.librequenda.ui.components.SettingsSection
import gal.tribo.librequenda.ui.components.SectionDivider
import androidx.compose.foundation.layout.Box
import gal.tribo.librequenda.ui.theme.Lq
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.launch
import java.time.Instant
import java.time.LocalDate
import java.time.ZoneOffset
import java.time.format.DateTimeFormatter
import java.time.format.FormatStyle

/** What the profile needs from the app: the session's token and what to do after saving. */
interface ProfileActions {
    suspend fun <T> authed(block: suspend (token: String) -> T): T
    fun meUpdated(me: Me)
    fun reloadMe()
    fun accountDeleted()
}

/**
 * /perfil, as on the web: one page in sections. Personal data, the sign-in
 * account, the calendar and «your data» (export, delete). The preferences are
 * in the menu, beside the app's language.
 */
@Composable
fun ProfileScreen(
    session: SessionState,
    actions: ProfileActions,
    onSignIn: () -> Unit,
    onRegister: () -> Unit,
    onSignOut: () -> Unit,
    modifier: Modifier = Modifier,
) {
    // a page of its own (from the menu): its bar carries the title. Sections lie on
    // the page, one after another, with a band between them (no boxes: an app, not a web)
    Column(modifier.fillMaxSize().background(Lq.colors.card).verticalScroll(rememberScrollState()).navigationBarsPadding()) {
        val me = session.me
        val pad = Modifier.padding(horizontal = 16.dp, vertical = 24.dp)
        when {
            !session.signedIn && session.accountDeleted -> Column(pad, verticalArrangement = Arrangement.spacedBy(12.dp)) {
                Text(stringResource(R.string.deleted_title), style = MaterialTheme.typography.titleMedium, color = Lq.colors.foreground)
                Text(stringResource(R.string.deleted_body, stringResource(R.string.app_name)), color = Lq.colors.mutedForeground)
            }
            !session.signedIn -> Column(pad, verticalArrangement = Arrangement.spacedBy(16.dp)) {
                Text(stringResource(R.string.profile_signed_out), color = Lq.colors.foreground)
                PrimaryButton(stringResource(R.string.sign_in), onSignIn, Modifier.fillMaxWidth(), enabled = !session.busy)
                SecondaryButton(stringResource(R.string.create_account), onRegister, Modifier.fillMaxWidth(), enabled = !session.busy)
            }
            me == null -> LinearProgressIndicator(pad.fillMaxWidth())
            else -> {
                // the date of birth is required before booking
                if (!me.profileComplete) Box(Modifier.padding(start = 16.dp, end = 16.dp, top = 16.dp)) {
                    Notice(stringResource(R.string.profile_incomplete), error = true)
                }
                Box(pad) { DetailsSection(me, actions) }
                SectionDivider()
                Box(pad) { ResidentSection(me) }
                SectionDivider()
                Box(pad) { AccountSection(actions) }
                SectionDivider()
                Box(pad) { CalendarSection(actions) }
                SectionDivider()
                Box(pad) { PrivacySection(me, actions) }
                SectionDivider()
                Box(pad) { SecondaryButton(stringResource(R.string.sign_out), onSignOut, Modifier.fillMaxWidth()) }
            }
        }
    }
}

// --- personal data ---------------------------------------------------------------

private val IMAGE_TYPES = setOf("image/jpeg", "image/png", "image/webp")
private const val MAX_IMAGE = 5 * 1024 * 1024

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun DetailsSection(me: Me, actions: ProfileActions) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    // without a photo of their own, the one on their sign-in account shows (no ref: saving keeps none here)
    var avatar by remember(me.id) { mutableStateOf(me.avatarUrl?.let { StoredImage(me.avatarRef, it) }) }
    var birthDate by rememberSaveable(me.id) { mutableStateOf(me.birthDate) }
    var phone by rememberSaveable(me.id) { mutableStateOf(me.phone) }
    var picking by remember { mutableStateOf(false) }
    var uploading by remember { mutableStateOf(false) }
    var saving by remember { mutableStateOf(false) }
    var notice by remember { mutableStateOf<String?>(null) }
    var error by remember { mutableStateOf<String?>(null) }
    val texts = Texts.current()

    val pickPhoto = rememberLauncherForActivityResult(ActivityResultContracts.PickVisualMedia()) { uri ->
        if (uri == null) return@rememberLauncherForActivityResult
        val type = context.contentResolver.getType(uri)
        if (type !in IMAGE_TYPES) { error = texts.wrongType; return@rememberLauncherForActivityResult }
        scope.launch {
            uploading = true; error = null; notice = null
            try {
                val bytes = context.contentResolver.openInputStream(uri)!!.use { it.readBytes() }
                if (bytes.size > MAX_IMAGE) { error = texts.tooLarge; return@launch }
                avatar = actions.authed { uploadImage(it, "users", bytes, type!!) }
            } catch (c: CancellationException) { throw c } catch (e: Exception) {
                error = e.userMessage(texts)
            } finally { uploading = false }
        }
    }

    SettingsSection(stringResource(R.string.profile_details), stringResource(R.string.profile_subtitle)) {
        SettingsCard {
            // the photo
            Text(stringResource(R.string.profile_photo), style = MaterialTheme.typography.labelLarge, color = Lq.colors.foreground)
            Row(verticalAlignment = Alignment.CenterVertically) {
                if (avatar != null) {
                    AsyncImage(avatar!!.url, contentDescription = null, contentScale = ContentScale.Crop, modifier = Modifier.size(72.dp).clip(CircleShape).background(Lq.colors.muted))
                } else {
                    InitialsAvatar(me.initials, size = 72.dp)
                }
                Spacer(Modifier.size(16.dp))
                Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                    // beside the photo: the average others gave them after public bookings; nothing until someone did
                    me.rating?.let { gal.tribo.librequenda.ui.components.RatingBadge(it, Modifier.padding(bottom = 4.dp)) }
                    SecondaryButton(
                        stringResource(if (uploading) R.string.image_uploading else if (avatar != null) R.string.image_change else R.string.image_upload),
                        { pickPhoto.launch(PickVisualMediaRequest(ActivityResultContracts.PickVisualMedia.ImageOnly)) },
                        enabled = !uploading,
                        height = 40.dp,
                    )
                    if (avatar != null) GhostButton(stringResource(R.string.image_remove), { avatar = null }, enabled = !uploading)
                }
            }
            Text(stringResource(R.string.image_hint), style = MaterialTheme.typography.bodySmall, color = Lq.colors.mutedForeground)
            HorizontalDivider(color = Lq.colors.border)

            LqPickerField(
                label = stringResource(R.string.birth_date),
                value = formatDate(birthDate),
                placeholder = stringResource(R.string.birth_date_placeholder),
                onClick = { picking = true },
                help = stringResource(R.string.birth_date_help),
            )
            LqTextField(
                label = "${stringResource(R.string.phone)} (${stringResource(R.string.optional)})",
                value = phone,
                onValueChange = { if (it.length <= 30) phone = it },
                help = stringResource(R.string.phone_help),
                keyboardType = KeyboardType.Phone,
            )
            Notice(notice, error = false)
            Notice(error, error = true)
            PrimaryButton(
                stringResource(if (saving) R.string.saving else R.string.save_profile),
                {
                    if (birthDate.isBlank()) { notice = null; error = texts.birthDateRequired; return@PrimaryButton }
                    scope.launch {
                        saving = true; error = null; notice = null
                        try {
                            val saved = actions.authed { updateProfile(it, birthDate, phone, avatar?.ref.orEmpty()) }
                            actions.meUpdated(saved)
                            notice = texts.profileSaved
                        } catch (c: CancellationException) { throw c } catch (e: Exception) {
                            error = e.userMessage(texts)
                        } finally { saving = false }
                    }
                },
                Modifier.fillMaxWidth(),
                enabled = !saving && !uploading,
            )
        }
    }

    // the calendar comes up from the bottom (the app has no dialogs)
    if (picking) {
        val today = LocalDate.now()
        DateSheet(
            stringResource(R.string.birth_date),
            selected = birthDate.takeIf { it.isNotBlank() }?.let(LocalDate::parse) ?: today.minusYears(30),
            from = LocalDate.of(1906, 1, 1),
            until = today,
            onSelect = { birthDate = it.toString() },
            onDismiss = { picking = false },
        )
    }
}

// --- the council they are a resident of -------------------------------------------

/** As the web's /perfil: the council that verified them as its resident (one at most), or how to be verified. */
@Composable
private fun ResidentSection(me: Me) {
    SettingsSection(stringResource(R.string.resident_of_title), stringResource(R.string.resident_of_help)) {
        if (me.residentCouncils.isEmpty()) {
            Text(stringResource(R.string.resident_of_none), style = MaterialTheme.typography.bodyMedium, color = Lq.colors.mutedForeground)
        } else {
            Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                me.residentCouncils.forEach { c ->
                    Column(verticalArrangement = Arrangement.spacedBy(2.dp)) {
                        Text(c.name, style = MaterialTheme.typography.bodyLarge, color = Lq.colors.foreground)
                        Text(stringResource(R.string.resident_of_since, formatDate(c.since)), style = MaterialTheme.typography.bodySmall, color = Lq.colors.mutedForeground)
                    }
                }
            }
        }
    }
}

@Composable
private fun formatDate(iso: String): String {
    if (iso.isBlank()) return ""
    val locale = LocalConfiguration.current.locales[0]
    return runCatching { LocalDate.parse(iso).format(DateTimeFormatter.ofLocalizedDate(FormatStyle.LONG).withLocale(locale)) }.getOrDefault(iso)
}

// --- the sign-in account ---------------------------------------------------------

@Composable
private fun AccountSection(actions: ProfileActions) {
    var account by remember { mutableStateOf<Account?>(null) }
    var failed by remember { mutableStateOf(false) }
    var attempt by remember { mutableIntStateOf(0) }
    LaunchedEffect(attempt) {
        failed = false
        try { account = actions.authed { AccountApi.get(it) } } catch (c: CancellationException) { throw c } catch (_: Exception) { failed = true }
    }
    val reload: () -> Unit = { attempt++ }

    SettingsSection(stringResource(R.string.account_title), stringResource(R.string.account_help, stringResource(R.string.app_name))) {
        val a = account
        when {
            a != null -> {
                NameCard(a, actions) { reload(); actions.reloadMe() }
                EmailCard(a, actions) { reload(); actions.reloadMe() }
                PasswordCard(a, actions)
            }
            failed -> SettingsCard {
                Text(stringResource(R.string.account_load_error), color = Lq.colors.foreground)
                SecondaryButton(stringResource(R.string.retry), reload, height = 40.dp)
            }
            else -> LinearProgressIndicator(Modifier.fillMaxWidth())
        }
    }
}

/** A card's busy flag and its green or red line, and how to run an action with them. */
private class CardState(private val texts: Texts) {
    var busy by mutableStateOf(false)
    var notice by mutableStateOf<String?>(null)
    var error by mutableStateOf<String?>(null)

    suspend fun run(done: String?, block: suspend () -> Unit) {
        busy = true; notice = null; error = null
        try { block(); notice = done } catch (c: CancellationException) { throw c } catch (e: Exception) {
            error = e.userMessage(texts)
        } finally { busy = false }
    }
}

@Composable
private fun rememberCardState(): CardState {
    val texts = Texts.current()
    return remember { CardState(texts) }
}

@Composable
private fun NameCard(account: Account, actions: ProfileActions, onSaved: () -> Unit) {
    val scope = rememberCoroutineScope()
    val card = rememberCardState()
    var given by rememberSaveable { mutableStateOf(account.givenName) }
    var family by rememberSaveable { mutableStateOf(account.familyName) }
    val saved = stringResource(R.string.name_saved)
    SettingsCard(stringResource(R.string.account_name)) {
        LqTextField(stringResource(R.string.given_name), given, { if (it.length <= 100) given = it })
        LqTextField(stringResource(R.string.family_name), family, { if (it.length <= 100) family = it })
        Notice(card.notice, false); Notice(card.error, true)
        SecondaryButton(
            stringResource(if (card.busy) R.string.saving else R.string.save_name),
            { scope.launch { card.run(saved) { actions.authed { AccountApi.setName(it, given.trim(), family.trim()) }; onSaved() } } },
            Modifier.fillMaxWidth(),
            enabled = !card.busy && given.isNotBlank() && family.isNotBlank(),
        )
    }
}

@Composable
private fun EmailCard(account: Account, actions: ProfileActions, onVerified: () -> Unit) {
    val scope = rememberCoroutineScope()
    val card = rememberCardState()
    var code by rememberSaveable { mutableStateOf("") }
    var editing by rememberSaveable { mutableStateOf(false) }
    var newEmail by rememberSaveable { mutableStateOf("") }
    var sentTo by rememberSaveable { mutableStateOf<String?>(null) }
    val verifiedText = stringResource(R.string.email_verified)
    val sentText = stringResource(R.string.code_sent, sentTo ?: account.email)
    val newSentText = stringResource(R.string.code_sent, newEmail.trim())

    SettingsCard(stringResource(R.string.account_email)) {
        Text(
            "${account.email} · ${stringResource(if (account.emailVerified) R.string.verified else R.string.not_verified)}",
            style = MaterialTheme.typography.bodyLarge,
            color = Lq.colors.foreground,
        )
        if (!account.emailVerified) {
            LqTextField(
                label = stringResource(R.string.code),
                value = code,
                onValueChange = { code = it },
                help = stringResource(R.string.code_help, sentTo ?: account.email),
            )
            PrimaryButton(
                stringResource(R.string.verify),
                { scope.launch { card.run(verifiedText) { actions.authed { AccountApi.verifyEmail(it, code.trim()) }; code = ""; onVerified() } } },
                Modifier.fillMaxWidth(),
                enabled = !card.busy && code.isNotBlank(),
            )
            GhostButton(stringResource(R.string.resend), { scope.launch { card.run(sentText) { actions.authed { AccountApi.resendEmailCode(it) } } } }, enabled = !card.busy)
        }
        if (editing) {
            LqTextField(stringResource(R.string.new_email), newEmail, { newEmail = it }, help = stringResource(R.string.new_email_help), keyboardType = KeyboardType.Email)
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                SecondaryButton(
                    stringResource(R.string.send_code),
                    {
                        scope.launch {
                            card.run(newSentText) {
                                actions.authed { AccountApi.setEmail(it, newEmail.trim()) }
                                sentTo = newEmail.trim(); editing = false; onVerified()
                            }
                        }
                    },
                    enabled = !card.busy && newEmail.contains('@'),
                    height = 44.dp,
                )
                GhostButton(stringResource(R.string.cancel), { editing = false })
            }
        } else {
            SecondaryButton(stringResource(R.string.change_email), { editing = true }, height = 44.dp)
        }
        Notice(card.notice, false); Notice(card.error, true)
    }
}

@Composable
private fun PasswordCard(account: Account, actions: ProfileActions) {
    val scope = rememberCoroutineScope()
    val card = rememberCardState()
    var current by remember { mutableStateOf("") }
    var next by remember { mutableStateOf("") }
    var repeat by remember { mutableStateOf("") }
    val rules = account.rules
    val checks = buildList {
        add((next.length >= rules.minLength) to pluralStringResource(R.plurals.rule_length, rules.minLength, rules.minLength))
        if (rules.uppercase) add(next.any { it.isUpperCase() } to stringResource(R.string.rule_uppercase))
        if (rules.lowercase) add(next.any { it.isLowerCase() } to stringResource(R.string.rule_lowercase))
        if (rules.number) add(next.any { it.isDigit() } to stringResource(R.string.rule_number))
        if (rules.symbol) add(next.any { !it.isLetterOrDigit() } to stringResource(R.string.rule_symbol))
    }
    val mismatch = repeat.isNotEmpty() && repeat != next
    val saved = stringResource(R.string.password_saved)

    SettingsCard(stringResource(R.string.password)) {
        LqTextField(stringResource(R.string.current_password), current, { current = it }, password = true)
        LqTextField(stringResource(R.string.new_password), next, { next = it }, password = true)
        Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
            checks.forEach { (ok, label) ->
                Text(
                    (if (ok) "✓ " else "• ") + label,
                    style = MaterialTheme.typography.bodySmall,
                    color = if (ok) Lq.colors.pillPositiveInk else Lq.colors.mutedForeground,
                )
            }
        }
        LqTextField(
            stringResource(R.string.repeat_password), repeat, { repeat = it }, password = true,
            error = if (mismatch) stringResource(R.string.mismatch) else null,
        )
        Notice(card.notice, false); Notice(card.error, true)
        SecondaryButton(
            stringResource(if (card.busy) R.string.saving else R.string.save_password),
            {
                scope.launch {
                    card.run(saved) {
                        actions.authed { AccountApi.setPassword(it, current, next) }
                        current = ""; next = ""; repeat = ""
                    }
                }
            },
            Modifier.fillMaxWidth(),
            enabled = !card.busy && !mismatch && repeat == next && current.isNotEmpty() && checks.all { it.first },
        )
    }
}

// --- calendar --------------------------------------------------------------------

@Composable
private fun CalendarSection(actions: ProfileActions) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val card = rememberCardState()
    var url by remember { mutableStateOf<String?>(null) }
    var copied by remember { mutableStateOf(false) }
    LaunchedEffect(Unit) {
        try { url = actions.authed { CalendarApi.get(it) } } catch (c: CancellationException) { throw c } catch (_: Exception) { url = "" }
    }
    val name = stringResource(R.string.app_name)

    SettingsSection(stringResource(R.string.calendar_title), stringResource(R.string.calendar_help)) {
        SettingsCard {
            val u = url
            when {
                u == null -> LinearProgressIndicator(Modifier.fillMaxWidth())
                u.isBlank() -> {
                    Text(stringResource(R.string.calendar_how), color = Lq.colors.mutedForeground, style = MaterialTheme.typography.bodyMedium)
                    PrimaryButton(stringResource(R.string.calendar_turn_on), {
                        scope.launch { card.run(null) { url = actions.authed { CalendarApi.reset(it) } } }
                    }, Modifier.fillMaxWidth(), enabled = !card.busy)
                }
                else -> {
                    val webcal = u.replaceFirst(Regex("^https?:"), "webcal:")
                    fun outlook(host: String) = "https://$host/calendar/0/addfromweb?url=${Uri.encode(u)}&name=${Uri.encode(name)}"
                    // each calendar app as a row with its own logo, as an app shows «open in…»
                    val links = listOf(
                        Triple(R.drawable.ic_brand_google_calendar, Color(0xFF4285F4), R.string.calendar_google) to "https://calendar.google.com/calendar/render?cid=${Uri.encode(webcal)}",
                        Triple(R.drawable.ic_brand_outlook, Color(0xFF0078D4), R.string.calendar_outlook) to outlook("outlook.live.com"),
                        Triple(R.drawable.ic_brand_outlook, Color(0xFF0078D4), R.string.calendar_outlook_work) to outlook("outlook.office.com"),
                        Triple(R.drawable.ic_brand_apple, Lq.colors.foreground, R.string.calendar_apple) to webcal,
                    )
                    Column {
                        links.forEachIndexed { i, (brand, link) ->
                            if (i > 0) HorizontalDivider(color = Lq.colors.border)
                            CalendarAppRow(brand.first, brand.second, stringResource(brand.third)) { open(context, link) }
                        }
                    }
                    Text(stringResource(R.string.calendar_google_slow), style = MaterialTheme.typography.bodySmall, color = Lq.colors.mutedForeground)
                    HorizontalDivider(color = Lq.colors.border)
                    LqTextField(stringResource(R.string.calendar_address), u, {}, help = stringResource(R.string.calendar_private))
                    SecondaryButton(stringResource(if (copied) R.string.calendar_copied else R.string.calendar_copy), {
                        context.getSystemService(ClipboardManager::class.java).setPrimaryClip(ClipData.newPlainText(name, u))
                        copied = true
                    }, Modifier.fillMaxWidth(), height = 44.dp)
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        SecondaryButton(stringResource(R.string.calendar_reset), {
                            scope.launch { card.run(null) { url = actions.authed { CalendarApi.reset(it) }; copied = false } }
                        }, enabled = !card.busy, height = 44.dp)
                        GhostButton(stringResource(R.string.calendar_turn_off), {
                            scope.launch { card.run(null) { actions.authed { CalendarApi.turnOff(it) }; url = "" } }
                        }, enabled = !card.busy)
                    }
                    Text(stringResource(R.string.calendar_reset_hint), style = MaterialTheme.typography.bodySmall, color = Lq.colors.mutedForeground)
                }
            }
            Notice(card.error, true)
        }
    }
}

/** A calendar app to subscribe in: its logo on a pale tile, its name, and «opens outside». */
@Composable
private fun CalendarAppRow(@androidx.annotation.DrawableRes logo: Int, tint: Color, label: String, onClick: () -> Unit) {
    Row(
        Modifier.fillMaxWidth().clickable(onClick = onClick).padding(vertical = 14.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        androidx.compose.foundation.layout.Box(
            Modifier.size(44.dp).clip(androidx.compose.foundation.shape.RoundedCornerShape(12.dp)).background(Lq.colors.muted),
            contentAlignment = Alignment.Center,
        ) {
            Icon(painterResource(logo), contentDescription = null, tint = tint, modifier = Modifier.size(24.dp))
        }
        Spacer(Modifier.size(14.dp))
        Text(label, style = MaterialTheme.typography.bodyLarge, color = Lq.colors.foreground, modifier = Modifier.weight(1f))
        Icon(painterResource(R.drawable.ic_external_link), contentDescription = null, tint = Lq.colors.mutedForeground, modifier = Modifier.size(18.dp))
    }
}

private fun open(context: Context, link: String) {
    runCatching { context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(link))) }
}

// --- your data ---------------------------------------------------------------------

@Composable
private fun PrivacySection(me: Me, actions: ProfileActions) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val card = rememberCardState()
    var preparing by remember { mutableStateOf<String?>(null) }
    var pending by remember { mutableStateOf<Pair<ByteArray, String>?>(null) } // the file, until a place is chosen
    var deleting by remember { mutableStateOf(false) }
    val savedText = stringResource(R.string.export_saved)

    val saveFile = rememberLauncherForActivityResult(ActivityResultContracts.CreateDocument("*/*")) { uri ->
        val file = pending ?: return@rememberLauncherForActivityResult
        pending = null
        if (uri != null) scope.launch {
            card.run(savedText) { context.contentResolver.openOutputStream(uri)!!.use { it.write(file.first) } }
        }
    }

    fun download(format: String) = scope.launch {
        preparing = format
        card.run(null) {
            val (bytes, name) = actions.authed { exportMyData(it, format) }
            pending = bytes to (name ?: "librequenda.$format")
            saveFile.launch(pending!!.second)
        }
        preparing = null
    }

    SettingsSection(stringResource(R.string.privacy_title), stringResource(R.string.privacy_help, stringResource(R.string.app_name))) {
        SettingsCard(stringResource(R.string.export_title)) {
            Text(stringResource(R.string.export_help), style = MaterialTheme.typography.bodyMedium, color = Lq.colors.mutedForeground)
            SecondaryButton(stringResource(if (preparing == "json") R.string.preparing else R.string.export_json), { download("json") }, Modifier.fillMaxWidth(), enabled = preparing == null, height = 44.dp)
            SecondaryButton(stringResource(if (preparing == "csv") R.string.preparing else R.string.export_csv), { download("csv") }, Modifier.fillMaxWidth(), enabled = preparing == null, height = 44.dp)
            Notice(card.notice, false); Notice(card.error, true)
        }
        SettingsCard(stringResource(R.string.delete_title)) {
            Text(stringResource(R.string.delete_help), style = MaterialTheme.typography.bodyMedium, color = Lq.colors.mutedForeground)
            DangerButton(stringResource(R.string.delete_account), { deleting = true }, Modifier.fillMaxWidth())
        }
    }

    if (deleting) DeleteDialog(me.email, actions) { deleting = false }
}

/** Deleting the account, from the bottom: what goes, and typing their email to confirm. */
@Composable
private fun DeleteDialog(email: String, actions: ProfileActions, onClose: () -> Unit) {
    val texts = Texts.current()
    var typed by remember { mutableStateOf("") }
    ConfirmSheet(
        title = stringResource(R.string.delete_dialog_title),
        text = stringResource(R.string.delete_dialog_body),
        confirm = stringResource(R.string.delete_confirm),
        confirming = stringResource(R.string.deleting),
        keep = stringResource(R.string.cancel),
        onDismiss = onClose,
        onConfirm = { actions.authed { deleteAccount(it) }; onClose(); actions.accountDeleted() },
        errorOf = { it.userMessage(texts) },
        enabled = typed.trim().equals(email, ignoreCase = true),
    ) {
        Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
            listOf(R.string.delete_upcoming, R.string.delete_past, R.string.delete_sign_in).forEach {
                Text("• " + stringResource(it), style = MaterialTheme.typography.bodyMedium, color = Lq.colors.mutedForeground)
            }
            LqTextField(stringResource(R.string.delete_confirm_label, email), typed, { typed = it }, keyboardType = KeyboardType.Email)
        }
    }
}

// --- errors --------------------------------------------------------------------------

/** The texts errors need outside composition. */
internal class Texts(
    val generic: String,
    val rateLimited: String,
    val wrongType: String,
    val tooLarge: String,
    val birthDateRequired: String,
    val profileSaved: String,
) {
    companion object {
        @Composable
        fun current() = Texts(
            stringResource(R.string.generic_error),
            stringResource(R.string.rate_limited),
            stringResource(R.string.image_wrong_type),
            stringResource(R.string.image_too_large),
            stringResource(R.string.birth_date_required),
            stringResource(R.string.profile_saved),
        )
    }
}

/** The API's own message (already in the app's language), or a general one. */
internal fun Throwable.userMessage(texts: Texts): String = when {
    this is Api.HttpException && status == 429 -> texts.rateLimited
    this is Api.HttpException && message?.startsWith("HTTP ") == false -> message!!
    else -> texts.generic
}
