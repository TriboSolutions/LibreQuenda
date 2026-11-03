package gal.tribo.librequenda.ui.components

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.selection.selectable
import androidx.compose.foundation.selection.selectableGroup
import androidx.compose.foundation.selection.toggleable
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Text
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.launch
import gal.tribo.librequenda.R
import gal.tribo.librequenda.ui.theme.Lq

// The app never uses dropdowns: every choice comes up from the bottom, in one of these.

/** A sheet from the bottom with a title, for any choice. */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LqSheet(title: String, onDismiss: () -> Unit, content: @Composable () -> Unit) {
    ModalBottomSheet(
        onDismissRequest = onDismiss,
        sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true),
        containerColor = Lq.colors.card,
    ) {
        Column(Modifier.fillMaxWidth().verticalScroll(rememberScrollState()).navigationBarsPadding().padding(horizontal = 16.dp).padding(bottom = 16.dp)) {
            Text(title, style = MaterialTheme.typography.headlineSmall, color = Lq.colors.foreground, modifier = Modifier.semantics { heading() })
            Spacer(Modifier.height(12.dp))
            content()
        }
    }
}

/** One of several: tapping an option picks it and closes the sheet. */
@Composable
fun <T> SingleChoiceSheet(title: String, options: List<Pair<T, String>>, selected: T, onSelect: (T) -> Unit, onDismiss: () -> Unit) {
    LqSheet(title, onDismiss) {
        Column(Modifier.selectableGroup()) {
            options.forEachIndexed { i, (value, label) ->
                if (i > 0) HorizontalDivider(color = Lq.colors.border)
                Row(
                    Modifier.fillMaxWidth()
                        .selectable(selected = value == selected, role = Role.RadioButton) { onSelect(value); onDismiss() }
                        .padding(vertical = 16.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text(label, style = MaterialTheme.typography.bodyLarge, color = Lq.colors.foreground, modifier = Modifier.weight(1f))
                    Radio(value == selected)
                }
            }
        }
    }
}

/** Any of several: ticked here, applied with the button. */
@Composable
fun MultiChoiceSheet(title: String, options: List<Pair<String, String>>, selected: Set<String>, onApply: (Set<String>) -> Unit, onDismiss: () -> Unit, applyLabel: String) {
    var chosen by remember { mutableStateOf(selected) }
    LqSheet(title, onDismiss) {
        options.forEachIndexed { i, (value, label) ->
            if (i > 0) HorizontalDivider(color = Lq.colors.border)
            Row(
                Modifier.fillMaxWidth()
                    .toggleable(value = value in chosen, role = Role.Checkbox) { chosen = if (it) chosen + value else chosen - value }
                    .padding(vertical = 16.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(label, style = MaterialTheme.typography.bodyLarge, color = Lq.colors.foreground, modifier = Modifier.weight(1f))
                Tick(value in chosen)
            }
        }
        Spacer(Modifier.height(16.dp))
        PrimaryButton(applyLabel, { onApply(chosen); onDismiss() }, Modifier.fillMaxWidth())
    }
}

@Composable
fun Radio(selected: Boolean) {
    Box(
        Modifier.size(22.dp).clip(CircleShape).border(2.dp, if (selected) Lq.colors.primary else Lq.colors.input, CircleShape),
        contentAlignment = Alignment.Center,
    ) {
        if (selected) Box(Modifier.size(11.dp).clip(CircleShape).background(Lq.colors.primary))
    }
}

@Composable
private fun Tick(checked: Boolean) {
    Box(
        Modifier.size(22.dp).clip(RoundedCornerShape(6.dp))
            .background(if (checked) Lq.colors.primary else Lq.colors.card)
            .border(2.dp, if (checked) Lq.colors.primary else Lq.colors.input, RoundedCornerShape(6.dp)),
        contentAlignment = Alignment.Center,
    ) {
        if (checked) Icon(painterResource(R.drawable.ic_check), contentDescription = null, tint = Lq.colors.primaryForeground, modifier = Modifier.size(16.dp))
    }
}

/**
 * A day from a month calendar, in a sheet: tapping a day picks it and closes it.
 * Only [from]..[until] can be chosen (the days one can book).
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DateSheet(title: String, selected: java.time.LocalDate, from: java.time.LocalDate, until: java.time.LocalDate, onSelect: (java.time.LocalDate) -> Unit, onDismiss: () -> Unit) {
    fun millis(d: java.time.LocalDate) = d.atStartOfDay(java.time.ZoneOffset.UTC).toInstant().toEpochMilli()
    fun date(m: Long) = java.time.Instant.ofEpochMilli(m).atZone(java.time.ZoneOffset.UTC).toLocalDate()
    val state = androidx.compose.material3.rememberDatePickerState(
        initialSelectedDateMillis = millis(selected),
        yearRange = from.year..until.year,
        selectableDates = object : androidx.compose.material3.SelectableDates {
            override fun isSelectableDate(utcTimeMillis: Long) = date(utcTimeMillis).let { !it.isBefore(from) && !it.isAfter(until) }
        },
    )
    androidx.compose.runtime.LaunchedEffect(state.selectedDateMillis) {
        state.selectedDateMillis?.let(::date)?.takeIf { it != selected }?.let { onSelect(it); onDismiss() }
    }
    LqSheet(title, onDismiss) {
        androidx.compose.material3.DatePicker(
            state = state,
            title = null,
            headline = null,
            showModeToggle = false,
            colors = androidx.compose.material3.DatePickerDefaults.colors(
                containerColor = Lq.colors.card,
                selectedDayContainerColor = Lq.colors.primary,
                selectedDayContentColor = Lq.colors.primaryForeground,
                todayDateBorderColor = Lq.colors.primary,
                todayContentColor = Lq.colors.primary,
            ),
        )
    }
}

/**
 * «Are you sure?» from the bottom (the app has no dialogs): what happens, anything
 * else it needs ([content]), then the action (red when [danger]) and «keep» under it.
 * The sheet shows the action's error and stays open; done, it closes itself.
 */
@Composable
fun ConfirmSheet(
    title: String,
    text: String,
    confirm: String,
    confirming: String,
    keep: String,
    onDismiss: () -> Unit,
    onConfirm: suspend () -> Unit,
    errorOf: (Throwable) -> String,
    danger: Boolean = true,
    enabled: Boolean = true,
    content: @Composable () -> Unit = {},
) {
    val scope = androidx.compose.runtime.rememberCoroutineScope()
    var busy by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    LqSheet(title, onDismiss = { if (!busy) onDismiss() }) {
        Text(text, style = MaterialTheme.typography.bodyLarge, color = Lq.colors.foreground)
        Spacer(Modifier.height(12.dp))
        content()
        Notice(error, error = true)
        Spacer(Modifier.height(20.dp))
        val run: () -> Unit = {
            scope.launch {
                busy = true; error = null
                try { onConfirm() } catch (c: kotlinx.coroutines.CancellationException) { throw c } catch (e: Exception) { error = errorOf(e) }
                busy = false
            }
        }
        if (danger) DangerButton(if (busy) confirming else confirm, run, Modifier.fillMaxWidth(), enabled = enabled && !busy)
        else PrimaryButton(if (busy) confirming else confirm, run, Modifier.fillMaxWidth(), enabled = enabled && !busy)
        Spacer(Modifier.height(10.dp))
        SecondaryButton(keep, onDismiss, Modifier.fillMaxWidth(), enabled = !busy)
    }
}
