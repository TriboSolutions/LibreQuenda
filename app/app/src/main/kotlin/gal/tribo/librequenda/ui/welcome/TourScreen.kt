package gal.tribo.librequenda.ui.welcome

import androidx.activity.compose.BackHandler
import androidx.annotation.DrawableRes
import androidx.annotation.StringRes
import androidx.compose.animation.animateColorAsState
import androidx.compose.animation.core.animateDpAsState
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.safeDrawingPadding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.pager.HorizontalPager
import androidx.compose.foundation.pager.rememberPagerState
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.clearAndSetSemantics
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import gal.tribo.librequenda.R
import gal.tribo.librequenda.ui.components.PrimaryButton
import gal.tribo.librequenda.ui.theme.Lq
import kotlinx.coroutines.launch

/** One step of the tour: what part of the app it shows, and what you can do there. */
private data class TourStep(@DrawableRes val icon: Int, @StringRes val title: Int, @StringRes val body: Int)

private val Steps = listOf(
    TourStep(R.drawable.ic_search, R.string.tour_find_title, R.string.tour_find_body),
    TourStep(R.drawable.ic_calendar_plus, R.string.tour_book_title, R.string.tour_book_body),
    TourStep(R.drawable.ic_users, R.string.tour_games_title, R.string.tour_games_body),
    TourStep(R.drawable.ic_zap, R.string.tour_flash_title, R.string.tour_flash_body),
    TourStep(R.drawable.ic_qr_code, R.string.tour_enter_title, R.string.tour_enter_body),
    TourStep(R.drawable.ic_bell, R.string.tour_bookings_title, R.string.tour_bookings_body),
)

/**
 * The first thing a fresh install shows, signed in or not: a few swipeable steps
 * through what the app does. «Skip» or the last step's button ends it; the menu
 * can bring it back.
 */
@Composable
fun TourScreen(onDone: () -> Unit) {
    val pager = rememberPagerState { Steps.size }
    val scope = rememberCoroutineScope()
    val last = pager.currentPage == Steps.lastIndex
    BackHandler {
        if (pager.currentPage > 0) scope.launch { pager.animateScrollToPage(pager.currentPage - 1) } else onDone()
    }

    Column(Modifier.fillMaxSize().background(Lq.colors.background).safeDrawingPadding()) {
        // the language was picked on the welcome screen; here only «Skip», kept in place on the last step
        Box(Modifier.fillMaxWidth().height(64.dp).padding(horizontal = 8.dp), contentAlignment = Alignment.CenterEnd) {
            if (!last) {
                TextButton(onClick = onDone) {
                    Text(stringResource(R.string.tour_skip), color = Lq.colors.mutedForeground, style = MaterialTheme.typography.labelLarge)
                }
            }
        }

        HorizontalPager(pager, Modifier.weight(1f).fillMaxWidth()) { page ->
            StepPage(Steps[page])
        }

        val position = stringResource(R.string.tour_position, pager.currentPage + 1, Steps.size)
        Row(
            Modifier.fillMaxWidth().padding(vertical = 16.dp).clearAndSetSemantics { contentDescription = position },
            horizontalArrangement = Arrangement.spacedBy(8.dp, Alignment.CenterHorizontally),
        ) {
            Steps.indices.forEach { Dot(selected = it == pager.currentPage) }
        }

        Box(Modifier.fillMaxWidth().padding(start = 24.dp, end = 24.dp, bottom = 24.dp), contentAlignment = Alignment.Center) {
            PrimaryButton(
                stringResource(if (last) R.string.tour_start else R.string.tour_next),
                onClick = { if (last) onDone() else scope.launch { pager.animateScrollToPage(pager.currentPage + 1) } },
                modifier = Modifier.widthIn(max = 440.dp).fillMaxWidth(),
            )
        }
    }
}

@Composable
private fun StepPage(step: TourStep) {
    Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
        Column(
            Modifier
                .widthIn(max = 440.dp)
                .fillMaxWidth()
                .verticalScroll(rememberScrollState())
                .padding(horizontal = 32.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            // the logo's blue, as the app's round buttons
            Box(Modifier.size(128.dp).clip(CircleShape).background(Lq.colors.pillAccentBg), contentAlignment = Alignment.Center) {
                Box(Modifier.size(80.dp).clip(CircleShape).background(Lq.colors.brandBlue), contentAlignment = Alignment.Center) {
                    Icon(painterResource(step.icon), contentDescription = null, tint = Lq.colors.primaryForeground, modifier = Modifier.size(40.dp))
                }
            }
            Spacer(Modifier.height(40.dp))
            Text(
                stringResource(step.title),
                style = MaterialTheme.typography.headlineMedium,
                color = Lq.colors.foreground,
                textAlign = TextAlign.Center,
                modifier = Modifier.semantics { heading() },
            )
            Spacer(Modifier.height(12.dp))
            Text(
                stringResource(step.body),
                style = MaterialTheme.typography.bodyLarge,
                color = Lq.colors.mutedForeground,
                textAlign = TextAlign.Center,
            )
        }
    }
}

@Composable
private fun Dot(selected: Boolean) {
    val width by animateDpAsState(if (selected) 24.dp else 8.dp, label = "dot")
    val color by animateColorAsState(if (selected) Lq.colors.primary else Lq.colors.border, label = "dot")
    Box(Modifier.height(8.dp).width(width).clip(RoundedCornerShape(4.dp)).background(color))
}
