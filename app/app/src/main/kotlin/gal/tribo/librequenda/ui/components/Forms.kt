package gal.tribo.librequenda.ui.components

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Checkbox
import androidx.compose.material3.CheckboxDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.LiveRegionMode
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.liveRegion
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import gal.tribo.librequenda.R
import gal.tribo.librequenda.ui.theme.Lq

// The web's settings building blocks (SettingsRow, cards, inputs, alerts), for /perfil and the preferences.

/** A settings section: its title and what it is for, then its card(s). */
@Composable
fun SettingsSection(title: String, description: String?, content: @Composable ColumnScope.() -> Unit) {
    Column(Modifier.fillMaxWidth()) {
        Text(title, style = MaterialTheme.typography.titleLarge, color = Lq.colors.foreground, modifier = Modifier.semantics { heading() })
        if (description != null) {
            Spacer(Modifier.height(4.dp))
            Text(description, style = MaterialTheme.typography.bodyMedium, color = Lq.colors.mutedForeground)
        }
        Spacer(Modifier.height(20.dp))
        Column(verticalArrangement = Arrangement.spacedBy(28.dp), content = content)
    }
}

/** The thin line between two sections of a page. */
@Composable
fun SectionDivider() = androidx.compose.material3.HorizontalDivider(Modifier.padding(horizontal = 16.dp), color = Lq.colors.border)

/**
 * A group of fields or actions inside a section, laid straight on the page: no
 * box around it, so the app reads as an app and not as a web page.
 */
@Composable
fun SettingsCard(title: String? = null, content: @Composable ColumnScope.() -> Unit) {
    Column(Modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(16.dp)) {
        if (title != null) Text(title, style = MaterialTheme.typography.titleMedium, color = Lq.colors.foreground)
        content()
    }
}

/**
 * A labelled text field, with an optional help line under it. With a maxLength it
 * takes no more than that, and says how many characters are used of how many.
 */
@Composable
fun LqTextField(
    label: String,
    value: String,
    onValueChange: (String) -> Unit,
    modifier: Modifier = Modifier,
    help: String? = null,
    password: Boolean = false,
    keyboardType: KeyboardType = KeyboardType.Text,
    error: String? = null,
    enabled: Boolean = true,
    minLines: Int = 1, // more: a text of several lines
    maxLength: Int? = null,
) {
    Column(modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(6.dp)) {
        Text(label, style = MaterialTheme.typography.labelLarge, color = Lq.colors.foreground)
        OutlinedTextField(
            value = value,
            onValueChange = { if (maxLength == null || it.length <= maxLength) onValueChange(it) else onValueChange(it.take(maxLength)) },
            singleLine = minLines == 1,
            minLines = minLines,
            enabled = enabled,
            isError = error != null,
            visualTransformation = if (password) PasswordVisualTransformation() else androidx.compose.ui.text.input.VisualTransformation.None,
            keyboardOptions = KeyboardOptions(keyboardType = if (password) KeyboardType.Password else keyboardType),
            shape = RoundedCornerShape(6.dp),
            textStyle = MaterialTheme.typography.bodyLarge,
            colors = OutlinedTextFieldDefaults.colors(
                unfocusedBorderColor = Lq.colors.input,
                focusedBorderColor = Lq.colors.ring,
                unfocusedContainerColor = Lq.colors.card,
                focusedContainerColor = Lq.colors.card,
            ),
            modifier = Modifier.fillMaxWidth(),
        )
        if (error != null) Text(error, style = MaterialTheme.typography.bodySmall, color = Lq.colors.destructive)
        else if (help != null) Text(help, style = MaterialTheme.typography.bodySmall, color = Lq.colors.mutedForeground)
        if (maxLength != null) Text(
            stringResource(R.string.text_count, value.length.toString(), maxLength.toString()),
            style = MaterialTheme.typography.bodySmall,
            color = if (value.length >= maxLength) Lq.colors.foreground else Lq.colors.mutedForeground,
            modifier = Modifier.align(Alignment.End),
        )
    }
}

