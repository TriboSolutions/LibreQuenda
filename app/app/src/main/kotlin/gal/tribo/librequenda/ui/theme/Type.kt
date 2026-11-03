package gal.tribo.librequenda.ui.theme

import androidx.compose.material3.Typography
import androidx.compose.ui.text.ExperimentalTextApi
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.Font
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontVariation
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.sp
import gal.tribo.librequenda.R

// DM Sans, the web's typeface (variable font, OFL; res/font/dm_sans.ttf)
@OptIn(ExperimentalTextApi::class)
private fun dmSans(weight: FontWeight) =
    Font(R.font.dm_sans, weight, variationSettings = FontVariation.Settings(FontVariation.weight(weight.weight)))

val DmSans = FontFamily(
    dmSans(FontWeight.Light),
    dmSans(FontWeight.Normal),
    dmSans(FontWeight.Medium),
    dmSans(FontWeight.SemiBold),
    dmSans(FontWeight.Bold),
)

private val base = Typography()

private fun TextStyle.dm(weight: FontWeight? = null) =
    copy(fontFamily = DmSans, fontWeight = weight ?: fontWeight, letterSpacing = 0.sp)

val Typography = Typography(
    displayLarge = base.displayLarge.dm(),
    displayMedium = base.displayMedium.dm(),
    displaySmall = base.displaySmall.dm(),
    // page titles: the web's text-2xl/3xl bold, tight
    headlineLarge = base.headlineLarge.dm(FontWeight.Bold).copy(fontSize = 28.sp, lineHeight = 34.sp, letterSpacing = (-0.4).sp),
    headlineMedium = base.headlineMedium.dm(FontWeight.Bold).copy(letterSpacing = (-0.3).sp),
    headlineSmall = base.headlineSmall.dm(FontWeight.SemiBold),
    titleLarge = base.titleLarge.dm(FontWeight.SemiBold).copy(fontSize = 18.sp, lineHeight = 24.sp),
    titleMedium = base.titleMedium.dm(FontWeight.SemiBold),
    titleSmall = base.titleSmall.dm(FontWeight.Medium),
    bodyLarge = base.bodyLarge.dm(),
    bodyMedium = base.bodyMedium.dm().copy(fontSize = 15.sp, lineHeight = 22.sp),
    bodySmall = base.bodySmall.dm().copy(fontSize = 13.sp, lineHeight = 18.sp),
    labelLarge = base.labelLarge.dm(FontWeight.Medium).copy(fontSize = 15.sp),
    labelMedium = base.labelMedium.dm(FontWeight.Medium),
    labelSmall = base.labelSmall.dm(FontWeight.Medium),
)
