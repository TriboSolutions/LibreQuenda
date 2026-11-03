package gal.tribo.librequenda.ui.theme

import androidx.compose.runtime.Immutable
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.graphics.Color

// The web's tokens (web/src/globals.css), light and dark, so the app and the web look alike,
// except the primary colour: in the app it is the logo's blue (docs/brand), everywhere.

@Immutable
data class LqColors(
    val background: Color,
    val foreground: Color,
    val card: Color,
    val primary: Color, // the logo's blue: buttons, chips, links, selections
    val primaryDeep: Color, // the deeper blue of a selected main action (the flash booking tab)
    val primaryForeground: Color,
    val muted: Color,
    val mutedForeground: Color,
    val accent: Color,
    val accentForeground: Color,
    val destructive: Color,
    val border: Color,
    val input: Color,
    val ring: Color,
    val brandInk: Color,
    val brandBlue: Color,
    val pillPositiveBg: Color,
    val pillPositiveInk: Color,
    val pillNegativeBg: Color,
    val pillNegativeInk: Color,
    val pillNeutralBg: Color,
    val pillAccentBg: Color,
    val pillAccentInk: Color,
    val pillNeutralInk: Color,
    val isDark: Boolean,
)

val LightLqColors = LqColors(
    background = Color(0xFFF6F9FB),
    foreground = Color(0xFF111827),
    card = Color(0xFFFFFFFF),
    primary = Color(0xFF0098D4),
    primaryDeep = Color(0xFF074F80),
    primaryForeground = Color(0xFFFFFFFF),
    muted = Color(0xFFF1F4F7),
    mutedForeground = Color(0xFF434C59),
    accent = Color(0xFFEEF6FB),
    accentForeground = Color(0xFF0B3D61),
    destructive = Color(0xFF9F1C21),
    border = Color(0xFFE3E8ED),
    input = Color(0xFF758290),
    ring = Color(0xFF0A6FB0),
    brandInk = Color(0xFF0B2545),
    brandBlue = Color(0xFF0098D4),
    pillPositiveBg = Color(0xFFDCFCE7),
    pillPositiveInk = Color(0xFF14532D),
    pillNegativeBg = Color(0xFFFFEDEA),
    pillNegativeInk = Color(0xFF831C1E),
    pillNeutralBg = Color(0xFFF1F4F7),
    pillNeutralInk = Color(0xFF374151),
    pillAccentBg = Color(0xFFEEF6FB),
    pillAccentInk = Color(0xFF0B3D61),
    isDark = false,
)

val DarkLqColors = LqColors(
    background = Color(0xFF0E0F10),
    foreground = Color(0xFFF3F4F6),
    card = Color(0xFF17181A),
    primary = Color(0xFF33B4EB),
    primaryDeep = Color(0xFF8FCAF2),
    primaryForeground = Color(0xFF06253A),
    muted = Color(0xFF212226),
    mutedForeground = Color(0xFFC4C8CE),
    accent = Color(0xFF10283A),
    accentForeground = Color(0xFFB5DAF3),
    destructive = Color(0xFFFFA49E),
    border = Color(0xFF2A2C30),
    input = Color(0xFF6B7078),
    ring = Color(0xFF5FB0E6),
    brandInk = Color(0xFFFFFFFF),
    brandBlue = Color(0xFF33B4EB),
    pillPositiveBg = Color(0xFF12301F),
    pillPositiveInk = Color(0xFF74D6A0),
    pillNegativeBg = Color(0xFF3A1A1C),
    pillNegativeInk = Color(0xFFFFABA6),
    pillNeutralBg = Color(0xFF25272B),
    pillNeutralInk = Color(0xFFD1D5DB),
    pillAccentBg = Color(0xFF10283A),
    pillAccentInk = Color(0xFFB5DAF3),
    isDark = true,
)

val LocalLqColors = staticCompositionLocalOf { LightLqColors }
