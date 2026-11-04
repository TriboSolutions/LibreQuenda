package gal.tribo.librequenda.ui.components

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.selection.selectable
import androidx.compose.foundation.selection.selectableGroup
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.spring
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.remember
import androidx.compose.ui.graphics.graphicsLayer
import kotlinx.coroutines.delay
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.pluralStringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.clearAndSetSemantics
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import gal.tribo.librequenda.R
import gal.tribo.librequenda.data.RatingSummary
import gal.tribo.librequenda.ui.theme.Lq

/** An average, compact: ★ 4.3 (12); read aloud in full. Nothing until someone rated. */
@Composable
fun RatingBadge(rating: RatingSummary?, modifier: Modifier = Modifier) {
    if (rating == null || rating.count == 0) return
    val locale = LocalConfiguration.current.locales[0]
    val average = String.format(locale, "%.1f", rating.average)
    val spoken = pluralStringResource(R.plurals.rating_summary, rating.count, average, rating.count)
    Row(
        modifier.clearAndSetSemantics { contentDescription = spoken },
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(4.dp),
    ) {
        Icon(painterResource(R.drawable.ic_star_filled), contentDescription = null, tint = Lq.colors.star, modifier = Modifier.size(16.dp))
        Text(average, style = MaterialTheme.typography.bodyMedium, fontWeight = FontWeight.Medium, color = Lq.colors.foreground)
        Text("(${rating.count})", style = MaterialTheme.typography.bodyMedium, color = Lq.colors.mutedForeground)
    }
}

/** 1 to 5 stars to choose (0 = none yet): five radio buttons, drawn as stars, each 48dp to tap. */
@Composable
fun StarInput(label: String, value: Int, onChange: (Int) -> Unit, enabled: Boolean = true) {
    Column {
        Text(label, style = MaterialTheme.typography.titleSmall, color = Lq.colors.foreground)
        Row(Modifier.selectableGroup()) {
            for (stars in 1..5) {
                val description = pluralStringResource(R.plurals.rating_stars, stars, stars)
                Box(
                    Modifier.size(48.dp).clip(RoundedCornerShape(8.dp))
                        .selectable(selected = value == stars, enabled = enabled, role = Role.RadioButton) { onChange(stars) }
                        .semantics { contentDescription = description },
                    contentAlignment = Alignment.Center,
                ) {
                    val given = stars <= value
                    // given, each star pops in turn, from the first to the one tapped
                    val pop = remember { Animatable(1f) }
                    LaunchedEffect(given, value) {
                        if (given && enabled) {
                            delay((stars - 1) * 40L)
                            pop.snapTo(0.6f)
                            pop.animateTo(1f, spring(dampingRatio = 0.35f, stiffness = 500f))
                        }
                    }
                    Icon(
                        painterResource(if (given) R.drawable.ic_star_filled else R.drawable.ic_star),
                        contentDescription = null,
                        tint = if (given) Lq.colors.star else Lq.colors.input,
                        modifier = Modifier.size(30.dp).graphicsLayer { scaleX = pop.value; scaleY = pop.value },
                    )
                }
            }
        }
    }
}
