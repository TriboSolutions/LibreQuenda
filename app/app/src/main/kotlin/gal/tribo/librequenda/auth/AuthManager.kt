package gal.tribo.librequenda.auth

import android.content.Context
import android.content.Intent
import android.net.Uri
import androidx.core.content.edit
import gal.tribo.librequenda.BuildConfig
import gal.tribo.librequenda.data.RuntimeConfig
import kotlinx.coroutines.suspendCancellableCoroutine
import net.openid.appauth.AppAuthConfiguration
import net.openid.appauth.AuthState
import net.openid.appauth.AuthorizationException
import net.openid.appauth.AuthorizationRequest
import net.openid.appauth.AuthorizationResponse
import net.openid.appauth.AuthorizationService
import net.openid.appauth.AuthorizationServiceConfiguration
import net.openid.appauth.EndSessionRequest
import net.openid.appauth.ResponseTypeValues
import net.openid.appauth.connectivity.ConnectionBuilder
import net.openid.appauth.connectivity.DefaultConnectionBuilder
import java.net.HttpURLConnection
import java.net.URL
import kotlin.coroutines.resume
import kotlin.coroutines.resumeWithException

/**
 * Sign-in with Zitadel, as the web does it: authorization code with PKCE in the
 * browser (a Custom Tab, so the person sees the real sign-in page and its address),
 * then short access tokens renewed with the refresh token (offline_access).
 * The tokens stay in the app's private storage, never in a URL or the logs.
 */
class AuthManager(context: Context) {
    private val appContext = context.applicationContext
    private val prefs = context.getSharedPreferences("auth", Context.MODE_PRIVATE)
    private val service = AuthorizationService(context, appAuthConfiguration())
    private var state: AuthState = prefs.getString(KEY_STATE, null)
        ?.let { runCatching { AuthState.jsonDeserialize(it) }.getOrNull() } ?: AuthState()

    val isSignedIn: Boolean get() = state.isAuthorized

    /** The intent that opens Zitadel's sign-in (or its sign-up form, with [register]). */
    suspend fun signInIntent(register: Boolean): Intent {
        val config = RuntimeConfig.load()
        val serviceConfig = discover(config.authority)
        val request = AuthorizationRequest.Builder(serviceConfig, config.clientId, ResponseTypeValues.CODE, REDIRECT_URI)
            .setScope(config.scope)
            // Zitadel's pages in the app's language
            .setUiLocales(appContext.resources.configuration.locales[0].language)
            .apply { if (register) setPrompt("create") }
            .build()
        return service.getAuthorizationRequestIntent(request)
    }

    /** Exchanges the code Zitadel sent back for tokens. */
    suspend fun completeSignIn(data: Intent) {
        val response = AuthorizationResponse.fromIntent(data)
        val error = AuthorizationException.fromIntent(data)
        state = AuthState(response, error)
        if (response == null) throw error ?: IllegalStateException("No authorization response")
        suspendCancellableCoroutine { cont ->
            service.performTokenRequest(response.createTokenExchangeRequest()) { tokens, ex ->
                state.update(tokens, ex)
                save()
                if (tokens != null) cont.resume(Unit) else cont.resumeWithException(ex ?: IllegalStateException("No tokens"))
            }
        }
    }

    /** A valid access token, renewed first when it is about to expire; null when signed out. */
    suspend fun accessToken(): String? {
        if (!state.isAuthorized) return null
        return suspendCancellableCoroutine { cont ->
            state.performActionWithFreshTokens(service) { token, _, ex ->
                save()
                when {
                    token != null -> cont.resume(token)
                    // the refresh token was revoked or expired: sign in again
                    ex?.type == AuthorizationException.TYPE_OAUTH_TOKEN_ERROR -> { clear(); cont.resume(null) }
                    else -> cont.resumeWithException(ex ?: IllegalStateException("No access token"))
                }
            }
        }
    }

    /**
     * Signs out here, and returns the intent that ends the session at Zitadel too
     * (otherwise the next sign-in would go through without asking), or null when
     * there is no session to end.
     */
    fun signOut(): Intent? {
        val config = state.authorizationServiceConfiguration
        val idToken = state.idToken
        clear()
        if (config?.endSessionEndpoint == null || idToken == null) return null
        val request = EndSessionRequest.Builder(config)
            .setIdTokenHint(idToken)
            .setPostLogoutRedirectUri(REDIRECT_URI)
            .build()
        return service.getEndSessionRequestIntent(request)
    }

    fun dispose() = service.dispose()

    /** Forgets the session here only (the account is gone: there is nothing to end at Zitadel). */
    fun clear() {
        state = AuthState()
        prefs.edit { remove(KEY_STATE) }
    }

    private fun save() = prefs.edit { putString(KEY_STATE, state.jsonSerializeString()) }

    private suspend fun discover(authority: String): AuthorizationServiceConfiguration =
        suspendCancellableCoroutine { cont ->
            AuthorizationServiceConfiguration.fetchFromIssuer(Uri.parse(authority), { config, ex ->
                if (config != null) cont.resume(config) else cont.resumeWithException(ex ?: IllegalStateException("Discovery failed"))
            }, connectionBuilder())
        }

    companion object {
        private const val KEY_STATE = "state"
        val REDIRECT_URI: Uri = Uri.parse("gal.tribo.librequenda:/oauth2redirect")

        // The development stack serves Zitadel over plain HTTP (http://auth.localhost);
        // only debug builds accept it, release builds require HTTPS.
        private fun connectionBuilder(): ConnectionBuilder =
            if (BuildConfig.DEBUG) PlainHttpConnectionBuilder else DefaultConnectionBuilder.INSTANCE

        private fun appAuthConfiguration() = AppAuthConfiguration.Builder()
            .setConnectionBuilder(connectionBuilder())
            .setSkipIssuerHttpsCheck(BuildConfig.DEBUG)
            .build()
    }
}

private object PlainHttpConnectionBuilder : ConnectionBuilder {
    override fun openConnection(uri: Uri): HttpURLConnection =
        (URL(uri.toString()).openConnection() as HttpURLConnection).apply {
            connectTimeout = 15_000
            readTimeout = 10_000
            instanceFollowRedirects = false
        }
}