/** A read-only field that opens something (a date picker) when tapped. */
@Composable
fun LqPickerField(label: String, value: String, placeholder: String, onClick: () -> Unit, help: String? = null) {
    Column(Modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(6.dp)) {
        Text(label, style = MaterialTheme.typography.labelLarge, color = Lq.colors.foreground)
        Box(
            Modifier
                .fillMaxWidth()
                .height(52.dp)
                .clip(RoundedCornerShape(6.dp))
                .border(1.dp, Lq.colors.input, RoundedCornerShape(6.dp))
                .background(Lq.colors.card)
                .clickable(onClick = onClick)
                .padding(horizontal = 16.dp),
            contentAlignment = Alignment.CenterStart,
        ) {
            Text(
                value.ifBlank { placeholder },
                style = MaterialTheme.typography.bodyLarge,
                color = if (value.isBlank()) Lq.colors.mutedForeground else Lq.colors.foreground,
            )
        }
        if (help != null) Text(help, style = MaterialTheme.typography.bodySmall, color = Lq.colors.mutedForeground)
    }
}

/** A choice from a short list: the field shows it, a sheet from the bottom changes it (no dropdowns). */
@Composable
fun <T> LqSelect(label: String, options: List<Pair<T, String>>, selected: T, onSelect: (T) -> Unit) {
    var open by remember { mutableStateOf(false) }
    Column(Modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(6.dp)) {
        Text(label, style = MaterialTheme.typography.labelLarge, color = Lq.colors.foreground)
        Row(
            Modifier
                .fillMaxWidth()
                .height(52.dp)
                .clip(RoundedCornerShape(6.dp))
                .border(1.dp, Lq.colors.input, RoundedCornerShape(6.dp))
                .background(Lq.colors.card)
                .clickable { open = true }
                .padding(horizontal = 16.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(options.firstOrNull { it.first == selected }?.second.orEmpty(), style = MaterialTheme.typography.bodyLarge, color = Lq.colors.foreground, modifier = Modifier.weight(1f))
            Icon(painterResource(R.drawable.ic_chevron_down), contentDescription = null, tint = Lq.colors.mutedForeground, modifier = Modifier.size(18.dp))
        }
    }
    if (open) SingleChoiceSheet(label, options, selected, onSelect, onDismiss = { open = false })
}

/** A checkbox with its label (and help), the whole row tappable. */
@Composable
fun CheckRow(label: String, checked: Boolean, onChange: (Boolean) -> Unit, help: String? = null, enabled: Boolean = true) {
    Row(
        Modifier.fillMaxWidth().clip(RoundedCornerShape(6.dp)).clickable(enabled = enabled) { onChange(!checked) },
        verticalAlignment = Alignment.Top,
    ) {
        Checkbox(
            checked = checked,
            onCheckedChange = null,
            enabled = enabled,
            colors = CheckboxDefaults.colors(checkedColor = Lq.colors.primary, uncheckedColor = Lq.colors.input),
            modifier = Modifier.padding(top = 2.dp, end = 10.dp).size(20.dp),
        )
        Column(verticalArrangement = Arrangement.spacedBy(2.dp)) {
            Text(label, style = MaterialTheme.typography.bodyLarge, color = if (enabled) Lq.colors.foreground else Lq.colors.mutedForeground)
            if (help != null) Text(help, style = MaterialTheme.typography.bodySmall, color = Lq.colors.mutedForeground)
        }
    }
}

/** The web's alert: green when something was saved, red when it failed. Read out when it appears. */
@Composable
fun Notice(text: String?, error: Boolean) {
    if (text == null) return
    Text(
        text,
        style = MaterialTheme.typography.bodyMedium,
        color = if (error) Lq.colors.destructive else Lq.colors.pillPositiveInk,
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(6.dp))
            .background(if (error) Lq.colors.pillNegativeBg else Lq.colors.pillPositiveBg)
            .padding(12.dp)
            .semantics { liveRegion = LiveRegionMode.Polite },
    )
}

/** The web's destructive button. */
@Composable
fun DangerButton(text: String, onClick: () -> Unit, modifier: Modifier = Modifier, enabled: Boolean = true) {
    androidx.compose.material3.Button(
        onClick = onClick,
        enabled = enabled,
        modifier = modifier.height(48.dp),
        shape = RoundedCornerShape(6.dp),
        colors = ButtonDefaults.buttonColors(containerColor = Lq.colors.destructive, contentColor = Lq.colors.card),
    ) { Text(text, style = MaterialTheme.typography.labelLarge) }
}

/** The web's ghost button: text only. */
@Composable
fun GhostButton(text: String, onClick: () -> Unit, modifier: Modifier = Modifier, enabled: Boolean = true) {
    androidx.compose.material3.TextButton(onClick = onClick, enabled = enabled, modifier = modifier, shape = RoundedCornerShape(6.dp)) {
        Text(text, style = MaterialTheme.typography.labelLarge, color = Lq.colors.primary)
    }
}
