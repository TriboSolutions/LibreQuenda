package gal.tribo.librequenda.ui.theme

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.ColorScheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Shapes
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.ReadOnlyComposable
import androidx.compose.ui.unit.dp

private fun LqColors.toMaterial(): ColorScheme {
    val base = if (isDark) darkColorScheme() else lightColorScheme()
    return base.copy(
        primary = primary,
        onPrimary = primaryForeground,
        primaryContainer = accent,
        onPrimaryContainer = accentForeground,
        secondary = primary,
        onSecondary = primaryForeground,
        secondaryContainer = accent,
        onSecondaryContainer = accentForeground,
        background = background,
        onBackground = foreground,
        surface = card,
        onSurface = foreground,
        surfaceVariant = muted,
        onSurfaceVariant = mutedForeground,
        surfaceContainerLowest = card,
        surfaceContainerLow = card,
        surfaceContainer = card,
        surfaceContainerHigh = card,
        surfaceContainerHighest = muted,
        outline = input,
        outlineVariant = border,
        error = destructive,
    )
}

// --radius: 0.5rem on the web: cards 8, controls 6
private val LqShapes = Shapes(
    extraSmall = RoundedCornerShape(4.dp),
    small = RoundedCornerShape(6.dp),
    medium = RoundedCornerShape(8.dp),
    large = RoundedCornerShape(12.dp),
)

@Composable
fun LibreQuendaTheme(darkTheme: Boolean = isSystemInDarkTheme(), content: @Composable () -> Unit) {
    val colors = if (darkTheme) DarkLqColors else LightLqColors
    CompositionLocalProvider(LocalLqColors provides colors) {
        MaterialTheme(colorScheme = colors.toMaterial(), typography = Typography, shapes = LqShapes, content = content)
    }
}

/** The web's tokens, for what Material's scheme has no slot for (pills, brand colours). */
object Lq {
    val colors: LqColors
        @Composable @ReadOnlyComposable get() = LocalLqColors.current
}

/** What reads on the logo's blue: white on light, the deep ink on the lighter blue of dark mode. */
val LqColors.onBrandBlue: androidx.compose.ui.graphics.Color
    get() = if (isDark) primaryForeground else androidx.compose.ui.graphics.Color.White
