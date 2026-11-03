package gal.tribo.librequenda.ui.welcome

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.safeDrawingPadding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.LiveRegionMode
import androidx.compose.ui.semantics.liveRegion
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import gal.tribo.librequenda.AuthError
import gal.tribo.librequenda.R
import gal.tribo.librequenda.ui.components.BrandLockup
import gal.tribo.librequenda.ui.components.LanguageMenuButton
import gal.tribo.librequenda.ui.components.PrimaryButton
import gal.tribo.librequenda.ui.components.SecondaryButton
import gal.tribo.librequenda.ui.theme.Lq

/** First screen while signed out: sign in, create an account, or just look around. */
@Composable
fun WelcomeScreen(
    busy: Boolean,
    error: AuthError?,
    onSignIn: () -> Unit,
    onRegister: () -> Unit,
    onExplore: () -> Unit,
) {
    Box(
        Modifier.fillMaxSize().background(Lq.colors.background).safeDrawingPadding(),
        contentAlignment = Alignment.Center,
    ) {
        // the language first: everything after it reads in it
        LanguageMenuButton(Modifier.align(Alignment.TopEnd).padding(8.dp))
        Column(
            Modifier
                .widthIn(max = 440.dp)
                .fillMaxWidth()
                .verticalScroll(rememberScrollState())
                .padding(horizontal = 24.dp, vertical = 32.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            BrandLockup(height = 64.dp)
            Spacer(Modifier.height(48.dp))
            Text(
                stringResource(R.string.welcome_title),
                style = MaterialTheme.typography.headlineLarge,
                color = Lq.colors.foreground,
                textAlign = TextAlign.Center,
            )
            Spacer(Modifier.height(12.dp))
            Text(
                stringResource(R.string.welcome_body),
                style = MaterialTheme.typography.bodyLarge,
                color = Lq.colors.mutedForeground,
                textAlign = TextAlign.Center,
            )
            Spacer(Modifier.height(40.dp))

            Column(Modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(12.dp)) {
                PrimaryButton(stringResource(R.string.sign_in), onSignIn, Modifier.fillMaxWidth(), enabled = !busy)
                SecondaryButton(stringResource(R.string.create_account), onRegister, Modifier.fillMaxWidth(), enabled = !busy)
            }

            Box(Modifier.height(56.dp).padding(top = 16.dp), contentAlignment = Alignment.Center) {
                when {
                    busy -> CircularProgressIndicator(Modifier.size(24.dp), strokeWidth = 2.dp)
                    error != null -> Text(
                        stringResource(if (error == AuthError.Cancelled) R.string.sign_in_cancelled else R.string.sign_in_error),
                        color = if (error == AuthError.Cancelled) Lq.colors.mutedForeground else Lq.colors.destructive,
                        style = MaterialTheme.typography.bodyMedium,
                        textAlign = TextAlign.Center,
                        modifier = Modifier.semantics { liveRegion = LiveRegionMode.Polite },
                    )
                }
            }

            TextButton(onClick = onExplore, enabled = !busy) {
                Text(stringResource(R.string.welcome_explore), color = Lq.colors.primary, style = MaterialTheme.typography.labelLarge)
            }
        }
    }
}
