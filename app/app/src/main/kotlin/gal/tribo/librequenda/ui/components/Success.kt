package gal.tribo.librequenda.ui.components

import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.semantics.liveRegion
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import gal.tribo.librequenda.ui.theme.Lq

/**
 * Done (in at the entrance, booked, joined): the page turns green, a circle
 * springs up and the tick draws itself, the phone gives a short buzz, and the
 * words; then [onDone] goes on (to the booking).
 */
@Composable
fun SuccessOverlay(message: String, detail: String? = null, onDone: () -> Unit) {
    val haptics = androidx.compose.ui.platform.LocalHapticFeedback.current
    val circle = remember { androidx.compose.animation.core.Animatable(0f) }
    val tick = remember { androidx.compose.animation.core.Animatable(0f) }
    val words = remember { androidx.compose.animation.core.Animatable(0f) }
    androidx.compose.runtime.LaunchedEffect(Unit) {
        haptics.performHapticFeedback(androidx.compose.ui.hapticfeedback.HapticFeedbackType.LongPress)
        circle.animateTo(1f, androidx.compose.animation.core.spring(dampingRatio = 0.45f, stiffness = 300f))
    }
    androidx.compose.runtime.LaunchedEffect(Unit) {
        kotlinx.coroutines.delay(180)
        tick.animateTo(1f, tween(380))
        words.animateTo(1f, tween(250))
        kotlinx.coroutines.delay(1100)
        onDone()
    }
    val green = Lq.colors.pillPositiveInk
    Box(
        Modifier.fillMaxSize().background(Lq.colors.pillPositiveBg).clickable(interactionSource = null, indication = null) {},
        contentAlignment = Alignment.Center,
    ) {
        Column(horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(28.dp), modifier = Modifier.padding(32.dp)) {
            androidx.compose.foundation.Canvas(
                Modifier.size(132.dp).graphicsLayer { scaleX = circle.value; scaleY = circle.value }
            ) {
                drawCircle(green)
                // the tick, drawn along its length as it goes
                val path = androidx.compose.ui.graphics.Path().apply {
                    moveTo(size.width * 0.28f, size.height * 0.52f)
                    lineTo(size.width * 0.44f, size.height * 0.68f)
                    lineTo(size.width * 0.73f, size.height * 0.36f)
                }
                val measure = androidx.compose.ui.graphics.PathMeasure().apply { setPath(path, false) }
                val drawn = androidx.compose.ui.graphics.Path()
                measure.getSegment(0f, measure.length * tick.value, drawn, true)
                drawPath(drawn, androidx.compose.ui.graphics.Color.White,
                    style = androidx.compose.ui.graphics.drawscope.Stroke(width = size.width * 0.08f, cap = androidx.compose.ui.graphics.StrokeCap.Round, join = androidx.compose.ui.graphics.StrokeJoin.Round))
            }
            Text(
                message,
                style = MaterialTheme.typography.headlineSmall,
                color = green,
                textAlign = androidx.compose.ui.text.style.TextAlign.Center,
                modifier = Modifier.graphicsLayer { alpha = words.value; translationY = (1f - words.value) * 24f }
                    .semantics { liveRegion = androidx.compose.ui.semantics.LiveRegionMode.Assertive },
            )
            if (detail != null) Text(
                detail,
                style = MaterialTheme.typography.bodyLarge,
                color = green,
                textAlign = androidx.compose.ui.text.style.TextAlign.Center,
                modifier = Modifier.graphicsLayer { alpha = words.value },
            )
        }
    }
}

