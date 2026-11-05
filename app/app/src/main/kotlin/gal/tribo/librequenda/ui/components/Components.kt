package gal.tribo.librequenda.ui.components

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import gal.tribo.librequenda.R
import gal.tribo.librequenda.ui.theme.Lq

/**
 * LibreQuenda's lockup (mark, name and "Reserva de instalacións deportivas"),
 * the same image the web's top bar and Zitadel's sign-in pages show, light or dark.
 * Decorative: whatever wraps it carries the name.
 */
@Composable
fun BrandLockup(modifier: Modifier = Modifier, height: Dp = 36.dp) {
    Image(
        painter = painterResource(if (Lq.colors.isDark) R.drawable.logo_lockup_dark else R.drawable.logo_lockup),
        contentDescription = null,
        modifier = modifier.height(height),
    )
}

/** The web's primary button: brand blue fill, 6dp corners, 44dp tall. */
@Composable
fun PrimaryButton(text: String, onClick: () -> Unit, modifier: Modifier = Modifier, enabled: Boolean = true) {
    Button(
        onClick = onClick,
        enabled = enabled,
        modifier = modifier.height(48.dp),
        shape = RoundedCornerShape(6.dp),
        contentPadding = PaddingValues(horizontal = 20.dp),
    ) { Text(text, style = MaterialTheme.typography.labelLarge) }
}

/** The web's secondary (outline) button: card fill and a hairline border. */
@Composable
fun SecondaryButton(text: String, onClick: () -> Unit, modifier: Modifier = Modifier, enabled: Boolean = true, height: Dp = 48.dp) {
    OutlinedButton(
        onClick = onClick,
        enabled = enabled,
        modifier = modifier.height(height),
        shape = RoundedCornerShape(6.dp),
        border = BorderStroke(1.dp, Lq.colors.border),
        colors = ButtonDefaults.outlinedButtonColors(containerColor = Lq.colors.card, contentColor = Lq.colors.foreground),
        contentPadding = PaddingValues(horizontal = 16.dp),
    ) { Text(text, style = MaterialTheme.typography.labelLarge) }
}

/** A status pill, as on the web's facility cards (pale tint, dark ink of the same hue). */
@Composable
fun Pill(text: String, background: Color, ink: Color, modifier: Modifier = Modifier) {
    Text(
        text,
        color = ink,
        fontSize = 13.sp,
        fontWeight = FontWeight.Medium,
        modifier = modifier
            .clip(RoundedCornerShape(50))
            .background(background)
            .padding(horizontal = 10.dp, vertical = 3.dp),
    )
}

/** «Residents only»: a facility (and its events and public games) only the council's verified residents book or join. */
@Composable
fun ResidentsOnlyPill(modifier: Modifier = Modifier) =
    Pill(androidx.compose.ui.res.stringResource(R.string.residents_only_badge), Lq.colors.pillAccentBg, Lq.colors.pillAccentInk, modifier)

/** The person's initials in a pale circle, as in the web's account menu. */
@Composable
fun InitialsAvatar(initials: String, modifier: Modifier = Modifier, size: Dp = 32.dp) {
    Box(
        modifier.size(size).clip(CircleShape).background(Lq.colors.accent),
        contentAlignment = Alignment.Center,
    ) {
        Text(initials, color = Lq.colors.accentForeground, fontSize = (size.value * 0.38f).sp, fontWeight = FontWeight.SemiBold)
    }
}
