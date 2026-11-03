package gal.tribo.librequenda.data

import android.Manifest
import android.content.Context
import android.content.pm.PackageManager
import android.location.Location
import android.location.LocationManager
import android.os.Build
import androidx.core.content.ContextCompat
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlinx.coroutines.withTimeoutOrNull
import kotlin.coroutines.resume

/** Where the phone roughly is (network or last fix), or null without permission or a fix. Never stored. */
suspend fun currentLocation(context: Context): GeoPoint? {
    if (ContextCompat.checkSelfPermission(context, Manifest.permission.ACCESS_COARSE_LOCATION) != PackageManager.PERMISSION_GRANTED) return null
    val manager = context.getSystemService(LocationManager::class.java) ?: return null
    val providers = manager.getProviders(true)
    val last = providers.mapNotNull { runCatching { manager.getLastKnownLocation(it) }.getOrNull() }.maxByOrNull { it.time }
    val location = last ?: if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
        // the passive provider never asks for a fix itself
        val preferred = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) listOf(LocationManager.FUSED_PROVIDER) else emptyList()
        val provider = (preferred + listOf(LocationManager.NETWORK_PROVIDER, LocationManager.GPS_PROVIDER))
            .firstOrNull { it in providers } ?: return null
        withTimeoutOrNull(10_000) {
            suspendCancellableCoroutine<Location?> { cont ->
                manager.getCurrentLocation(provider, null, context.mainExecutor) { cont.resume(it) }
            }
        }
    } else null
    return location?.let { GeoPoint(it.latitude, it.longitude) }
}
