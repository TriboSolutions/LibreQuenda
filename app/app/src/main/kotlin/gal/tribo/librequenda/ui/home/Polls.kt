package gal.tribo.librequenda.ui.home

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.selection.selectable
import androidx.compose.foundation.selection.selectableGroup
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.RadioButton
import androidx.compose.material3.RadioButtonDefaults
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
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.pluralStringResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import gal.tribo.librequenda.R
import gal.tribo.librequenda.data.Poll
import gal.tribo.librequenda.ui.components.Notice
import gal.tribo.librequenda.ui.components.PrimaryButton
import gal.tribo.librequenda.ui.theme.Lq
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.launch
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter

private val ZONE = ZoneId.of("Europe/Madrid")

/**
 * The polls page: the councils' polls for the person, the open ones first, one
 * after another (no cards). One vote each, which can change while open; the
 * results once it closes. focus: the poll a notice named, scrolled to and
 * tinted. emailVerified: voting needs it, as booking does.
 */
@Composable
internal fun PollsContent(
    polls: List<Poll>,
    emailVerified: Boolean,
    focus: String,
    onVote: suspend (poll: String, option: String) -> Unit,
    hasMore: Boolean = false,
    onMore: () -> Unit = {},
) {
    val list = rememberLazyListState()
    // the poll a notice named: scrolled to once it is there (item 0 is the help)
    var scrolled by remember(focus) { mutableStateOf(false) }
    LaunchedEffect(focus, polls) {
        val index = polls.indexOfFirst { it.id == focus }
        if (!scrolled && index >= 0) { list.animateScrollToItem(index + 1); scrolled = true }
    }
    LazyColumn(state = list, modifier = Modifier.fillMaxSize().navigationBarsPadding(), contentPadding = PaddingValues(top = 8.dp, bottom = 32.dp)) {
        item {
            Text(stringResource(R.string.polls_help), style = MaterialTheme.typography.bodyMedium, color = Lq.colors.mutedForeground, modifier = Modifier.padding(horizontal = 16.dp).padding(bottom = 16.dp))
            if (polls.isEmpty()) {
                Text(stringResource(R.string.polls_empty), style = MaterialTheme.typography.bodyMedium, color = Lq.colors.mutedForeground, modifier = Modifier.padding(horizontal = 16.dp))
            }
        }
        itemsIndexed(polls, key = { _, poll -> poll.id }) { i, poll ->
            // a page at a time: the next one as the end comes into view, as the facilities' search
            if (hasMore && i >= polls.size - 3) LaunchedEffect(polls.size) { onMore() }
            if (i > 0) HorizontalDivider(Modifier.padding(horizontal = 16.dp, vertical = 12.dp), color = Lq.colors.border)
            val focused = poll.id == focus
            Box(Modifier.fillMaxWidth().then(if (focused) Modifier.background(Lq.colors.pillAccentBg).padding(vertical = 12.dp) else Modifier)) {
                PollItem(poll, emailVerified, onVote)
            }
        }
        if (hasMore) item { LinearProgressIndicator(Modifier.fillMaxWidth().padding(16.dp)) }
    }
}

