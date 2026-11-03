package gal.tribo.librequenda.ui.components

import android.app.LocaleManager
import android.os.Build
import android.os.LocaleList
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.selection.selectable
import androidx.compose.foundation.selection.selectableGroup
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.unit.dp
import androidx.compose.foundation.border
import gal.tribo.librequenda.R
import gal.tribo.librequenda.ui.theme.Lq

/** The app's languages, as the web's; each named in itself. */
val LANGUAGES = listOf("gl" to "Galego", "es" to "Español", "ca" to "Català", "en" to "English")

/**
 * The app's language (Android 13+ keeps it per app, as Settings › Apps › Language does);
 * "the phone's" follows the system. Older Android versions use the phone's language.
 */
@Composable
fun LanguagePicker(modifier: Modifier = Modifier) {
    if (Build.VERSION.SDK_INT < Build.VERSION_CODES.TIRAMISU) return
    val context = LocalContext.current
    val manager = remember { context.getSystemService(LocaleManager::class.java) }
    var current by remember { mutableStateOf(manager.applicationLocales.takeIf { !it.isEmpty }?.get(0)?.language) }

    fun choose(tag: String?) {
        current = tag
        // the activity is recreated in the new language
        manager.applicationLocales = if (tag == null) LocaleList.getEmptyLocaleList() else LocaleList.forLanguageTags(tag)
    }

    Column(modifier) {
        Column(Modifier.fillMaxWidth().selectableGroup()) {
            val options = listOf(null to stringResource(R.string.language_system)) + LANGUAGES
            options.forEachIndexed { index, (tag, name) ->
                if (index > 0) HorizontalDivider(color = Lq.colors.border)
                Row(
                    Modifier
                        .fillMaxWidth()
                        .selectable(selected = current == tag, role = Role.RadioButton) { choose(tag) }
                        .padding(vertical = 16.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text(name, style = MaterialTheme.typography.bodyLarge, color = Lq.colors.foreground, modifier = Modifier.weight(1f))
                    Radio(current == tag)
                }
            }
        }
    }
}

/**
 * The language, from the very first screen: the current one with a chevron,
 * opening the list. Android 13+ only (older versions follow the phone).
 */
@Composable
fun LanguageMenuButton(modifier: Modifier = Modifier) {
    if (Build.VERSION.SDK_INT < Build.VERSION_CODES.TIRAMISU) return
    val context = LocalContext.current
    val manager = remember { context.getSystemService(LocaleManager::class.java) }
    val current = androidx.compose.ui.platform.LocalConfiguration.current.locales[0].language
    var open by remember { mutableStateOf(false) }
    androidx.compose.foundation.layout.Box(modifier) {
        Row(
            Modifier
                .clip(RoundedCornerShape(20.dp))
                .clickable { open = true }
                .padding(horizontal = 12.dp, vertical = 8.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(LANGUAGES.firstOrNull { it.first == current }?.second ?: current, style = MaterialTheme.typography.labelLarge, color = Lq.colors.foreground)
            Spacer(Modifier.size(6.dp))
            androidx.compose.material3.Icon(
                androidx.compose.ui.res.painterResource(R.drawable.ic_chevron_down), contentDescription = stringResource(R.string.app_language),
                tint = Lq.colors.foreground, modifier = Modifier.size(16.dp),
            )
        }
    }
    if (open) {
        SingleChoiceSheet(stringResource(R.string.app_language), LANGUAGES, current, onSelect = {
            manager.applicationLocales = LocaleList.forLanguageTags(it)
        }, onDismiss = { open = false })
    }
}
