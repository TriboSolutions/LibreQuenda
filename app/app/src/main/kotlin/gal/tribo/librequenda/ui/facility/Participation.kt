package gal.tribo.librequenda.ui.facility

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.pluralStringResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import gal.tribo.librequenda.R
import gal.tribo.librequenda.data.CouncilNotice
import gal.tribo.librequenda.data.IssuesSummary
import gal.tribo.librequenda.ui.theme.Lq
import java.text.NumberFormat
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter

/** The council's current announcements, newest first: a tinted block each, as on the web. */
@Composable
internal fun FacilityNotices(notices: List<CouncilNotice>, modifier: Modifier = Modifier) {
    if (notices.isEmpty()) return
    val label = stringResource(R.string.notices_label)
    Column(modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(8.dp)) {
        notices.forEach { notice ->
            Row(
                Modifier.fillMaxWidth().clip(RoundedCornerShape(12.dp)).background(Lq.colors.pillAccentBg).padding(14.dp)
                    .semantics(mergeDescendants = true) {},
                verticalAlignment = Alignment.Top,
            ) {
                Icon(painterResource(R.drawable.ic_megaphone), contentDescription = label, tint = Lq.colors.pillAccentInk, modifier = Modifier.size(20.dp))
                Spacer(Modifier.size(12.dp))
                Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                    Text(notice.title, style = MaterialTheme.typography.titleSmall, color = Lq.colors.pillAccentInk)
                    Text(notice.body, style = MaterialTheme.typography.bodyMedium, color = Lq.colors.pillAccentInk)
                    Text(
                        stringResource(R.string.notices_published, longDay(notice.createdAt)) + " · " +
                            stringResource(R.string.notices_until, longDay(notice.expiresAt.minusMillis(1))),
                        style = MaterialTheme.typography.bodySmall, color = Lq.colors.pillAccentInk,
                    )
                }
            }
        }
    }
}

/**
 * The facility's status: what is known to be broken (nobody needs to report it again),
 * what was fixed lately and how long the council takes. Categories and dates only.
 */
@Composable
internal fun FacilityState(issues: IssuesSummary, councilName: String) {
    Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
        if (issues.open.isEmpty()) {
            Line(R.drawable.ic_circle_check, stringResource(R.string.state_no_known), null, Lq.colors.pillPositiveInk)
        } else {
            Text(stringResource(R.string.state_known), style = MaterialTheme.typography.labelLarge, color = Lq.colors.foreground)
            issues.open.forEach { issue ->
                val detail = buildString {
                    append(stringResource(if (issue.inProgress) R.string.state_in_progress else R.string.state_reported))
                    append(" · ").append(stringResource(R.string.state_since, longDay(issue.firstReportedAt)))
                    if (issue.reports > 1) append(" · ").append(pluralStringResource(R.plurals.state_reports, issue.reports, issue.reports))
                }
                Line(R.drawable.ic_wrench, category(issue.category), detail, Lq.colors.mutedForeground)
            }
            Text(stringResource(R.string.state_already_known), style = MaterialTheme.typography.bodySmall, color = Lq.colors.mutedForeground)
        }
        if (issues.recentlySolved.isNotEmpty()) {
            Text(stringResource(R.string.state_solved), style = MaterialTheme.typography.labelLarge, color = Lq.colors.foreground, modifier = Modifier.padding(top = 4.dp))
            issues.recentlySolved.forEach { issue ->
                Line(R.drawable.ic_circle_check, category(issue.category), stringResource(R.string.state_solved_on, longDay(issue.solvedAt)), Lq.colors.pillPositiveInk)
            }
        }
        issues.averageDays?.let { days ->
            val locale = LocalConfiguration.current.locales[0]
            val formatted = NumberFormat.getNumberInstance(locale).apply { maximumFractionDigits = 1 }.format(days)
            Text(
                pluralStringResource(R.plurals.state_average, issues.solvedCount, councilName, formatted, issues.solvedCount),
                style = MaterialTheme.typography.bodyMedium, color = Lq.colors.foreground,
                modifier = Modifier.padding(top = 4.dp).fillMaxWidth().clip(RoundedCornerShape(12.dp)).background(Lq.colors.muted).padding(12.dp),
            )
        }
    }
}

@Composable
private fun Line(icon: Int, title: String, detail: String?, tint: androidx.compose.ui.graphics.Color) {
    Row(Modifier.fillMaxWidth().semantics(mergeDescendants = true) {}, verticalAlignment = Alignment.Top) {
        Icon(painterResource(icon), contentDescription = null, tint = tint, modifier = Modifier.padding(top = 2.dp).size(16.dp))
        Spacer(Modifier.size(8.dp))
        Column(Modifier.weight(1f)) {
            Text(title, style = MaterialTheme.typography.bodyMedium, color = Lq.colors.foreground)
            if (detail != null) Text(detail, style = MaterialTheme.typography.bodySmall, color = Lq.colors.mutedForeground)
        }
    }
}

@Composable
private fun category(key: String): String = stringResource(
    when (key) {
        "equipment" -> R.string.issue_cat_equipment
        "lighting" -> R.string.issue_cat_lighting
        "cleaning" -> R.string.issue_cat_cleaning
        "changing_rooms" -> R.string.issue_cat_changing_rooms
        "accessibility" -> R.string.issue_cat_accessibility
        "safety" -> R.string.issue_cat_safety
        else -> R.string.issue_cat_other
    },
)

@Composable
private fun longDay(at: Instant): String =
    at.atZone(ZONE).format(DateTimeFormatter.ofPattern("d MMMM", LocalConfiguration.current.locales[0]))
private val ZONE = ZoneId.of("Europe/Madrid")