@Composable
private fun PollItem(poll: Poll, emailVerified: Boolean, onVote: suspend (String, String) -> Unit) {
    val scope = rememberCoroutineScope()
    var choice by remember(poll.id, poll.myOption) { mutableStateOf(poll.myOption) }
    var busy by remember { mutableStateOf(false) }
    var saved by remember(poll.id) { mutableStateOf(false) }
    var error by remember(poll.id) { mutableStateOf<String?>(null) }
    Column(Modifier.padding(horizontal = 16.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
        Row(verticalAlignment = Alignment.Top) {
            Icon(painterResource(R.drawable.ic_vote), contentDescription = null, tint = Lq.colors.mutedForeground, modifier = Modifier.padding(top = 2.dp).size(20.dp))
            Spacer(Modifier.size(10.dp))
            Column(Modifier.weight(1f)) {
                Text(poll.question, style = MaterialTheme.typography.titleMedium, color = Lq.colors.foreground)
                Text(listOf(poll.councilName, poll.facilityName).filter { it.isNotBlank() }.joinToString(" · "), style = MaterialTheme.typography.bodySmall, color = Lq.colors.mutedForeground)
                Text(
                    buildString {
                        append(stringResource(if (poll.open) R.string.polls_closes else R.string.polls_closed, dateTime(poll.closesAt)))
                        if (poll.residentsOnly) append(" · ").append(stringResource(R.string.polls_residents_only))
                    },
                    style = MaterialTheme.typography.bodySmall, color = Lq.colors.mutedForeground,
                )
            }
        }
        if (poll.open) {
            Column(Modifier.selectableGroup()) {
                poll.options.forEach { option ->
                    Row(
                        Modifier.fillMaxWidth().heightIn(min = 48.dp)
                            .selectable(selected = choice == option.id, role = Role.RadioButton, onClick = { choice = option.id; saved = false }),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        RadioButton(selected = choice == option.id, onClick = null, colors = RadioButtonDefaults.colors(selectedColor = Lq.colors.primary))
                        Spacer(Modifier.size(12.dp))
                        Text(option.label, style = MaterialTheme.typography.bodyLarge, color = Lq.colors.foreground)
                    }
                }
            }
            if (saved) Notice(stringResource(R.string.polls_saved), error = false)
            error?.let { Notice(it, error = true) }
            if (!emailVerified) {
                Text(stringResource(R.string.polls_verify_email), style = MaterialTheme.typography.bodyMedium, color = Lq.colors.mutedForeground)
            } else {
                PrimaryButton(
                    stringResource(if (poll.myOption.isNotEmpty()) R.string.polls_change else R.string.polls_vote),
                    {
                        busy = true; error = null
                        scope.launch {
                            try { onVote(poll.id, choice); saved = true }
                            catch (c: CancellationException) { throw c }
                            catch (e: Exception) { error = e.message }
                            finally { busy = false }
                        }
                    },
                    Modifier.fillMaxWidth(),
                    enabled = !busy && choice.isNotEmpty() && choice != poll.myOption,
                )
            }
            if (poll.myOption.isNotEmpty()) Text(stringResource(R.string.polls_results_later), style = MaterialTheme.typography.bodySmall, color = Lq.colors.mutedForeground)
        } else {
            val total = poll.votes ?: 0
            val top = poll.options.maxOfOrNull { it.votes ?: 0 } ?: 0
            poll.options.forEach { option ->
                val votes = option.votes ?: 0
                val share = if (total > 0) votes.toFloat() / total else 0f
                Column(Modifier.fillMaxWidth().padding(vertical = 2.dp).semantics(mergeDescendants = true) {}, verticalArrangement = Arrangement.spacedBy(4.dp)) {
                    Row(verticalAlignment = Alignment.Bottom) {
                        Text(
                            option.label + if (poll.myOption == option.id) " (${stringResource(R.string.polls_yours)})" else "",
                            style = MaterialTheme.typography.bodyMedium, color = Lq.colors.foreground,
                            fontWeight = if (votes == top && total > 0) FontWeight.SemiBold else null,
                            modifier = Modifier.weight(1f),
                        )
                        Text("${Math.round(share * 100)}% · " + pluralStringResource(R.plurals.polls_votes, votes, votes), style = MaterialTheme.typography.bodySmall, color = Lq.colors.mutedForeground)
                    }
                    LinearProgressIndicator(progress = { share }, modifier = Modifier.fillMaxWidth().height(6.dp), color = Lq.colors.primary, trackColor = Lq.colors.muted, drawStopIndicator = {})
                }
            }
            Text(pluralStringResource(R.plurals.polls_total, total, total), style = MaterialTheme.typography.bodySmall, color = Lq.colors.mutedForeground)
        }
    }
}

@Composable
private fun dateTime(at: Instant): String =
    at.atZone(ZONE).format(DateTimeFormatter.ofPattern("EEE d MMM, HH:mm", LocalConfiguration.current.locales[0]))
