package gal.tribo.librequenda.ui.flash

import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.spring
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.runtime.withFrameNanos
import androidx.compose.ui.Alignment
import androidx.compose.ui.BiasAlignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.graphics.drawscope.scale
import androidx.compose.ui.graphics.drawscope.translate
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.hapticfeedback.HapticFeedbackType
import androidx.compose.ui.platform.LocalHapticFeedback
import androidx.compose.ui.platform.LocalSoftwareKeyboardController
import androidx.compose.ui.semantics.LiveRegionMode
import androidx.compose.ui.semantics.liveRegion
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import gal.tribo.librequenda.ui.theme.Lq
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlin.math.max
import kotlin.random.Random

/** Milliseconds the whole show lasts before going on; a tap skips it. */
private const val SHOW_MS = 2000L

/**
 * Booked on the spot and in: the flash booking's own celebration, unlike the
 * others' green tick. A quick flash, the flash tab's deep blue with a few speed
 * lines going past, the bolt landing with one shock wave and a buzz, and the
 * words shooting in. Then [onDone] goes on (to the bookings); a tap skips it.
 */
@Composable
fun FlashSuccessOverlay(message: String, detail: String? = null, onDone: () -> Unit) {
    val haptics = LocalHapticFeedback.current
    val keyboard = LocalSoftwareKeyboardController.current
    val deep = Lq.colors.primaryDeep
    val bright = Lq.colors.brandBlue
    // the clock of the whole show, in seconds since it started
    var time by remember { mutableFloatStateOf(0f) }
    val bolt = remember { Animatable(0f) } // the bolt's slam: 0 far and big, 1 in place
    val words = remember { Animatable(0f) }
    val streaks = remember { List(10) { Streak(Random.nextFloat(), Random.nextFloat(), 0.25f + Random.nextFloat() * 0.6f, 1.6f + Random.nextFloat() * 2.8f, 0.12f + Random.nextFloat() * 0.45f) } }

    LaunchedEffect(Unit) {
        val start = withFrameNanos { it }
        while (true) withFrameNanos { time = (it - start) / 1e9f }
    }
    LaunchedEffect(Unit) {
        keyboard?.hide() // the code was just typed: nothing over the show
        haptics.performHapticFeedback(HapticFeedbackType.LongPress)
        launch { bolt.animateTo(1f, spring(dampingRatio = 0.5f, stiffness = 500f)) }
        delay(260)
        words.animateTo(1f, spring(dampingRatio = 0.62f, stiffness = 420f))
        delay(SHOW_MS - 260)
        onDone()
    }

    Box(Modifier.fillMaxSize().clickable(interactionSource = null, indication = null) { onDone() }, contentAlignment = Alignment.Center) {
        Canvas(Modifier.fillMaxSize()) {
            val center = Offset(size.width / 2f, size.height * 0.4f)
            // the ground: the flash blue, a little lighter where the bolt lands
            drawRect(deep)
            drawCircle(Brush.radialGradient(listOf(bright.copy(alpha = 0.35f), Color.Transparent), center, size.minDimension * 0.6f), size.minDimension * 0.6f, center)

            // a few speed lines going past, quick at first and then calm
            val rush = 1f + 2f * max(0f, 1f - time / 0.7f)
            streaks.forEach { s ->
                val x = ((s.x - time * s.speed * rush * 0.5f) % 1f + 1f) % 1f * (size.width + size.width * s.length) - size.width * s.length
                drawLine(Color.White.copy(alpha = s.alpha * 0.6f), Offset(x, s.y * size.height), Offset(x + size.width * s.length * 0.4f, s.y * size.height), strokeWidth = 2.dp.toPx(), cap = StrokeCap.Round)
            }

            // one shock wave as the bolt lands
            val p = ((time - 0.12f) / 0.7f).coerceIn(0f, 1f)
            if (p > 0f && p < 1f) drawCircle(Color.White.copy(alpha = (1f - p) * 0.5f), size.minDimension * 0.2f + p * size.minDimension * 0.5f, center, style = Stroke((1f - p) * 6.dp.toPx()))

            // the bolt: lands from a little bigger, in white on a soft glow
            val b = bolt.value
            val boltSize = size.minDimension * 0.36f
            translate(center.x - boltSize / 2f, center.y - boltSize / 2f) {
                val s = 1.8f - 0.8f * b
                scale(s, s, pivot = Offset(boltSize / 2f, boltSize / 2f)) {
                    drawCircle(bright.copy(alpha = 0.5f * b.coerceIn(0f, 1f)), boltSize * 0.6f, Offset(boltSize / 2f, boltSize / 2f))
                    drawPath(zapPath(boltSize), Color.White.copy(alpha = b.coerceIn(0f, 1f)))
                }
            }

            // a single soft flash at the start
            if (time < 0.18f) drawRect(Color.White.copy(alpha = 0.7f * (1f - time / 0.18f)))
        }

        // the words, shooting in from the left and leaning forward
        Column(
            Modifier.align(BiasAlignment(0f, 0.42f)).fillMaxWidth().padding(horizontal = 32.dp)
                .graphicsLayer {
                    alpha = words.value.coerceIn(0f, 1f)
                    translationX = (1f - words.value) * -size.width
                },
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            Text(
                message,
                style = MaterialTheme.typography.headlineMedium.copy(fontWeight = FontWeight.ExtraBold, fontStyle = FontStyle.Italic),
                color = Color.White,
                textAlign = TextAlign.Center,
                modifier = Modifier.semantics { liveRegion = LiveRegionMode.Assertive },
            )
            if (detail != null) Text(detail, style = MaterialTheme.typography.bodyLarge, color = Color.White.copy(alpha = 0.85f), textAlign = TextAlign.Center)
        }
    }
}

private class Streak(val x: Float, val y: Float, val length: Float, val speed: Float, val alpha: Float)

/** Lucide's «zap», the flash booking's mark, filling a [side]-wide square. */
private fun zapPath(side: Float): Path {
    val k = side / 24f
    return Path().apply {
        moveTo(13f * k, 2f * k)
        lineTo(3f * k, 14f * k)
        lineTo(12f * k, 14f * k)
        lineTo(11f * k, 22f * k)
        lineTo(21f * k, 10f * k)
        lineTo(12f * k, 10f * k)
        close()
    }
}
