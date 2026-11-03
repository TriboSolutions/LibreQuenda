package gal.tribo.librequenda.ui.legal

import android.content.Context
import android.content.Intent
import android.net.Uri
import androidx.annotation.DrawableRes
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import gal.tribo.librequenda.R
import gal.tribo.librequenda.data.Branding
import gal.tribo.librequenda.ui.theme.Lq
import org.json.JSONObject

/** The terms of use or the privacy policy: the web's legal.json, copied into the app by app/tools/web_strings.py. */
enum class LegalText(val key: String) { Terms("terms"), Privacy("privacy") }

private data class LegalSection(val title: String, val body: List<String>)
private data class LegalDoc(val intro: String, val sections: List<LegalSection>)

/** Reads the text in the app's language (Galician when it has no copy), with the installation's name for {{app}}. */
private fun loadLegal(context: Context, which: LegalText, language: String, app: String): LegalDoc {
    val lang = language.takeIf { it in setOf("gl", "es", "ca", "en") } ?: "gl"
    val text = context.assets.open("legal/$lang.json").bufferedReader().use { it.readText() }
    val doc = JSONObject(text).getJSONObject(which.key)
    fun fill(s: String) = s.replace("{{app}}", app)
    val sections = doc.getJSONArray("sections")
    return LegalDoc(
        intro = fill(doc.getString("intro")),
        sections = List(sections.length()) { i ->
            val s = sections.getJSONObject(i)
            val body = s.getJSONArray("body")
            LegalSection(fill(s.getString("title")), List(body.length()) { fill(body.getString(it)) })
        },
    )
}

/** The page's body under its title: when it was updated, the introduction and each section. */
@Composable
fun LegalContent(which: LegalText, branding: Branding) {
    val context = LocalContext.current
    val language = LocalConfiguration.current.locales[0].language
    val doc = remember(which, language, branding.name) { loadLegal(context, which, language, branding.name) }
    Column(
        Modifier.fillMaxWidth().verticalScroll(rememberScrollState()).navigationBarsPadding().padding(start = 16.dp, end = 16.dp, top = 8.dp, bottom = 40.dp),
        verticalArrangement = Arrangement.spacedBy(20.dp),
    ) {
        Text(stringResource(R.string.legal_updated), style = MaterialTheme.typography.bodySmall, color = Lq.colors.mutedForeground)
        Text(doc.intro, style = MaterialTheme.typography.bodyLarge, color = Lq.colors.foreground)
        doc.sections.forEach { section ->
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text(section.title, style = MaterialTheme.typography.titleMedium, color = Lq.colors.foreground, modifier = Modifier.semantics { heading() })
                section.body.forEach { Text(it, style = MaterialTheme.typography.bodyLarge, color = Lq.colors.foreground) }
            }
        }
        if (branding.hasSupport) {
            Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                Text(stringResource(R.string.support_title), style = MaterialTheme.typography.titleMedium, color = Lq.colors.foreground, modifier = Modifier.semantics { heading() })
                SupportRows(branding)
            }
        }
    }
}

/** Help and support: who to write to or call when something in the app is unclear (this installation's own). */
@Composable
fun SupportContent(branding: Branding) {
    Column(
        Modifier.fillMaxWidth().verticalScroll(rememberScrollState()).navigationBarsPadding().padding(start = 16.dp, end = 16.dp, top = 8.dp, bottom = 40.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        Text(stringResource(R.string.support_help), style = MaterialTheme.typography.bodyLarge, color = Lq.colors.foreground)
        SupportRows(branding)
    }
}

/** The email (opens the mail app) and the phone (opens the dialler), each when set. */
@Composable
private fun SupportRows(branding: Branding) {
    val context = LocalContext.current
    Column {
        if (branding.supportEmail.isNotBlank()) {
            ContactRow(R.drawable.ic_mail, stringResource(R.string.support_email), branding.supportEmail) {
                runCatching { context.startActivity(Intent(Intent.ACTION_SENDTO, Uri.parse("mailto:" + branding.supportEmail))) }
            }
        }
        if (branding.supportEmail.isNotBlank() && branding.supportPhone.isNotBlank()) HorizontalDivider(Modifier.padding(start = 42.dp), color = Lq.colors.border)
        if (branding.supportPhone.isNotBlank()) {
            ContactRow(R.drawable.ic_phone, stringResource(R.string.support_phone), branding.supportPhone) {
                runCatching { context.startActivity(Intent(Intent.ACTION_DIAL, Uri.parse("tel:" + branding.supportPhone.filterNot(Char::isWhitespace)))) }
            }
        }
    }
}

@Composable
private fun ContactRow(@DrawableRes icon: Int, action: String, value: String, onClick: () -> Unit) {
    Row(Modifier.fillMaxWidth().clickable(onClick = onClick).padding(vertical = 14.dp), verticalAlignment = Alignment.CenterVertically) {
        Icon(painterResource(icon), contentDescription = null, tint = Lq.colors.primary, modifier = Modifier.size(24.dp))
        Spacer(Modifier.size(18.dp))
        Column(Modifier.weight(1f)) {
            Text(value, style = MaterialTheme.typography.bodyLarge, color = Lq.colors.foreground)
            Text(action, style = MaterialTheme.typography.bodySmall, color = Lq.colors.mutedForeground)
        }
        Icon(painterResource(R.drawable.ic_chevron_right), contentDescription = null, tint = Lq.colors.mutedForeground, modifier = Modifier.size(20.dp))
    }
}
