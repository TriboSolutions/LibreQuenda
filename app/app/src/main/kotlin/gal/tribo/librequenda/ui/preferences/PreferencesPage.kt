package gal.tribo.librequenda.ui.preferences

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import gal.tribo.librequenda.R
import gal.tribo.librequenda.SessionState
import gal.tribo.librequenda.data.Me
import gal.tribo.librequenda.data.updatePreferences
import gal.tribo.librequenda.ui.components.CheckRow
import gal.tribo.librequenda.ui.components.LANGUAGES
import gal.tribo.librequenda.ui.components.LanguagePicker
import gal.tribo.librequenda.ui.components.LqSelect
import gal.tribo.librequenda.ui.components.Notice
import gal.tribo.librequenda.ui.components.PrimaryButton
import gal.tribo.librequenda.ui.components.SettingsCard
import gal.tribo.librequenda.ui.components.SettingsSection
import gal.tribo.librequenda.ui.components.SectionDivider
import gal.tribo.librequenda.ui.theme.Lq
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import gal.tribo.librequenda.ui.profile.ProfileActions
import gal.tribo.librequenda.ui.profile.Texts
import gal.tribo.librequenda.ui.profile.userMessage
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.launch

/** Preferences, from the menu: the app's language on this phone, then the web's preferences (signed in). */
@Composable
fun PreferencesContent(session: SessionState, actions: ProfileActions, modifier: Modifier = Modifier) {
    Column(modifier.fillMaxSize().background(Lq.colors.card).verticalScroll(rememberScrollState()).navigationBarsPadding()) {
        val pad = Modifier.padding(horizontal = 16.dp, vertical = 24.dp)
        Box(pad) {
            SettingsSection(stringResource(R.string.app_language), stringResource(R.string.app_language_help)) {
                LanguagePicker()
            }
        }
        session.me?.let {
            SectionDivider()
            Box(pad) { PreferencesForm(it, actions) }
        }
    }
}

/** As the web's: saved apart from the personal data. */
@Composable
private fun PreferencesForm(me: Me, actions: ProfileActions) {
    val scope = rememberCoroutineScope()
    val texts = Texts.current()
    var language by rememberSaveable(me.id) { mutableStateOf(me.preferredLanguage) }
    var emails by rememberSaveable(me.id) { mutableStateOf(me.emailNotifications) }
    var reminders by rememberSaveable(me.id) { mutableStateOf(me.emailReminders) }
    var recommendations by rememberSaveable(me.id) { mutableStateOf(me.showRecommendations) }
    var saving by remember { mutableStateOf(false) }
    var notice by remember { mutableStateOf<String?>(null) }
    var error by remember { mutableStateOf<String?>(null) }
    val savedText = stringResource(R.string.preferences_saved)

    SettingsSection(stringResource(R.string.preferences), stringResource(R.string.preferences_help)) {
        SettingsCard {
            LqSelect(
                label = stringResource(R.string.language_emails),
                options = listOf("" to stringResource(R.string.language_auto_app)) + LANGUAGES,
                selected = language,
                onSelect = { language = it },
            )
            CheckRow(stringResource(R.string.emails), emails, { emails = it }, help = stringResource(R.string.emails_help))
            CheckRow(stringResource(R.string.reminders), reminders && emails, { reminders = it }, enabled = emails)
            CheckRow(stringResource(R.string.recommendations), recommendations, { recommendations = it }, help = stringResource(R.string.recommendations_help))
            Notice(notice, error = false)
            Notice(error, error = true)
            PrimaryButton(
                stringResource(if (saving) R.string.saving else R.string.save_preferences),
                {
                    scope.launch {
                        saving = true; notice = null; error = null
                        try {
                            val saved = actions.authed { updatePreferences(it, language, emails, reminders, recommendations) }
                            actions.meUpdated(saved)
                            notice = savedText
                        } catch (c: CancellationException) { throw c } catch (e: Exception) {
                            error = e.userMessage(texts)
                        } finally { saving = false }
                    }
                },
                Modifier.fillMaxWidth(),
                enabled = !saving,
            )
        }
    }
}
